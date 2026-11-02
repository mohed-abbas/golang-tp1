package service

import (
	"context"
	"fmt"

	"compagnie-aerienne/internal/apperr"
	"compagnie-aerienne/internal/model"
)

// fakeRepo est un Repository en mémoire utilisé par les tests. Si err est
// renseignée, toutes les méthodes la renvoient.
type fakeRepo struct {
	avions map[string]model.Avion
	vols   map[string]model.Vol
	err    error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		avions: make(map[string]model.Avion),
		vols:   make(map[string]model.Vol),
	}
}

func (f *fakeRepo) ListAvions(ctx context.Context) ([]model.Avion, error) {
	if f.err != nil {
		return nil, f.err
	}
	avions := make([]model.Avion, 0, len(f.avions))
	for _, a := range f.avions {
		avions = append(avions, a)
	}
	return avions, nil
}

func (f *fakeRepo) GetAvion(ctx context.Context, immatriculation string) (model.Avion, error) {
	if f.err != nil {
		return model.Avion{}, f.err
	}
	a, ok := f.avions[immatriculation]
	if !ok {
		return model.Avion{}, fmt.Errorf("avion %q : %w", immatriculation, apperr.ErrNotFound)
	}
	return a, nil
}

func (f *fakeRepo) CreateAvion(ctx context.Context, a model.Avion) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.avions[a.Immatriculation]; ok {
		return fmt.Errorf("avion %q : %w", a.Immatriculation, apperr.ErrConflict)
	}
	f.avions[a.Immatriculation] = a
	return nil
}

func (f *fakeRepo) DeleteAvion(ctx context.Context, immatriculation string) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.avions[immatriculation]; !ok {
		return fmt.Errorf("avion %q : %w", immatriculation, apperr.ErrNotFound)
	}
	delete(f.avions, immatriculation)
	return nil
}

func (f *fakeRepo) ListVolsByAvion(ctx context.Context, immatriculation string) ([]model.Vol, error) {
	if f.err != nil {
		return nil, f.err
	}
	vols := make([]model.Vol, 0)
	for _, v := range f.vols {
		if v.Immatriculation == immatriculation {
			vols = append(vols, v)
		}
	}
	return vols, nil
}

func (f *fakeRepo) VolExiste(ctx context.Context, numeroVol string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	_, ok := f.vols[numeroVol]
	return ok, nil
}

func (f *fakeRepo) CountVolsByAvion(ctx context.Context, immatriculation string) (int, error) {
	vols, err := f.ListVolsByAvion(ctx, immatriculation)
	return len(vols), err
}

func (f *fakeRepo) CreateVol(ctx context.Context, v model.Vol) error {
	if f.err != nil {
		return f.err
	}
	f.vols[v.NumeroVol] = v
	return nil
}

func (f *fakeRepo) CountAvions(ctx context.Context) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return len(f.avions), nil
}

func (f *fakeRepo) CountVolsParStatut(ctx context.Context) (map[model.Statut]int, error) {
	if f.err != nil {
		return nil, f.err
	}
	compteurs := make(map[model.Statut]int)
	for _, v := range f.vols {
		compteurs[v.Statut]++
	}
	return compteurs, nil
}
