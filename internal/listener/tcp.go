//go:build unix

// Package listener : implémentation TCP IPv4 basée sur les appels système.
package listener

import (
	"fmt"
	"net"
	"syscall"
)

// TcpListener ouvre une socket TCP IPv4 avec un backlog maîtrisé.
type TcpListener struct {
	host    string
	port    int
	backlog int

	fd int // socket brute, valide entre Bind et Listen
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

// Close libère la socket si elle a été ouverte.
func (l *TcpListener) Close() error {
	if l.fd >= 0 {
		err := syscall.Close(l.fd)
		l.fd = -1
		return err
	}
	return nil
}
