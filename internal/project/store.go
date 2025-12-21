package project

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/techdufus/openkanban/internal/config"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrDuplicatePath   = errors.New("project with this repository path already exists")
)

type ProjectRegistry struct {
	Projects map[string]*Project `json:"projects"`
}

func newRegistry() *ProjectRegistry {
	return &ProjectRegistry{
		Projects: make(map[string]*Project),
	}
}

func registryPath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "projects.json"), nil
}

func LoadRegistry() (*ProjectRegistry, error) {
	path, err := registryPath()
	if err != nil {
		return newRegistry(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newRegistry(), nil
		}
		return nil, err
	}

	var reg ProjectRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}

	if reg.Projects == nil {
		reg.Projects = make(map[string]*Project)
	}

	return &reg, nil
}

func (r *ProjectRegistry) Save() error {
	path, err := registryPath()
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

func (r *ProjectRegistry) Add(p *Project) error {
	for _, existing := range r.Projects {
		if existing.RepoPath == p.RepoPath {
			return ErrDuplicatePath
		}
	}
	r.Projects[p.ID] = p
	return r.Save()
}

func (r *ProjectRegistry) Get(id string) (*Project, error) {
	p, ok := r.Projects[id]
	if !ok {
		return nil, ErrProjectNotFound
	}
	return p, nil
}

func (r *ProjectRegistry) FindByPath(repoPath string) (*Project, error) {
	for _, p := range r.Projects {
		if p.RepoPath == repoPath {
			return p, nil
		}
	}
	return nil, ErrProjectNotFound
}

func (r *ProjectRegistry) Update(p *Project) error {
	if _, ok := r.Projects[p.ID]; !ok {
		return ErrProjectNotFound
	}
	p.Touch()
	r.Projects[p.ID] = p
	return r.Save()
}

func (r *ProjectRegistry) Delete(id string) error {
	if _, ok := r.Projects[id]; !ok {
		return ErrProjectNotFound
	}
	delete(r.Projects, id)
	return r.Save()
}

func (r *ProjectRegistry) List() []*Project {
	result := make([]*Project, 0, len(r.Projects))
	for _, p := range r.Projects {
		result = append(result, p)
	}
	return result
}
