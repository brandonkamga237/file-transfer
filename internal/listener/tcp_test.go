//go:build unix

package listener

import (
	"net"
	"testing"
	"time"
)

func TestTcpListenerAcceptsConnection(t *testing.T) {
	// Port 0 : le système choisit un port libre, pas de conflit.
	l := NewTcpListener("127.0.0.1", 0, 16)

	if err := l.Bind(); err != nil {
		t.Fatalf("Bind : %v", err)
	}
	if err := l.Listen(); err != nil {
		t.Fatalf("Listen : %v", err)
	}
	defer l.Close()

	addr := l.Addr().String()

	// La connexion est placée dans la file du backlog avant accept().
	client, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("Dial vers %s : %v", addr, err)
	}
	defer client.Close()

	c, err := l.Accept()
	if err != nil {
		t.Fatalf("Accept : %v", err)
	}
	defer c.Close()

	if c.RemoteAddr() == "" {
		t.Fatal("RemoteAddr vide")
	}
}

func TestListenBeforeBindFails(t *testing.T) {
	l := NewTcpListener("127.0.0.1", 0, 1)
	if err := l.Listen(); err == nil {
		t.Fatal("Listen sans Bind doit échouer")
	}
}

func TestBindInvalidHost(t *testing.T) {
	l := NewTcpListener("pas-une-ip", 0, 1)
	defer l.Close()

	if err := l.Bind(); err == nil {
		t.Fatal("Bind avec une adresse invalide doit échouer")
	}
}
