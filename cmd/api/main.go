// Command api démarre le serveur HTTP de gestion de la flotte et des vols
// de la compagnie aérienne.
//
// Variables d'environnement :
//
//	ADDR     adresse d'écoute (défaut « :8080 »)
//	DB_PATH  chemin du fichier SQLite (défaut « compagnie.db »)
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"compagnie-aerienne/internal/handler"
	"compagnie-aerienne/internal/service"
	"compagnie-aerienne/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("arrêt sur erreur", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) (err error) {
	addr := envOr("ADDR", ":8080")
	dbPath := envOr("DB_PATH", "compagnie.db")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, dbPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := st.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("fermeture de la base : %w", closeErr))
		}
	}()

	h := handler.New(service.New(st), logger)
	srv := &http.Server{
		Addr:              addr,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("serveur démarré", "addr", addr, "db", dbPath)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serveur HTTP : %w", err)
	case <-ctx.Done():
	}

	logger.Info("arrêt en cours")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("arrêt du serveur : %w", err)
	}
	return nil
}

// envOr renvoie la variable d'environnement key, ou def si elle est vide.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
