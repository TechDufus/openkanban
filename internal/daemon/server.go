package daemon

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
)

type clientConn struct {
	conn      net.Conn
	sessionID string
}

type Server struct {
	socketPath string
	listener   net.Listener
	sessions   map[string]*Session
	clients    map[net.Conn]*clientConn
	mu         sync.RWMutex
	done       chan struct{}
}

func NewServer(socketPath string) *Server {
	return &Server{
		socketPath: socketPath,
		sessions:   make(map[string]*Session),
		clients:    make(map[net.Conn]*clientConn),
		done:       make(chan struct{}),
	}
}

func (s *Server) SocketPath() string {
	return s.socketPath
}

func (s *Server) Start() error {
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0700); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}

	os.Remove(s.socketPath)

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = listener

	return nil
}

func (s *Server) Accept() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil
			default:
				return fmt.Errorf("accept: %w", err)
			}
		}

		s.mu.Lock()
		s.clients[conn] = &clientConn{conn: conn}
		s.mu.Unlock()

		log.Printf("client connected: %s", conn.RemoteAddr())
		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		s.removeClient(conn)
		conn.Close()
		log.Printf("client disconnected")
	}()

	for {
		msg, err := ReadMessage(conn)
		if err != nil {
			return
		}

		switch msg.Type {
		case MsgCreate:
			s.handleCreate(conn, msg.Data)

		case MsgAttach:
			s.handleAttach(conn, msg.Data)

		case MsgDetach:
			s.handleDetach(conn)

		case MsgData:
			s.handleData(conn, msg.Data)

		case MsgResize:
			s.handleResize(conn, msg.Data)

		case MsgList:
			s.handleList(conn)
		}
	}
}

func (s *Server) handleCreate(conn net.Conn, data []byte) {
	sessionID, workdir, command, args := DecodeCreate(data)
	if sessionID == "" || command == "" {
		s.sendError(conn, "invalid create request")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[sessionID]; exists {
		client := s.clients[conn]
		if client != nil {
			client.sessionID = sessionID
		}
		WriteMessage(conn, Message{Type: MsgSessionOK})
		log.Printf("client attached to existing session: %s", sessionID)
		return
	}

	session, err := NewSession(sessionID, command, args...)
	if err != nil {
		s.sendError(conn, err.Error())
		return
	}

	if workdir != "" {
		session.SetWorkdir(workdir)
	}

	if err := session.Start(); err != nil {
		s.sendError(conn, err.Error())
		return
	}

	s.sessions[sessionID] = session

	client := s.clients[conn]
	if client != nil {
		client.sessionID = sessionID
	}

	go s.sessionReadLoop(session)
	go s.sessionWaitLoop(session)

	WriteMessage(conn, Message{Type: MsgSessionOK})
	log.Printf("created session: %s (cmd: %s)", sessionID, command)
}

func (s *Server) handleAttach(conn net.Conn, data []byte) {
	sessionID := DecodeAttach(data)
	if sessionID == "" {
		s.sendError(conn, "empty session ID")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		s.sendError(conn, "session not found: "+sessionID)
		return
	}

	if !session.Running() {
		s.sendError(conn, "session not running: "+sessionID)
		return
	}

	client := s.clients[conn]
	if client != nil {
		client.sessionID = sessionID
	}

	WriteMessage(conn, Message{Type: MsgSessionOK})
	log.Printf("client attached to session: %s", sessionID)
}

func (s *Server) handleDetach(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	client := s.clients[conn]
	if client != nil {
		log.Printf("client detached from session: %s", client.sessionID)
		client.sessionID = ""
	}
}

func (s *Server) handleData(conn net.Conn, data []byte) {
	s.mu.RLock()
	client := s.clients[conn]
	if client == nil || client.sessionID == "" {
		s.mu.RUnlock()
		return
	}
	session := s.sessions[client.sessionID]
	s.mu.RUnlock()

	if session != nil && session.Running() {
		session.Write(data)
	}
}

func (s *Server) handleResize(conn net.Conn, data []byte) {
	rows, cols, err := DecodeResize(data)
	if err != nil {
		return
	}

	s.mu.RLock()
	client := s.clients[conn]
	if client == nil || client.sessionID == "" {
		s.mu.RUnlock()
		return
	}
	session := s.sessions[client.sessionID]
	s.mu.RUnlock()

	if session != nil {
		session.Resize(rows, cols)
	}
}

func (s *Server) handleList(conn net.Conn) {
	s.mu.RLock()
	ids := make([]string, 0, len(s.sessions))
	for id, session := range s.sessions {
		if session.Running() {
			ids = append(ids, id)
		}
	}
	s.mu.RUnlock()

	var data []byte
	for i, id := range ids {
		if i > 0 {
			data = append(data, 0)
		}
		data = append(data, []byte(id)...)
	}
	WriteMessage(conn, Message{Type: MsgListResponse, Data: data})
}

func (s *Server) sessionReadLoop(session *Session) {
	buf := make([]byte, 32*1024)
	for {
		n, err := session.Read(buf)
		if err != nil {
			return
		}
		if n > 0 {
			s.broadcastToSession(session.ID, Message{Type: MsgData, Data: buf[:n]})
		}
	}
}

func (s *Server) sessionWaitLoop(session *Session) {
	session.Wait()
	s.broadcastToSession(session.ID, Message{Type: MsgExit})
	log.Printf("session exited: %s", session.ID)

	s.mu.Lock()
	delete(s.sessions, session.ID)
	s.mu.Unlock()
}

func (s *Server) broadcastToSession(sessionID string, msg Message) {
	s.mu.RLock()
	var conns []net.Conn
	for _, client := range s.clients {
		if client.sessionID == sessionID {
			conns = append(conns, client.conn)
		}
	}
	s.mu.RUnlock()

	for _, conn := range conns {
		WriteMessage(conn, msg)
	}
}

func (s *Server) sendError(conn net.Conn, errMsg string) {
	WriteMessage(conn, Message{Type: MsgSessionError, Data: []byte(errMsg)})
}

func (s *Server) removeClient(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, conn)
}

func (s *Server) Close() error {
	close(s.done)

	s.mu.Lock()
	for _, session := range s.sessions {
		session.Close()
	}
	s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}
	os.Remove(s.socketPath)
	return nil
}

func (s *Server) ListSessions() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ids []string
	for id := range s.sessions {
		ids = append(ids, id)
	}
	return ids
}
