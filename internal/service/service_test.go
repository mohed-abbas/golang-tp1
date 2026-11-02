package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"compagnie-aerienne/internal/apperr"
	"compagnie-aerienne/internal/model"
)

var maintenant = time.Date(2026, 11, 2, 9, 30, 15, 0, time.UTC)

// newTestService crée un Service sur repo, avec une horloge figée.
func newTestService(repo Repository) *Service {
	svc := New(repo)
	svc.now = func() time.Time { return maintenant }
	return svc
}

func avionValide() model.Avion {
	return model.Avion{Immatriculation: "F-GKXA", Modele: "A320", NombrePlaces: 180, Compagnie: "Air France"}
}

func volValide() model.Vol {
	return model.Vol{
		NumeroVol: "AF1234",
		Trajet:    model.Trajet{Depart: "CDG", Arrivee: "JFK"},
		Date:      "2026-12-01",
	}
}

// verifierErreur contrôle err : aucune erreur si wantErr et wantChamp sont
// vides, sinon la sentinelle wantErr ou une ValidationError sur wantChamp.
func verifierErreur(t *testing.T, err, wantErr error, wantChamp string) {
	t.Helper()
	switch {
	case wantChamp != "":
		var ve *apperr.ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("erreur = %v, attendu une ValidationError sur %q", err, wantChamp)
		}
		if ve.Champ != wantChamp {
			t.Fatalf("champ en erreur = %q, attendu %q", ve.Champ, wantChamp)
		}
	case wantErr != nil:
		if !errors.Is(err, wantErr) {
			t.Fatalf("erreur = %v, attendu %v", err, wantErr)
		}
	case err != nil:
		t.Fatalf("erreur inattendue : %v", err)
	}
}

func TestCreerAvion(t *testing.T) {
	modifier := func(f func(a *model.Avion)) model.Avion {
		a := avionValide()
		f(&a)
		return a
	}

	tests := []struct {
		name      string
		avion     model.Avion
		wantErr   error
		wantChamp string
		wantImmat string
	}{
		{name: "valide", avion: avionValide(), wantImmat: "F-GKXA"},
		{name: "normalisé", avion: modifier(func(a *model.Avion) { a.Immatriculation = " f-hbxc " }), wantImmat: "F-HBXC"},
		{name: "immatriculation vide", avion: modifier(func(a *model.Avion) { a.Immatriculation = "  " }), wantChamp: "immatriculation"},
		{name: "immatriculation avec espace", avion: modifier(func(a *model.Avion) { a.Immatriculation = "F GKXA" }), wantChamp: "immatriculation"},
		{name: "immatriculation finissant par un tiret", avion: modifier(func(a *model.Avion) { a.Immatriculation = "FGKXA-" }), wantChamp: "immatriculation"},
		{name: "immatriculation trop longue", avion: modifier(func(a *model.Avion) { a.Immatriculation = "F-ABCDEFGHIJ" }), wantChamp: "immatriculation"},
		{name: "modèle vide", avion: modifier(func(a *model.Avion) { a.Modele = "" }), wantChamp: "modele"},
		{name: "modèle trop long", avion: modifier(func(a *model.Avion) { a.Modele = strings.Repeat("a", 101) }), wantChamp: "modele"},
		{name: "zéro place", avion: modifier(func(a *model.Avion) { a.NombrePlaces = 0 }), wantChamp: "nombre_places"},
		{name: "places négatives", avion: modifier(func(a *model.Avion) { a.NombrePlaces = -5 }), wantChamp: "nombre_places"},
		{name: "trop de places", avion: modifier(func(a *model.Avion) { a.NombrePlaces = 1001 }), wantChamp: "nombre_places"},
		{name: "compagnie vide", avion: modifier(func(a *model.Avion) { a.Compagnie = "" }), wantChamp: "compagnie"},
		{name: "doublon", avion: modifier(func(a *model.Avion) { a.Immatriculation = "f-exist" }), wantErr: apperr.ErrConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.avions["F-EXIST"] = model.Avion{Immatriculation: "F-EXIST"}
			svc := newTestService(repo)

			got, err := svc.CreerAvion(t.Context(), tt.avion)
			verifierErreur(t, err, tt.wantErr, tt.wantChamp)
			if err != nil {
				return
			}

			if got.Immatriculation != tt.wantImmat {
				t.Errorf("immatriculation = %q, attendu %q", got.Immatriculation, tt.wantImmat)
			}
			if !got.CreeLe.Equal(maintenant) {
				t.Errorf("CreeLe = %v, attendu %v", got.CreeLe, maintenant)
			}
			if _, ok := repo.avions[tt.wantImmat]; !ok {
				t.Error("l'avion n'a pas été enregistré")
			}
		})
	}
}

