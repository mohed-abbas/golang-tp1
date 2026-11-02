// Package store gère la persistance des avions et des vols dans une base
// SQLite, via le pilote pur Go modernc.org/sqlite.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// schema crée les tables si elles n'existent pas encore.
var schema = []string{
	`CREATE TABLE IF NOT EXISTS avions (
		immatriculation TEXT PRIMARY KEY,
		modele          TEXT NOT NULL,
		nombre_places   INTEGER NOT NULL CHECK (nombre_places > 0),
		compagnie       TEXT NOT NULL,
		cree_le         TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS vols (
		numero_vol       TEXT PRIMARY KEY,
		aeroport_depart  TEXT NOT NULL,
		aeroport_arrivee TEXT NOT NULL,
		date             TEXT NOT NULL,
		immatriculation  TEXT NOT NULL REFERENCES avions (immatriculation) ON DELETE RESTRICT,
		statut           TEXT NOT NULL CHECK (statut IN ('programme', 'en_cours', 'annule')),
		cree_le          TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_vols_immatriculation ON vols (immatriculation)`,
}

// Store donne accès à la base SQLite.
type Store struct {
	db *sql.DB
}

// Open ouvre (ou crée) la base située à path et applique le schéma.
func Open(ctx context.Context, path string) (*Store, error) {
	dsn := "file:" + path +
		"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("ouverture de la base %q : %w", path, err)
	}

	for _, stmt := range schema {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return nil, errors.Join(fmt.Errorf("création du schéma : %w", err), db.Close())
		}
	}
	return &Store{db: db}, nil
}

// Close ferme la connexion à la base.
func (s *Store) Close() error {
	return s.db.Close()
}

// scanner est satisfait par *sql.Row et *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// estViolation indique si err est une violation de contrainte SQLite portant
// l'un des codes étendus fournis.
func estViolation(err error, codes ...int) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	for _, code := range codes {
		if sqliteErr.Code() == code {
			return true
		}
	}
	return false
}

// estDoublon indique si err provient d'une clé primaire ou unique déjà utilisée.
func estDoublon(err error) bool {
	return estViolation(err, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, sqlite3.SQLITE_CONSTRAINT_UNIQUE)
}

// estCleEtrangere indique si err provient d'une clé étrangère non respectée.
func estCleEtrangere(err error) bool {
	return estViolation(err, sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY)
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("horodatage %q illisible : %w", s, err)
	}
	return t, nil
}
