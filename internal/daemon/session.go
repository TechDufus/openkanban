package daemon

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

type Session struct {
	ID      string
	cmd     *exec.Cmd
	ptmx    *os.File
	mu      sync.Mutex
	running bool
	workdir string
}

func NewSession(id, command string, args ...string) (*Session, error) {
	s := &Session{ID: id}

	s.cmd = exec.Command(command, args...)
	s.cmd.Env = buildCleanEnv()

	log.Printf("NewSession: id=%s cmd=%s args=%v", id, command, args)
	return s, nil
}

func (s *Session) SetWorkdir(dir string) {
	s.workdir = dir
	if s.cmd != nil {
		s.cmd.Dir = dir
	}
}

func (s *Session) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("session already running")
	}

	log.Printf("Session.Start: id=%s workdir=%s cmd=%s", s.ID, s.workdir, s.cmd.Path)

	ptmx, err := pty.Start(s.cmd)
	if err != nil {
		log.Printf("Session.Start: pty.Start failed: %v", err)
		return fmt.Errorf("pty.Start: %w", err)
	}

	s.ptmx = ptmx
	s.running = true
	log.Printf("Session.Start: started successfully, pid=%d", s.cmd.Process.Pid)
	return nil
}

func (s *Session) Read(buf []byte) (int, error) {
	return s.ptmx.Read(buf)
}

func (s *Session) Write(data []byte) (int, error) {
	return s.ptmx.Write(data)
}

func (s *Session) Resize(rows, cols uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ptmx == nil {
		return fmt.Errorf("session not running")
	}

	return pty.Setsize(s.ptmx, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

func (s *Session) Wait() error {
	err := s.cmd.Wait()
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
	if err != nil {
		log.Printf("Session.Wait: id=%s exited with error: %v", s.ID, err)
	} else {
		log.Printf("Session.Wait: id=%s exited cleanly", s.ID)
	}
	return err
}

func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	if s.ptmx != nil {
		s.ptmx.Close()
	}
	s.running = false
	return nil
}

func (s *Session) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Session) PTY() *os.File {
	return s.ptmx
}

func buildCleanEnv() []string {
	env := os.Environ()
	return append(env, "TERM=xterm-256color")
}
