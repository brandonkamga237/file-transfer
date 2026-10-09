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

type Server struct {
	cfg      config.ServerConfig
	listener listener.Listener
}

func New(cfg config.ServerConfig, l listener.Listener) *Server {
	return &Server{cfg: cfg, listener: l}
}

// Start blocks until Stop is called.
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

func (s *Server) Stop() error {
	return s.listener.Close()
}

func (s *Server) acceptLoop() error {
	for {
		c, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept: %w", err)
		}
		s.handle(c)
	}
}

func (s *Server) handle(c *conn.ClientConnection) {
	defer c.Close()
}