func TestCreerVol(t *testing.T) {
	modifier := func(f func(v *model.Vol)) model.Vol {
		v := volValide()
		f(&v)
		return v
	}

	tests := []struct {
		name            string
		immatriculation string
		vol             model.Vol
		wantErr         error
		wantChamp       string
		wantStatut      model.Statut
	}{
		{name: "statut par défaut", immatriculation: "F-GKXA", vol: volValide(), wantStatut: model.StatutProgramme},
		{name: "statut fourni", immatriculation: "F-GKXA",
			vol: modifier(func(v *model.Vol) { v.Statut = model.StatutEnCours }), wantStatut: model.StatutEnCours},
		{name: "immatriculation de l'URL normalisée", immatriculation: "f-gkxa", vol: volValide(), wantStatut: model.StatutProgramme},
		{name: "immatriculation du corps identique", immatriculation: "F-GKXA",
			vol: modifier(func(v *model.Vol) { v.Immatriculation = "f-gkxa" }), wantStatut: model.StatutProgramme},
		{name: "immatriculation du corps différente", immatriculation: "F-GKXA",
			vol: modifier(func(v *model.Vol) { v.Immatriculation = "F-HBXC" }), wantChamp: "immatriculation"},
		{name: "numéro vide", immatriculation: "F-GKXA", vol: modifier(func(v *model.Vol) { v.NumeroVol = "" }), wantChamp: "numero_vol"},
		{name: "numéro invalide", immatriculation: "F-GKXA", vol: modifier(func(v *model.Vol) { v.NumeroVol = "AF 12" }), wantChamp: "numero_vol"},
		{name: "départ vide", immatriculation: "F-GKXA", vol: modifier(func(v *model.Vol) { v.Depart = "" }), wantChamp: "aeroport_depart"},
		{name: "arrivée vide", immatriculation: "F-GKXA", vol: modifier(func(v *model.Vol) { v.Arrivee = "" }), wantChamp: "aeroport_arrivee"},
		{name: "même aéroport", immatriculation: "F-GKXA",
			vol: modifier(func(v *model.Vol) { v.Arrivee = "cdg" }), wantChamp: "aeroport_arrivee"},
		{name: "date vide", immatriculation: "F-GKXA", vol: modifier(func(v *model.Vol) { v.Date = "" }), wantChamp: "date"},
		{name: "date impossible", immatriculation: "F-GKXA", vol: modifier(func(v *model.Vol) { v.Date = "2026-02-30" }), wantChamp: "date"},
		{name: "date au mauvais format", immatriculation: "F-GKXA", vol: modifier(func(v *model.Vol) { v.Date = "01/12/2026" }), wantChamp: "date"},
		{name: "statut inconnu", immatriculation: "F-GKXA",
			vol: modifier(func(v *model.Vol) { v.Statut = "en_retard" }), wantChamp: "statut"},
		{name: "avion inconnu", immatriculation: "F-NOPE", vol: volValide(), wantErr: apperr.ErrNotFound},
		{name: "numéro déjà pris", immatriculation: "F-GKXA",
			vol: modifier(func(v *model.Vol) { v.NumeroVol = "AF0001" }), wantErr: apperr.ErrConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.avions["F-GKXA"] = model.Avion{Immatriculation: "F-GKXA"}
			repo.vols["AF0001"] = model.Vol{NumeroVol: "AF0001", Immatriculation: "F-GKXA"}
			svc := newTestService(repo)

			got, err := svc.CreerVol(t.Context(), tt.immatriculation, tt.vol)
			verifierErreur(t, err, tt.wantErr, tt.wantChamp)
			if err != nil {
				return
			}

			if got.Statut != tt.wantStatut {
				t.Errorf("statut = %q, attendu %q", got.Statut, tt.wantStatut)
			}
			if got.Immatriculation != "F-GKXA" {
				t.Errorf("immatriculation = %q, attendu %q", got.Immatriculation, "F-GKXA")
			}
			if !got.CreeLe.Equal(maintenant) {
				t.Errorf("CreeLe = %v, attendu %v", got.CreeLe, maintenant)
			}
		})
	}
}

func TestSupprimerAvion(t *testing.T) {
	tests := []struct {
		name            string
		immatriculation string
		wantErr         error
	}{
		{"sans vol", "F-HBXC", nil},
		{"sans vol, en minuscules", "f-hbxc", nil},
		{"avec des vols", "F-GKXA", apperr.ErrAvionAvecVols},
		{"inexistant", "F-NOPE", apperr.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.avions["F-GKXA"] = model.Avion{Immatriculation: "F-GKXA"}
			repo.avions["F-HBXC"] = model.Avion{Immatriculation: "F-HBXC"}
			repo.vols["AF1"] = model.Vol{NumeroVol: "AF1", Immatriculation: "F-GKXA"}
			svc := newTestService(repo)

			err := svc.SupprimerAvion(t.Context(), tt.immatriculation)
			verifierErreur(t, err, tt.wantErr, "")
			if err == nil && len(repo.avions) != 1 {
				t.Errorf("%d avion(s) restant(s), attendu 1", len(repo.avions))
			}
		})
	}
}

