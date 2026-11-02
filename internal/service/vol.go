package service

import (
	"context"
	"fmt"
	"time"

	"compagnie-aerienne/internal/apperr"
	"compagnie-aerienne/internal/model"
)

// VolsAvion renvoie les vols d'un avion existant.
func (s *Service) VolsAvion(ctx context.Context, immatriculation string) ([]model.Vol, error) {
	immatriculation = model.NormaliserImmatriculation(immatriculation)

	if _, err := s.repo.GetAvion(ctx, immatriculation); err != nil {
		return nil, fmt.Errorf("liste des vols : %w", err)
	}
	vols, err := s.repo.ListVolsByAvion(ctx, immatriculation)
	if err != nil {
		return nil, fmt.Errorf("liste des vols : %w", err)
	}
	return vols, nil
}

// CreerVol valide puis enregistre un vol pour l'avion indiqué. Sans statut
// fourni, le vol est « programme ».
func (s *Service) CreerVol(ctx context.Context, immatriculation string, v model.Vol) (model.Vol, error) {
	immatriculation = model.NormaliserImmatriculation(immatriculation)
	v.Normaliser()

	if v.Immatriculation != "" && v.Immatriculation != immatriculation {
		return model.Vol{}, apperr.NewValidation("immatriculation", "ne correspond pas à l'avion de l'URL")
	}
	v.Immatriculation = immatriculation
	if v.Statut == "" {
		v.Statut = model.StatutProgramme
	}
	if err := validerVol(v); err != nil {
		return model.Vol{}, err
	}

	if _, err := s.repo.GetAvion(ctx, immatriculation); err != nil {
		return model.Vol{}, fmt.Errorf("création du vol : %w", err)
	}

	existe, err := s.repo.VolExiste(ctx, v.NumeroVol)
	if err != nil {
		return model.Vol{}, fmt.Errorf("vérification des doublons : %w", err)
	}
	if existe {
		return model.Vol{}, fmt.Errorf("vol %q : %w", v.NumeroVol, apperr.ErrConflict)
	}

	v.Horodatage = s.horodatage()
	if err := s.repo.CreateVol(ctx, v); err != nil {
		return model.Vol{}, fmt.Errorf("création du vol : %w", err)
	}
	return v, nil
}

func validerVol(v model.Vol) error {
	switch {
	case v.NumeroVol == "":
		return apperr.NewValidation("numero_vol", "obligatoire")
	case !identifiantValide(v.NumeroVol):
		return apperr.NewValidation("numero_vol", "2 à 10 caractères : lettres, chiffres et tirets")
	case v.Depart == "":
		return apperr.NewValidation("aeroport_depart", "obligatoire")
	case tropLong(v.Depart):
		return apperr.NewValidation("aeroport_depart", fmt.Sprintf("%d caractères maximum", maxTexte))
	case v.Arrivee == "":
		return apperr.NewValidation("aeroport_arrivee", "obligatoire")
	case tropLong(v.Arrivee):
		return apperr.NewValidation("aeroport_arrivee", fmt.Sprintf("%d caractères maximum", maxTexte))
	case v.Boucle():
		return apperr.NewValidation("aeroport_arrivee", "doit être différent de l'aéroport de départ")
	case v.Date == "":
		return apperr.NewValidation("date", "obligatoire")
	case !dateValide(v.Date):
		return apperr.NewValidation("date", "format attendu YYYY-MM-DD")
	case !v.Statut.Valide():
		return apperr.NewValidation("statut", "valeurs possibles : programme, en_cours, annule")
	}
	return nil
}

func dateValide(date string) bool {
	_, err := time.Parse(model.FormatDate, date)
	return err == nil
}
