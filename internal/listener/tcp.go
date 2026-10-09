//go:build unix

package listener

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"kross/internal/conn"
)

var _ Listener = (*TcpListener)(nil)

// TcpListener builds the socket with system calls because net.Listen does not expose the backlog.
type TcpListener struct {
	host    string
	port    int
	backlog int

	fd int          // raw socket, valid between Bind and Listen
	ln net.Listener // Go listener, valid after Listen
}

func NewTcpListener(host string, port, backlog int) *TcpListener {
	return &TcpListener{
		host:    host,
		port:    port,
		backlog: backlog,
		fd:      -1,
	}
}

func (l *TcpListener) Bind() error {
	ip := net.ParseIP(l.host).To4()
	if ip == nil {
		return fmt.Errorf("invalid IPv4 address: %q", l.host)
	}

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return fmt.Errorf("socket: %w", err)
	}

	// Allows restarting without waiting for TIME_WAIT to expire.
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("setsockopt: %w", err)
	}

	sa := &syscall.SockaddrInet4{Port: l.port}
	copy(sa.Addr[:], ip)

	if err := syscall.Bind(fd, sa); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("bind %s:%d: %w", l.host, l.port, err)
	}

	l.fd = fd
	return nil
}

// Listen starts listening and hands the socket over to a net.Listener.
func (l *TcpListener) Listen() error {
	if l.fd < 0 {
		return errors.New("listen called before bind")
	}

	if err := syscall.Listen(l.fd, l.backlog); err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	// os.NewFile takes ownership of the descriptor.
	f := os.NewFile(uintptr(l.fd), "tcp-listener")
	l.fd = -1

	ln, err := net.FileListener(f)
	f.Close() // net.FileListener duplicates the descriptor, so f can be closed
	if err != nil {
		return fmt.Errorf("convert to net.Listener: %w", err)
	}

	l.ln = ln
	return nil
}

func (l *TcpListener) Accept() (*conn.ClientConnection, error) {
	if l.ln == nil {
		return nil, errors.New("accept called before listen")
	}

	raw, err := l.ln.Accept()
	if err != nil {
		return nil, err
	}
	return conn.New(raw), nil
}

// Close closes the socket, which unblocks Accept with net.ErrClosed.
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

// Addr returns the actual address, useful when the requested port was 0.
func (l *TcpListener) Addr() net.Addr {
	if l.ln == nil {
		return nil
	}
	return l.ln.Addr()
}
