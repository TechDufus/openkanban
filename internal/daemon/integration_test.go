//go:build integration

package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMultiSessionIntegration(t *testing.T) {
	sockPath := filepath.Join(os.Getenv("HOME"), ".openkanban", "daemon.sock")

	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		t.Skip("daemon not running - start with: go run ./cmd/okd")
	}

	c1 := NewClient(sockPath)
	if err := c1.Connect(); err != nil {
		t.Fatalf("client1 connect failed: %v", err)
	}
	defer c1.Close()

	if err := c1.CreateSession("test-session-1", "/tmp", "bash", []string{"-c", "echo 'session1 started'; sleep 5"}); err != nil {
		t.Fatalf("create session1 failed: %v", err)
	}
	t.Log("session1 created")

	c2 := NewClient(sockPath)
	if err := c2.Connect(); err != nil {
		t.Fatalf("client2 connect failed: %v", err)
	}
	defer c2.Close()

	if err := c2.CreateSession("test-session-2", "/tmp", "bash", []string{"-c", "echo 'session2 started'; sleep 5"}); err != nil {
		t.Fatalf("create session2 failed: %v", err)
	}
	t.Log("session2 created")

	c3 := NewClient(sockPath)
	if err := c3.Connect(); err != nil {
		t.Fatalf("client3 connect failed: %v", err)
	}
	defer c3.Close()

	if err := c3.CreateSession("test-session-1", "/tmp", "bash", nil); err != nil {
		t.Fatalf("attach session1 failed: %v", err)
	}
	t.Log("client3 attached to session1")

	time.Sleep(500 * time.Millisecond)

	msgType, data, err := c1.ReadMessage()
	if err == nil {
		t.Logf("session1 received: type=%d len=%d", msgType, len(data))
	}

	msgType, data, err = c2.ReadMessage()
	if err == nil {
		t.Logf("session2 received: type=%d len=%d", msgType, len(data))
	}

	t.Log("multi-session test passed!")
}
