package daemon

import (
	"testing"
)

func TestDaemonClient(t *testing.T) {
	t.Log("Testing daemon client...")

	available := DaemonAvailable("")
	t.Logf("Daemon available: %v", available)

	if !available {
		t.Skip("Daemon not running, skipping live test")
		return
	}

	client := NewClient("")
	t.Logf("Socket: %s", client.SocketPath())

	if err := client.Connect(); err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	t.Log("Connected!")

	if !client.IsConnected() {
		t.Fatal("Expected IsConnected to return true")
	}

	if err := client.Resize(24, 80); err != nil {
		t.Errorf("Resize error: %v", err)
	} else {
		t.Log("Resize: 24x80")
	}

	_, err := client.Write([]byte("echo 'CLIENT_LIB_TEST'\n"))
	if err != nil {
		t.Errorf("Write error: %v", err)
	} else {
		t.Log("Command sent!")
	}

	for i := 0; i < 3; i++ {
		msgType, data, err := client.ReadMessage()
		if err != nil {
			t.Logf("Read error (may be expected): %v", err)
			break
		}
		preview := string(data)
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		t.Logf("Msg[%d]: %q", msgType, preview)
	}

	if err := client.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}
	t.Log("Done!")
}
