package server

import (
	"net"
	"sync"
	"testing"
	"time"

	"kross/internal/config"
	"kross/internal/conn"
)

// fakeListener replaces TcpListener: no real port is opened.
type fakeListener struct {
	bound    bool
	listened bool

	listenedCh chan struct{}
	closed     chan struct{}
	closeOnce  sync.Once
}

func newFakeListener() *fakeListener {
	return &fakeListener{
		listenedCh: make(chan struct{}),
		closed:     make(chan struct{}),
	}
}

func (f *fakeListener) Bind() error {
	f.bound = true
	return nil
}

func (f *fakeListener) Listen() error {
	f.listened = true
	close(f.listenedCh)
	return nil
}

func (f *fakeListener) Accept() (*conn.ClientConnection, error) {
	<-f.closed
	return nil, net.ErrClosed
}

func (f *fakeListener) Close() error {
	f.closeOnce.Do(func() { close(f.closed) })
	return nil
}

func TestStartBindsListensAndStopsCleanly(t *testing.T) {
	fl := newFakeListener()
	srv := New(config.Default(), fl)

	done := make(chan error, 1)
	go func() { done <- srv.Start() }()

	// Wait for Listen to be called before stopping.
	select {
	case <-fl.listenedCh:
	case <-time.After(time.Second):
		t.Fatal("Listen was not called")
	}

	if err := srv.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start must return nil after Stop, got: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not return after Stop")
	}

	if !fl.bound || !fl.listened {
		t.Fatal("Bind and Listen must be called before Accept")
	}
}
