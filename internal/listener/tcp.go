//go:build unix

// Package listener : implémentation TCP IPv4 basée sur les appels système.
package listener

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"kross/internal/conn"
)

// Vérifie à la compilation que TcpListener respecte le contrat Listener.
var _ Listener = (*TcpListener)(nil)

// TcpListener ouvre une socket TCP IPv4 avec un backlog maîtrisé.
type TcpListener struct {
	host    string
	port    int
	backlog int

	fd int          // socket brute, valide entre Bind et Listen
	ln net.Listener // listener Go, valide après Listen
}

// NewTcpListener prépare un listener sans ouvrir de socket.
func NewTcpListener(host string, port, backlog int) *TcpListener {
	return &TcpListener{
		host:    host,
		port:    port,
		backlog: backlog,
		fd:      -1,
	}
}

// Bind crée la socket et la lie à l'adresse et au port.
func (l *TcpListener) Bind() error {
	ip := net.ParseIP(l.host).To4()
	if ip == nil {
		return fmt.Errorf("adresse IPv4 invalide : %q", l.host)
	}

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return fmt.Errorf("socket : %w", err)
	}

	// Permet de relancer le serveur sans attendre la fin de TIME_WAIT.
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("setsockopt : %w", err)
	}

	sa := &syscall.SockaddrInet4{Port: l.port}
	copy(sa.Addr[:], ip)

	if err := syscall.Bind(fd, sa); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("bind %s:%d : %w", l.host, l.port, err)
	}

	l.fd = fd
	return nil
}

// Listen met la socket en écoute avec la taille de file configurée,
// puis la convertit en net.Listener utilisable par Go.
func (l *TcpListener) Listen() error {
	if l.fd < 0 {
		return errors.New("listen appelé avant bind")
	}

	if err := syscall.Listen(l.fd, l.backlog); err != nil {
		return fmt.Errorf("listen : %w", err)
	}

	// os.NewFile prend possession du descripteur.
	f := os.NewFile(uintptr(l.fd), "tcp-listener")
	l.fd = -1

	ln, err := net.FileListener(f)
	f.Close() // net.FileListener duplique le descripteur, on peut fermer f
	if err != nil {
		return fmt.Errorf("conversion en net.Listener : %w", err)
	}

	l.ln = ln
	return nil
}

// Accept attend une connexion entrante et l'enveloppe dans une ClientConnection.
func (l *TcpListener) Accept() (*conn.ClientConnection, error) {
	if l.ln == nil {
		return nil, errors.New("accept appelé avant listen")
	}

	raw, err := l.ln.Accept()
	if err != nil {
		return nil, err
	}
	return conn.New(raw), nil
}

// Close ferme la socket, ce qui débloque Accept avec net.ErrClosed.
func (l *TcpListener) Close() error {
	if l.ln != nil {
		return l.ln.Close()
	}
	if l.fd >= 0 {
		err := syscall.Close(l.fd)
		l.fd = -1
		return err
	}
	return nil
}

// Addr retourne l'adresse réelle, utile quand le port demandé était 0.
func (l *TcpListener) Addr() net.Addr {
	if l.ln == nil {
		return nil
	}
	return l.ln.Addr()
}
