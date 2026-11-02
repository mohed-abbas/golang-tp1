// Package service contient la logique métier de la compagnie : validation des
// données, détection des doublons, règles de suppression et calcul des
// statistiques. Il ne connaît ni HTTP ni SQL.
package service

import (
	"context"
	"time"
	"unicode/utf8"

	"compagnie-aerienne/internal/model"
)

// Limites appliquées lors de la validation.
const (
	maxIdentifiant = 10
	maxTexte       = 100
	maxPlaces      = 1000
)

// Repository décrit l'accès aux données dont le service a besoin.
// *store.Store le satisfait ; une implémentation en mémoire peut le remplacer
// dans les tests.
type Repository interface {
	ListAvions(ctx context.Context) ([]model.Avion, error)
	GetAvion(ctx context.Context, immatriculation string) (model.Avion, error)
	CreateAvion(ctx context.Context, a model.Avion) error
	DeleteAvion(ctx context.Context, immatriculation string) error

	ListVolsByAvion(ctx context.Context, immatriculation string) ([]model.Vol, error)
	VolExiste(ctx context.Context, numeroVol string) (bool, error)
	CountVolsByAvion(ctx context.Context, immatriculation string) (int, error)
	CreateVol(ctx context.Context, v model.Vol) error

	CountAvions(ctx context.Context) (int, error)
	CountVolsParStatut(ctx context.Context) (map[model.Statut]int, error)
}

// Service applique les règles métier de la compagnie.
type Service struct {
	repo Repository
	now  func() time.Time
}

// New crée un Service s'appuyant sur repo.
func New(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// horodatage renvoie l'instant courant, en UTC et à la seconde près.
func (s *Service) horodatage() model.Horodatage {
	return model.Horodatage{CreeLe: s.now().UTC().Truncate(time.Second)}
}

// identifiantValide vérifie qu'un identifiant (immatriculation, numéro de vol)
// ne contient que des majuscules, des chiffres et des tirets intérieurs.
func identifiantValide(id string) bool {
	if len(id) < 2 || len(id) > maxIdentifiant {
		return false
	}
	for i, r := range id {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-' && i > 0 && i < len(id)-1:
		default:
			return false
		}
	}
	return true
}

// tropLong indique si s dépasse la longueur maximale autorisée pour un texte.
func tropLong(s string) bool {
	return utf8.RuneCountInString(s) > maxTexte
}
