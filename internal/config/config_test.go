package config

import "testing"

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("la configuration par défaut doit être valide : %v", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ServerConfig
		wantErr bool
	}{
		{"port nul accepté (port libre choisi par le système)", ServerConfig{Host: "127.0.0.1", Port: 0, Backlog: 1}, false},
		{"port négatif", ServerConfig{Host: "127.0.0.1", Port: -1, Backlog: 1}, true},
		{"port trop grand", ServerConfig{Host: "127.0.0.1", Port: 70000, Backlog: 1}, true},
		{"backlog nul", ServerConfig{Host: "127.0.0.1", Port: 8080, Backlog: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() erreur = %v, attendue erreur = %v", err, tt.wantErr)
			}
		})
	}
}
