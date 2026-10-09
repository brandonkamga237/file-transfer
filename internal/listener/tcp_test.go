//go:build unix

package listener

import (
	"net"
	"testing"
	"time"
)

func TestTcpListenerAcceptsConnection(t *testing.T) {
	// Port 0: the OS picks a free port, so there is no conflict.
	l := NewTcpListener("127.0.0.1", 0, 16)

	if err := l.Bind(); err != nil {
		t.Fatalf("Bind: %v", err)
	}
	if err := l.Listen(); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer l.Close()

	addr := l.Addr().String()

	// The connection is queued in the backlog before accept().
	client, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("Dial to %s: %v", addr, err)
	}
	defer client.Close()

	c, err := l.Accept()
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	defer c.Close()

	if c.RemoteAddr() == "" {
		t.Fatal("empty RemoteAddr")
	}
}

func TestListenBeforeBindFails(t *testing.T) {
	l := NewTcpListener("127.0.0.1", 0, 1)
	if err := l.Listen(); err == nil {
		t.Fatal("Listen without Bind must fail")
	}
}

func TestBindInvalidHost(t *testing.T) {
	l := NewTcpListener("not-an-ip", 0, 1)
	defer l.Close()

	if err := l.Bind(); err == nil {
		t.Fatal("Bind with an invalid address must fail")
	}
}
