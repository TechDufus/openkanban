package agent

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/techdufus/openkanban/internal/config"
)

type OpencodeServer struct {
	config  *config.Config
	cmd     *exec.Cmd
	port    int
	running bool
	mu      sync.RWMutex
}

func NewOpencodeServer(cfg *config.Config) *OpencodeServer {
	return &OpencodeServer{
		config: cfg,
		port:   cfg.Opencode.ServerPort,
	}
}

func (s *OpencodeServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	if !s.config.Opencode.ServerEnabled {
		return nil
	}

	// Check if opencode binary exists
	if _, err := exec.LookPath("opencode"); err != nil {
		return nil // opencode not installed, skip gracefully
	}

	if s.isServerAlreadyRunning() {
		s.running = true
		return nil
	}

	s.cmd = exec.Command("opencode", "serve", "--port", fmt.Sprintf("%d", s.port))
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode server: %w", err)
	}

	timeout := s.config.Opencode.StartupTimeout
	if timeout <= 0 {
		timeout = 10 // default fallback
	}
	if err := s.waitForReady(time.Duration(timeout) * time.Second); err != nil {
		s.cmd.Process.Kill()
		s.cmd = nil
		return err
	}

	s.running = true
	return nil
}

func (s *OpencodeServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.cmd == nil {
		return nil
	}

	if s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}

	s.cmd = nil
	s.running = false
	return nil
}

func (s *OpencodeServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *OpencodeServer) Port() int {
	return s.port
}

func (s *OpencodeServer) URL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

func (s *OpencodeServer) isServerAlreadyRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	url := fmt.Sprintf("http://localhost:%d/session/status", s.port)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (s *OpencodeServer) waitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://localhost:%d/session/status", s.port)

	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			cancel()
			time.Sleep(100 * time.Millisecond)
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		cancel()
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("opencode server failed to become ready within %v", timeout)
}
