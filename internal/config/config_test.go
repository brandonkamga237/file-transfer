package config

import "testing"

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("default configuration must be valid: %v", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ServerConfig
		wantErr bool
	}{
		{"zero port accepted (free port picked by the OS)", ServerConfig{Host: "127.0.0.1", Port: 0, Backlog: 1}, false},
		{"negative port", ServerConfig{Host: "127.0.0.1", Port: -1, Backlog: 1}, true},
		{"port too large", ServerConfig{Host: "127.0.0.1", Port: 70000, Backlog: 1}, true},
		{"zero backlog", ServerConfig{Host: "127.0.0.1", Port: 8080, Backlog: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
