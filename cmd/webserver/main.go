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

	flag.StringVar(&cfg.Host, "host", cfg.Host, "adresse d'écoute")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "port d'écoute")
	flag.IntVar(&cfg.Backlog, "backlog", cfg.Backlog, "taille de la file d'attente")
	flag.Parse()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("configuration invalide : %v", err)
	}

	// Arrêt propre sur Ctrl+C ou SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	l := listener.NewTcpListener(cfg.Host, cfg.Port, cfg.Backlog)
	srv := server.New(cfg, l)

	go func() {
		<-ctx.Done()
		if err := srv.Stop(); err != nil {
			log.Printf("erreur à l'arrêt : %v", err)
		}
	}()

	log.Printf("écoute sur %s:%d (backlog %d)", cfg.Host, cfg.Port, cfg.Backlog)
	if err := srv.Start(); err != nil {
		log.Fatalf("serveur : %v", err)
	}
	log.Println("serveur arrêté")
}
