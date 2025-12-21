package project

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/techdufus/openkanban/internal/board"
	"github.com/techdufus/openkanban/internal/config"
)

type SavedFilter struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ProjectIDs []string `json:"project_ids,omitempty"`
	Statuses   []string `json:"statuses,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	IsDefault  bool     `json:"is_default"`
}

func NewFilter(name string) *SavedFilter {
	return &SavedFilter{
		ID:   uuid.New().String(),
		Name: name,
	}
}

func (f *SavedFilter) Matches(ticket *board.Ticket) bool {
	if len(f.ProjectIDs) > 0 {
		found := false
		for _, pid := range f.ProjectIDs {
			if pid == ticket.ProjectID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(f.Statuses) > 0 {
		found := false
		for _, s := range f.Statuses {
			if s == string(ticket.Status) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(f.Labels) > 0 {
		found := false
		for _, filterLabel := range f.Labels {
			for _, ticketLabel := range ticket.Labels {
				if filterLabel == ticketLabel {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

type FilterRegistry struct {
	Filters map[string]*SavedFilter `json:"filters"`
}

func newFilterRegistry() *FilterRegistry {
	return &FilterRegistry{
		Filters: make(map[string]*SavedFilter),
	}
}

func filterRegistryPath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "filters.json"), nil
}

func LoadFilterRegistry() (*FilterRegistry, error) {
	path, err := filterRegistryPath()
	if err != nil {
		return newFilterRegistry(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newFilterRegistry(), nil
		}
		return nil, err
	}

	var reg FilterRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}

	if reg.Filters == nil {
		reg.Filters = make(map[string]*SavedFilter)
	}

	return &reg, nil
}

func (r *FilterRegistry) Save() error {
	path, err := filterRegistryPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (r *FilterRegistry) Add(f *SavedFilter) error {
	r.Filters[f.ID] = f
	return r.Save()
}

func (r *FilterRegistry) Get(id string) *SavedFilter {
	return r.Filters[id]
}

func (r *FilterRegistry) GetDefault() *SavedFilter {
	for _, f := range r.Filters {
		if f.IsDefault {
			return f
		}
	}
	return nil
}

func (r *FilterRegistry) Delete(id string) error {
	delete(r.Filters, id)
	return r.Save()
}

func (r *FilterRegistry) List() []*SavedFilter {
	result := make([]*SavedFilter, 0, len(r.Filters))
	for _, f := range r.Filters {
		result = append(result, f)
	}
	return result
}
