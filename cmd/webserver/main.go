package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"kross/internal/config"
	"kross/internal/listener"
	"kross/internal/server"
)

func main() {
	cfg := config.Default()

	flag.StringVar(&cfg.Host, "host", cfg.Host, "listen address")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "listen port")
	flag.IntVar(&cfg.Backlog, "backlog", cfg.Backlog, "pending connection queue size")
	flag.Parse()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	l := listener.NewTcpListener(cfg.Host, cfg.Port, cfg.Backlog)
	srv := server.New(cfg, l)

	go func() {
		<-ctx.Done()
		if err := srv.Stop(); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("listening on %s:%d (backlog %d)", cfg.Host, cfg.Port, cfg.Backlog)
	if err := srv.Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
	log.Println("server stopped")
}
