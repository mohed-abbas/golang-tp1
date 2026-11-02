package service

import (
	"context"
	"fmt"

	"compagnie-aerienne/internal/model"
)

// Stats calcule le nombre d'avions et le nombre de vols par statut. Chaque
// statut apparaît dans le résultat, même à zéro.
func (s *Service) Stats(ctx context.Context) (model.Stats, error) {
	nbAvions, err := s.repo.CountAvions(ctx)
	if err != nil {
		return model.Stats{}, fmt.Errorf("statistiques : %w", err)
	}
	parStatut, err := s.repo.CountVolsParStatut(ctx)
	if err != nil {
		return model.Stats{}, fmt.Errorf("statistiques : %w", err)
	}

	stats := model.Stats{
		NombreAvions:  nbAvions,
		VolsParStatut: make(map[model.Statut]int, len(model.Statuts())),
	}
	for _, statut := range model.Statuts() {
		n := parStatut[statut]
		stats.VolsParStatut[statut] = n
		stats.NombreVols += n
	}
	return stats, nil
}
