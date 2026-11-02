package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"compagnie-aerienne/internal/apperr"
	"compagnie-aerienne/internal/model"
)

const colonnesAvion = `immatriculation, modele, nombre_places, compagnie, cree_le`

func scanAvion(sc scanner) (model.Avion, error) {
	var (
		a      model.Avion
		creeLe string
	)
	if err := sc.Scan(&a.Immatriculation, &a.Modele, &a.NombrePlaces, &a.Compagnie, &creeLe); err != nil {
		return model.Avion{}, err
	}
	t, err := parseTime(creeLe)
	if err != nil {
		return model.Avion{}, err
	}
	a.CreeLe = t
	return a, nil
}

// ListAvions renvoie tous les avions, triés par immatriculation.
func (s *Store) ListAvions(ctx context.Context) ([]model.Avion, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+colonnesAvion+` FROM avions ORDER BY immatriculation`)
	if err != nil {
		return nil, fmt.Errorf("liste des avions : %w", err)
	}
	defer rows.Close()

	avions := make([]model.Avion, 0)
	for rows.Next() {
		a, err := scanAvion(rows)
		if err != nil {
			return nil, fmt.Errorf("lecture d'un avion : %w", err)
		}
		avions = append(avions, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcours des avions : %w", err)
	}
	return avions, nil
}

// GetAvion renvoie l'avion portant l'immatriculation donnée, ou une erreur
// enveloppant apperr.ErrNotFound s'il n'existe pas.
func (s *Store) GetAvion(ctx context.Context, immatriculation string) (model.Avion, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+colonnesAvion+` FROM avions WHERE immatriculation = ?`, immatriculation)

	a, err := scanAvion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Avion{}, fmt.Errorf("avion %q : %w", immatriculation, apperr.ErrNotFound)
	}
	if err != nil {
		return model.Avion{}, fmt.Errorf("lecture de l'avion %q : %w", immatriculation, err)
	}
	return a, nil
}

// CreateAvion insère un avion. Une immatriculation déjà utilisée produit une
// erreur enveloppant apperr.ErrConflict.
func (s *Store) CreateAvion(ctx context.Context, a model.Avion) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO avions (`+colonnesAvion+`) VALUES (?, ?, ?, ?, ?)`,
		a.Immatriculation, a.Modele, a.NombrePlaces, a.Compagnie, formatTime(a.CreeLe))
	if estDoublon(err) {
		return fmt.Errorf("avion %q : %w", a.Immatriculation, apperr.ErrConflict)
	}
	if err != nil {
		return fmt.Errorf("insertion de l'avion %q : %w", a.Immatriculation, err)
	}
	return nil
}

// DeleteAvion supprime un avion. La clé étrangère des vols empêche la
// suppression d'un avion encore utilisé (apperr.ErrAvionAvecVols).
func (s *Store) DeleteAvion(ctx context.Context, immatriculation string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM avions WHERE immatriculation = ?`, immatriculation)
	if estCleEtrangere(err) {
		return fmt.Errorf("avion %q : %w", immatriculation, apperr.ErrAvionAvecVols)
	}
	if err != nil {
		return fmt.Errorf("suppression de l'avion %q : %w", immatriculation, err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("suppression de l'avion %q : %w", immatriculation, err)
	}
	if n == 0 {
		return fmt.Errorf("avion %q : %w", immatriculation, apperr.ErrNotFound)
	}
	return nil
}
