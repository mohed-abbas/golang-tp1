package service

import (
	"context"
	"errors"
	"fmt"

	"compagnie-aerienne/internal/apperr"
	"compagnie-aerienne/internal/model"
)

// ListerAvions renvoie tous les avions de la flotte.
func (s *Service) ListerAvions(ctx context.Context) ([]model.Avion, error) {
	avions, err := s.repo.ListAvions(ctx)
	if err != nil {
		return nil, fmt.Errorf("liste des avions : %w", err)
	}
	return avions, nil
}

// Avion renvoie l'avion portant l'immatriculation donnée.
func (s *Service) Avion(ctx context.Context, immatriculation string) (model.Avion, error) {
	a, err := s.repo.GetAvion(ctx, model.NormaliserImmatriculation(immatriculation))
	if err != nil {
		return model.Avion{}, fmt.Errorf("consultation : %w", err)
	}
	return a, nil
}

// CreerAvion valide puis enregistre un nouvel avion.
func (s *Service) CreerAvion(ctx context.Context, a model.Avion) (model.Avion, error) {
	a.Normaliser()
	if err := validerAvion(a); err != nil {
		return model.Avion{}, err
	}

	_, err := s.repo.GetAvion(ctx, a.Immatriculation)
	switch {
	case err == nil:
		return model.Avion{}, fmt.Errorf("avion %q : %w", a.Immatriculation, apperr.ErrConflict)
	case !errors.Is(err, apperr.ErrNotFound):
		return model.Avion{}, fmt.Errorf("vérification des doublons : %w", err)
	}

	a.Horodatage = s.horodatage()
	if err := s.repo.CreateAvion(ctx, a); err != nil {
		return model.Avion{}, fmt.Errorf("création : %w", err)
	}
	return a, nil
}

// SupprimerAvion supprime un avion, à condition qu'aucun vol ne lui soit
// rattaché.
func (s *Service) SupprimerAvion(ctx context.Context, immatriculation string) error {
	immatriculation = model.NormaliserImmatriculation(immatriculation)

	if _, err := s.repo.GetAvion(ctx, immatriculation); err != nil {
		return fmt.Errorf("suppression : %w", err)
	}

	n, err := s.repo.CountVolsByAvion(ctx, immatriculation)
	if err != nil {
		return fmt.Errorf("suppression : %w", err)
	}
	if n > 0 {
		return fmt.Errorf("avion %q (%d vol(s)) : %w", immatriculation, n, apperr.ErrAvionAvecVols)
	}

	if err := s.repo.DeleteAvion(ctx, immatriculation); err != nil {
		return fmt.Errorf("suppression : %w", err)
	}
	return nil
}

func validerAvion(a model.Avion) error {
	switch {
	case a.Immatriculation == "":
		return apperr.NewValidation("immatriculation", "obligatoire")
	case !identifiantValide(a.Immatriculation):
		return apperr.NewValidation("immatriculation", "2 à 10 caractères : lettres, chiffres et tirets")
	case a.Modele == "":
		return apperr.NewValidation("modele", "obligatoire")
	case tropLong(a.Modele):
		return apperr.NewValidation("modele", fmt.Sprintf("%d caractères maximum", maxTexte))
	case a.NombrePlaces <= 0:
		return apperr.NewValidation("nombre_places", "doit être strictement positif")
	case a.NombrePlaces > maxPlaces:
		return apperr.NewValidation("nombre_places", fmt.Sprintf("%d maximum", maxPlaces))
	case a.Compagnie == "":
		return apperr.NewValidation("compagnie", "obligatoire")
	case tropLong(a.Compagnie):
		return apperr.NewValidation("compagnie", fmt.Sprintf("%d caractères maximum", maxTexte))
	}
	return nil
}
