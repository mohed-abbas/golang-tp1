package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"compagnie-aerienne/internal/apperr"
	"compagnie-aerienne/internal/model"
)

var creeLe = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)

// newTestStore ouvre une base SQLite neuve dans un dossier temporaire.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(t.Context(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open : %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("Close : %v", err)
		}
	})
	return st
}

func avionTest(immatriculation string) model.Avion {
	return model.Avion{
		Immatriculation: immatriculation,
		Modele:          "A320",
		NombrePlaces:    180,
		Compagnie:       "Air France",
		Horodatage:      model.Horodatage{CreeLe: creeLe},
	}
}

func volTest(numero, immatriculation, date string, statut model.Statut) model.Vol {
	return model.Vol{
		NumeroVol:       numero,
		Trajet:          model.Trajet{Depart: "CDG", Arrivee: "JFK"},
		Date:            date,
		Immatriculation: immatriculation,
		Statut:          statut,
		Horodatage:      model.Horodatage{CreeLe: creeLe},
	}
}

func mustCreateAvion(t *testing.T, st *Store, a model.Avion) {
	t.Helper()
	if err := st.CreateAvion(t.Context(), a); err != nil {
		t.Fatalf("CreateAvion(%s) : %v", a.Immatriculation, err)
	}
}

func mustCreateVol(t *testing.T, st *Store, v model.Vol) {
	t.Helper()
	if err := st.CreateVol(t.Context(), v); err != nil {
		t.Fatalf("CreateVol(%s) : %v", v.NumeroVol, err)
	}
}

func TestOpenIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	for i := range 2 {
		st, err := Open(t.Context(), path)
		if err != nil {
			t.Fatalf("ouverture n°%d : %v", i+1, err)
		}
		if err := st.Close(); err != nil {
			t.Fatalf("fermeture n°%d : %v", i+1, err)
		}
	}
}

func TestGetAvion(t *testing.T) {
	st := newTestStore(t)
	mustCreateAvion(t, st, avionTest("F-GKXA"))

	tests := []struct {
		name            string
		immatriculation string
		wantErr         error
	}{
		{"existant", "F-GKXA", nil},
		{"inexistant", "F-NOPE", apperr.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := st.GetAvion(t.Context(), tt.immatriculation)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetAvion() erreur = %v, attendu %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && got != avionTest(tt.immatriculation) {
				t.Errorf("GetAvion() = %+v, attendu %+v", got, avionTest(tt.immatriculation))
			}
		})
	}
}

