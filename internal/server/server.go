// Package server orchestrates the server lifecycle.
package server

import (
	"errors"
	"fmt"
	"net"

	"kross/internal/config"
	"kross/internal/conn"
	"kross/internal/listener"
)

// Server drives startup, the accept loop and shutdown.
type Server struct {
	cfg      config.ServerConfig
	listener listener.Listener
}

// New receives the configuration and a listener (dependency injection).
func New(cfg config.ServerConfig, l listener.Listener) *Server {
	return &Server{cfg: cfg, listener: l}
}

// Start binds, listens, then accepts connections until Stop is called.
// It blocks while the server is running.
func (s *Server) Start() error {
	if err := s.listener.Bind(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := s.listener.Listen(); err != nil {
		s.listener.Close()
		return fmt.Errorf("start: %w", err)
	}
	return s.acceptLoop()
}

// Stop closes the listener, which ends the accept loop.
func (s *Server) Stop() error {
	return s.listener.Close()
}

func (s *Server) acceptLoop() error {
	for {
		c, err := s.listener.Accept()
		if err != nil {
			// Intentional shutdown: not an error.
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept: %w", err)
		}
		s.handle(c)
	}
}

// handle processes a connection. For now it is closed right away:
// reading the request will come with HTTP request parsing.
func (s *Server) handle(c *conn.ClientConnection) {
	defer c.Close()
}
