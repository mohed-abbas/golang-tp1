package store

import (
	"context"
	"fmt"

	"compagnie-aerienne/internal/model"
)

// CountAvions renvoie le nombre d'avions de la flotte.
func (s *Store) CountAvions(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM avions`).Scan(&n); err != nil {
		return 0, fmt.Errorf("comptage des avions : %w", err)
	}
	return n, nil
}

// CountVolsParStatut renvoie le nombre de vols pour chaque statut présent
// en base.
func (s *Store) CountVolsParStatut(ctx context.Context) (map[model.Statut]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT statut, COUNT(*) FROM vols GROUP BY statut`)
	if err != nil {
		return nil, fmt.Errorf("comptage des vols : %w", err)
	}
	defer rows.Close()

	compteurs := make(map[model.Statut]int)
	for rows.Next() {
		var (
			statut model.Statut
			n      int
		)
		if err := rows.Scan(&statut, &n); err != nil {
			return nil, fmt.Errorf("lecture d'un compteur : %w", err)
		}
		compteurs[statut] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcours des compteurs : %w", err)
	}
	return compteurs, nil
}