func TestCreateAvion(t *testing.T) {
	tests := []struct {
		name    string
		avion   model.Avion
		wantErr error
	}{
		{"nouvel avion", avionTest("F-HBXC"), nil},
		{"immatriculation déjà prise", avionTest("F-GKXA"), apperr.ErrConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newTestStore(t)
			mustCreateAvion(t, st, avionTest("F-GKXA"))

			if err := st.CreateAvion(t.Context(), tt.avion); !errors.Is(err, tt.wantErr) {
				t.Errorf("CreateAvion() erreur = %v, attendu %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteAvion(t *testing.T) {
	tests := []struct {
		name            string
		immatriculation string
		wantErr         error
	}{
		{"sans vol", "F-HBXC", nil},
		{"avec des vols", "F-GKXA", apperr.ErrAvionAvecVols},
		{"inexistant", "F-NOPE", apperr.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newTestStore(t)
			mustCreateAvion(t, st, avionTest("F-GKXA"))
			mustCreateAvion(t, st, avionTest("F-HBXC"))
			mustCreateVol(t, st, volTest("AF1", "F-GKXA", "2026-12-01", model.StatutProgramme))

			err := st.DeleteAvion(t.Context(), tt.immatriculation)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("DeleteAvion() erreur = %v, attendu %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if _, err := st.GetAvion(t.Context(), tt.immatriculation); !errors.Is(err, apperr.ErrNotFound) {
					t.Errorf("l'avion existe encore après suppression (erreur = %v)", err)
				}
			}
		})
	}
}

func TestCreateVol(t *testing.T) {
	tests := []struct {
		name    string
		vol     model.Vol
		wantErr error
	}{
		{"nouveau vol", volTest("AF2", "F-GKXA", "2026-12-02", model.StatutEnCours), nil},
		{"numéro déjà pris", volTest("AF1", "F-GKXA", "2026-12-03", model.StatutProgramme), apperr.ErrConflict},
		{"avion inconnu", volTest("AF3", "F-NOPE", "2026-12-03", model.StatutProgramme), apperr.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newTestStore(t)
			mustCreateAvion(t, st, avionTest("F-GKXA"))
			mustCreateVol(t, st, volTest("AF1", "F-GKXA", "2026-12-01", model.StatutProgramme))

			if err := st.CreateVol(t.Context(), tt.vol); !errors.Is(err, tt.wantErr) {
				t.Errorf("CreateVol() erreur = %v, attendu %v", err, tt.wantErr)
			}
		})
	}
}

func TestListes(t *testing.T) {
	st := newTestStore(t)

	avions, err := st.ListAvions(t.Context())
	if err != nil {
		t.Fatalf("ListAvions() sur base vide : %v", err)
	}
	if avions == nil || len(avions) != 0 {
		t.Fatalf("ListAvions() sur base vide = %#v, attendu une liste vide non nil", avions)
	}

	mustCreateAvion(t, st, avionTest("F-HBXC"))
	mustCreateAvion(t, st, avionTest("F-GKXA"))
	mustCreateVol(t, st, volTest("AF2", "F-GKXA", "2026-12-02", model.StatutProgramme))
	mustCreateVol(t, st, volTest("AF1", "F-GKXA", "2026-12-01", model.StatutAnnule))

	tests := []struct {
		name string
		list func() ([]string, error)
		want []string
	}{
		{"avions triés par immatriculation", func() ([]string, error) {
			avions, err := st.ListAvions(t.Context())
			ids := make([]string, 0, len(avions))
			for _, a := range avions {
				ids = append(ids, a.Immatriculation)
			}
			return ids, err
		}, []string{"F-GKXA", "F-HBXC"}},
		{"vols triés par date", func() ([]string, error) {
			vols, err := st.ListVolsByAvion(t.Context(), "F-GKXA")
			ids := make([]string, 0, len(vols))
			for _, v := range vols {
				ids = append(ids, v.NumeroVol)
			}
			return ids, err
		}, []string{"AF1", "AF2"}},
		{"avion sans vol", func() ([]string, error) {
			vols, err := st.ListVolsByAvion(t.Context(), "F-HBXC")
			ids := make([]string, 0, len(vols))
			for _, v := range vols {
				ids = append(ids, v.NumeroVol)
			}
			return ids, err
		}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.list()
			if err != nil {
				t.Fatalf("erreur inattendue : %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("obtenu %v, attendu %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("obtenu %v, attendu %v", got, tt.want)
					break
				}
			}
		})
	}
}

func TestCompteurs(t *testing.T) {
	st := newTestStore(t)
	mustCreateAvion(t, st, avionTest("F-GKXA"))
	mustCreateAvion(t, st, avionTest("F-HBXC"))
	mustCreateVol(t, st, volTest("AF1", "F-GKXA", "2026-12-01", model.StatutProgramme))
	mustCreateVol(t, st, volTest("AF2", "F-GKXA", "2026-12-02", model.StatutProgramme))
	mustCreateVol(t, st, volTest("AF3", "F-HBXC", "2026-12-03", model.StatutAnnule))

	t.Run("nombre d'avions", func(t *testing.T) {
		n, err := st.CountAvions(t.Context())
		if err != nil || n != 2 {
			t.Errorf("CountAvions() = %d, %v ; attendu 2, nil", n, err)
		}
	})

	t.Run("vols par statut", func(t *testing.T) {
		got, err := st.CountVolsParStatut(t.Context())
		if err != nil {
			t.Fatalf("CountVolsParStatut() : %v", err)
		}
		want := map[model.Statut]int{model.StatutProgramme: 2, model.StatutAnnule: 1}
		if len(got) != len(want) || got[model.StatutProgramme] != 2 || got[model.StatutAnnule] != 1 {
			t.Errorf("CountVolsParStatut() = %v, attendu %v", got, want)
		}
	})

	volsParAvion := []struct {
		immatriculation string
		want            int
	}{
		{"F-GKXA", 2},
		{"F-HBXC", 1},
		{"F-NOPE", 0},
	}
	for _, tt := range volsParAvion {
		t.Run("vols de "+tt.immatriculation, func(t *testing.T) {
			n, err := st.CountVolsByAvion(t.Context(), tt.immatriculation)
			if err != nil || n != tt.want {
				t.Errorf("CountVolsByAvion() = %d, %v ; attendu %d, nil", n, err, tt.want)
			}
		})
	}

	existence := []struct {
		numero string
		want   bool
	}{
		{"AF1", true},
		{"AF999", false},
	}
	for _, tt := range existence {
		t.Run("existence de "+tt.numero, func(t *testing.T) {
			existe, err := st.VolExiste(t.Context(), tt.numero)
			if err != nil || existe != tt.want {
				t.Errorf("VolExiste() = %v, %v ; attendu %v, nil", existe, err, tt.want)
			}
		})
	}
}
