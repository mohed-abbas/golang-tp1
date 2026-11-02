package main

import (
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvOr(t *testing.T) {
	tests := []struct {
		name   string
		valeur string
		defini bool
		want   string
	}{
		{"variable définie", "/data/test.db", true, "/data/test.db"},
		{"variable vide", "", true, "defaut"},
		{"variable absente", "", false, "defaut"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const key = "COMPAGNIE_TEST_ENV"
			if tt.defini {
				t.Setenv(key, tt.valeur)
			}
			if got := envOr(key, "defaut"); got != tt.want {
				t.Errorf("envOr() = %q, attendu %q", got, tt.want)
			}
		})
	}
}

func TestRunEchecDemarrage(t *testing.T) {
	occupe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("réservation d'un port : %v", err)
	}
	t.Cleanup(func() {
		if err := occupe.Close(); err != nil {
			t.Errorf("fermeture du port réservé : %v", err)
		}
	})

	tests := []struct {
		name    string
		dbPath  string
		addr    string
		wantErr string
	}{
		{
			name:    "dossier de la base inexistant",
			dbPath:  filepath.Join(t.TempDir(), "absent", "compagnie.db"),
			addr:    "127.0.0.1:0",
			wantErr: "création du schéma",
		},
		{
			name:    "port déjà utilisé",
			dbPath:  filepath.Join(t.TempDir(), "compagnie.db"),
			addr:    occupe.Addr().String(),
			wantErr: "serveur HTTP",
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DB_PATH", tt.dbPath)
			t.Setenv("ADDR", tt.addr)

			err := run(logger)
			if err == nil {
				t.Fatal("run() = nil, attendu une erreur")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("run() = %q, attendu une erreur contenant %q", err, tt.wantErr)
			}
		})
	}
}
