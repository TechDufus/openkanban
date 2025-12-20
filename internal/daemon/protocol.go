// Package daemon implements the OpenKanban daemon server/client protocol.
// Wire format: [type:1][length:4][data:length]
package daemon

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	MsgData   byte = 0x01 // PTY I/O (bidirectional)
	MsgResize byte = 0x02 // Window size (client -> server)
	MsgExit   byte = 0x03 // Session terminated (server -> client)

	MsgAttach       byte = 0x10 // Attach to session: [sessionID]
	MsgCreate       byte = 0x11 // Create session: [sessionID][workdir][command...]
	MsgSessionOK    byte = 0x12 // Session ready (server -> client)
	MsgSessionError byte = 0x13 // Session error: [error message]
	MsgDetach       byte = 0x14 // Detach from current session
	MsgList         byte = 0x15 // List sessions (client -> server)
	MsgListResponse byte = 0x16 // List response: [sessionID\0sessionID\0...]
)

type Message struct {
	Type byte
	Data []byte
}

func WriteMessage(w io.Writer, msg Message) error {
	if _, err := w.Write([]byte{msg.Type}); err != nil {
		return fmt.Errorf("write type: %w", err)
	}

	length := uint32(len(msg.Data))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return fmt.Errorf("write length: %w", err)
	}

	if len(msg.Data) > 0 {
		if _, err := w.Write(msg.Data); err != nil {
			return fmt.Errorf("write data: %w", err)
		}
	}

	return nil
}

func ReadMessage(r io.Reader) (Message, error) {
	var msg Message

	typeBuf := make([]byte, 1)
	if _, err := io.ReadFull(r, typeBuf); err != nil {
		return msg, fmt.Errorf("read type: %w", err)
	}
	msg.Type = typeBuf[0]

	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return msg, fmt.Errorf("read length: %w", err)
	}

	if length > 1<<20 {
		return msg, fmt.Errorf("message too large: %d bytes", length)
	}

	if length > 0 {
		msg.Data = make([]byte, length)
		if _, err := io.ReadFull(r, msg.Data); err != nil {
			return msg, fmt.Errorf("read data: %w", err)
		}
	}

	return msg, nil
}

func EncodeResize(rows, cols uint16) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint16(buf[0:2], rows)
	binary.BigEndian.PutUint16(buf[2:4], cols)
	return buf
}

func DecodeResize(data []byte) (rows, cols uint16, err error) {
	if len(data) < 4 {
		return 0, 0, fmt.Errorf("resize payload too short: %d bytes", len(data))
	}
	rows = binary.BigEndian.Uint16(data[0:2])
	cols = binary.BigEndian.Uint16(data[2:4])
	return rows, cols, nil
}

func EncodeAttach(sessionID string) []byte {
	return []byte(sessionID)
}

func DecodeAttach(data []byte) string {
	return string(data)
}

func EncodeCreate(sessionID, workdir, command string, args []string) []byte {
	parts := []string{sessionID, workdir, command}
	parts = append(parts, args...)

	var buf []byte
	for i, part := range parts {
		if i > 0 {
			buf = append(buf, 0)
		}
		buf = append(buf, []byte(part)...)
	}
	return buf
}

func DecodeCreate(data []byte) (sessionID, workdir, command string, args []string) {
	parts := splitNull(data)
	if len(parts) >= 1 {
		sessionID = parts[0]
	}
	if len(parts) >= 2 {
		workdir = parts[1]
	}
	if len(parts) >= 3 {
		command = parts[2]
	}
	if len(parts) > 3 {
		args = parts[3:]
	}
	return
}

func splitNull(data []byte) []string {
	var parts []string
	start := 0
	for i, b := range data {
		if b == 0 {
			parts = append(parts, string(data[start:i]))
			start = i + 1
		}
	}
	if start < len(data) {
		parts = append(parts, string(data[start:]))
	}
	return parts
}
