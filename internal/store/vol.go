package store

import (
	"context"
	"fmt"

	"compagnie-aerienne/internal/apperr"
	"compagnie-aerienne/internal/model"
)

const colonnesVol = `numero_vol, aeroport_depart, aeroport_arrivee, date, immatriculation, statut, cree_le`

func scanVol(sc scanner) (model.Vol, error) {
	var (
		v      model.Vol
		creeLe string
	)
	if err := sc.Scan(&v.NumeroVol, &v.Depart, &v.Arrivee, &v.Date,
		&v.Immatriculation, &v.Statut, &creeLe); err != nil {
		return model.Vol{}, err
	}
	t, err := parseTime(creeLe)
	if err != nil {
		return model.Vol{}, err
	}
	v.CreeLe = t
	return v, nil
}

// ListVolsByAvion renvoie les vols d'un avion, triés par date puis numéro.
func (s *Store) ListVolsByAvion(ctx context.Context, immatriculation string) ([]model.Vol, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+colonnesVol+` FROM vols WHERE immatriculation = ? ORDER BY date, numero_vol`,
		immatriculation)
	if err != nil {
		return nil, fmt.Errorf("liste des vols de %q : %w", immatriculation, err)
	}
	defer rows.Close()

	vols := make([]model.Vol, 0)
	for rows.Next() {
		v, err := scanVol(rows)
		if err != nil {
			return nil, fmt.Errorf("lecture d'un vol : %w", err)
		}
		vols = append(vols, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcours des vols : %w", err)
	}
	return vols, nil
}

// VolExiste indique si un vol porte déjà ce numéro.
func (s *Store) VolExiste(ctx context.Context, numeroVol string) (bool, error) {
	var existe bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM vols WHERE numero_vol = ?)`, numeroVol).Scan(&existe)
	if err != nil {
		return false, fmt.Errorf("recherche du vol %q : %w", numeroVol, err)
	}
	return existe, nil
}

// CountVolsByAvion renvoie le nombre de vols rattachés à un avion.
func (s *Store) CountVolsByAvion(ctx context.Context, immatriculation string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM vols WHERE immatriculation = ?`, immatriculation).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("comptage des vols de %q : %w", immatriculation, err)
	}
	return n, nil
}

// CreateVol insère un vol. Un numéro déjà utilisé produit apperr.ErrConflict,
// un avion inexistant produit apperr.ErrNotFound.
func (s *Store) CreateVol(ctx context.Context, v model.Vol) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO vols (`+colonnesVol+`) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		v.NumeroVol, v.Depart, v.Arrivee, v.Date, v.Immatriculation, string(v.Statut), formatTime(v.CreeLe))
	switch {
	case estDoublon(err):
		return fmt.Errorf("vol %q : %w", v.NumeroVol, apperr.ErrConflict)
	case estCleEtrangere(err):
		return fmt.Errorf("avion %q : %w", v.Immatriculation, apperr.ErrNotFound)
	case err != nil:
		return fmt.Errorf("insertion du vol %q : %w", v.NumeroVol, err)
	}
	return nil
}
