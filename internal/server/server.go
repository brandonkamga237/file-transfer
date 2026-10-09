// Package server orchestre le cycle de vie du serveur.
package server

import (
	"errors"
	"fmt"
	"net"

	"kross/internal/config"
	"kross/internal/conn"
	"kross/internal/listener"
)

// Server pilote le démarrage, la boucle d'acceptation et l'arrêt.
type Server struct {
	cfg      config.ServerConfig
	listener listener.Listener
}

// New reçoit la configuration et un listener (injection de dépendance).
func New(cfg config.ServerConfig, l listener.Listener) *Server {
	return &Server{cfg: cfg, listener: l}
}

// Start lie, met en écoute, puis accepte les connexions jusqu'à Stop.
// Il bloque tant que le serveur tourne.
func (s *Server) Start() error {
	if err := s.listener.Bind(); err != nil {
		return fmt.Errorf("démarrage : %w", err)
	}
	if err := s.listener.Listen(); err != nil {
		s.listener.Close()
		return fmt.Errorf("démarrage : %w", err)
	}
	return s.acceptLoop()
}

// Stop ferme le listener, ce qui arrête la boucle d'acceptation.
func (s *Server) Stop() error {
	return s.listener.Close()
}

func (s *Server) acceptLoop() error {
	for {
		c, err := s.listener.Accept()
		if err != nil {
			// Fermeture volontaire : ce n'est pas une erreur.
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept : %w", err)
		}
		s.handle(c)
	}
}

// handle traite une connexion. Pour l'instant elle est fermée aussitôt :
// la lecture de la requête arrivera avec la fonctionnalité 2.
func (s *Server) handle(c *conn.ClientConnection) {
	defer c.Close()
}