func TestVolsAvion(t *testing.T) {
	tests := []struct {
		name            string
		immatriculation string
		wantLen         int
		wantErr         error
	}{
		{"avec des vols", "F-GKXA", 2, nil},
		{"sans vol", "F-HBXC", 0, nil},
		{"avion inconnu", "F-NOPE", 0, apperr.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.avions["F-GKXA"] = model.Avion{Immatriculation: "F-GKXA"}
			repo.avions["F-HBXC"] = model.Avion{Immatriculation: "F-HBXC"}
			repo.vols["AF1"] = model.Vol{NumeroVol: "AF1", Immatriculation: "F-GKXA"}
			repo.vols["AF2"] = model.Vol{NumeroVol: "AF2", Immatriculation: "F-GKXA"}
			svc := newTestService(repo)

			vols, err := svc.VolsAvion(t.Context(), tt.immatriculation)
			verifierErreur(t, err, tt.wantErr, "")
			if len(vols) != tt.wantLen {
				t.Errorf("%d vol(s), attendu %d", len(vols), tt.wantLen)
			}
		})
	}
}

func TestStats(t *testing.T) {
	tests := []struct {
		name   string
		avions int
		vols   []model.Statut
		want   model.Stats
	}{
		{
			name: "base vide : tous les statuts à zéro",
			want: model.Stats{VolsParStatut: map[model.Statut]int{
				model.StatutProgramme: 0, model.StatutEnCours: 0, model.StatutAnnule: 0,
			}},
		},
		{
			name:   "flotte et vols",
			avions: 2,
			vols:   []model.Statut{model.StatutProgramme, model.StatutProgramme, model.StatutAnnule},
			want: model.Stats{NombreAvions: 2, NombreVols: 3, VolsParStatut: map[model.Statut]int{
				model.StatutProgramme: 2, model.StatutEnCours: 0, model.StatutAnnule: 1,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			for i := range tt.avions {
				immat := "F-" + string(rune('A'+i)) + "AAA"
				repo.avions[immat] = model.Avion{Immatriculation: immat}
			}
			for i, s := range tt.vols {
				numero := "AF" + string(rune('0'+i))
				repo.vols[numero] = model.Vol{NumeroVol: numero, Statut: s}
			}

			got, err := newTestService(repo).Stats(t.Context())
			if err != nil {
				t.Fatalf("Stats() : %v", err)
			}
			if got.NombreAvions != tt.want.NombreAvions || got.NombreVols != tt.want.NombreVols {
				t.Errorf("Stats() = %+v, attendu %+v", got, tt.want)
			}
			if len(got.VolsParStatut) != len(tt.want.VolsParStatut) {
				t.Fatalf("VolsParStatut = %v, attendu %v", got.VolsParStatut, tt.want.VolsParStatut)
			}
			for statut, n := range tt.want.VolsParStatut {
				if got.VolsParStatut[statut] != n {
					t.Errorf("VolsParStatut[%s] = %d, attendu %d", statut, got.VolsParStatut[statut], n)
				}
			}
		})
	}
}

func TestErreurDuRepositoryPropagee(t *testing.T) {
	errBase := errors.New("base indisponible")

	tests := []struct {
		name string
		call func(ctx context.Context, svc *Service) error
	}{
		{"ListerAvions", func(ctx context.Context, svc *Service) error {
			_, err := svc.ListerAvions(ctx)
			return err
		}},
		{"Avion", func(ctx context.Context, svc *Service) error {
			_, err := svc.Avion(ctx, "F-GKXA")
			return err
		}},
		{"CreerAvion", func(ctx context.Context, svc *Service) error {
			_, err := svc.CreerAvion(ctx, avionValide())
			return err
		}},
		{"SupprimerAvion", func(ctx context.Context, svc *Service) error {
			return svc.SupprimerAvion(ctx, "F-GKXA")
		}},
		{"CreerVol", func(ctx context.Context, svc *Service) error {
			_, err := svc.CreerVol(ctx, "F-GKXA", volValide())
			return err
		}},
		{"VolsAvion", func(ctx context.Context, svc *Service) error {
			_, err := svc.VolsAvion(ctx, "F-GKXA")
			return err
		}},
		{"Stats", func(ctx context.Context, svc *Service) error {
			_, err := svc.Stats(ctx)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.err = errBase

			err := tt.call(t.Context(), newTestService(repo))
			if !errors.Is(err, errBase) {
				t.Fatalf("erreur = %v, attendu une erreur enveloppant %v", err, errBase)
			}
			if errors.Is(err, apperr.ErrNotFound) || errors.Is(err, apperr.ErrInvalid) {
				t.Errorf("une panne technique ne doit pas devenir une erreur métier : %v", err)
			}
		})
	}
}
