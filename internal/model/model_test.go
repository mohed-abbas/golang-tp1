package model

import (
	"testing"
	"time"
)

func TestStatutValide(t *testing.T) {
	tests := []struct {
		name   string
		statut Statut
		want   bool
	}{
		{"programme", StatutProgramme, true},
		{"en cours", StatutEnCours, true},
		{"annulé", StatutAnnule, true},
		{"vide", "", false},
		{"inconnu", "en_retard", false},
		{"majuscules", "PROGRAMME", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.statut.Valide(); got != tt.want {
				t.Errorf("Statut(%q).Valide() = %v, attendu %v", tt.statut, got, tt.want)
			}
		})
	}
}

func TestStatuts(t *testing.T) {
	statuts := Statuts()
	if len(statuts) != 3 {
		t.Fatalf("Statuts() renvoie %d statuts, attendu 3", len(statuts))
	}
	for _, s := range statuts {
		t.Run(string(s), func(t *testing.T) {
			if !s.Valide() {
				t.Errorf("le statut %q renvoyé par Statuts() n'est pas valide", s)
			}
		})
	}
}

func TestTrajetBoucle(t *testing.T) {
	tests := []struct {
		name   string
		trajet Trajet
		want   bool
	}{
		{"aéroports différents", Trajet{Depart: "CDG", Arrivee: "JFK"}, false},
		{"même aéroport", Trajet{Depart: "CDG", Arrivee: "CDG"}, true},
		{"même aéroport, casse différente", Trajet{Depart: "cdg", Arrivee: "CDG"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.trajet.Boucle(); got != tt.want {
				t.Errorf("%+v.Boucle() = %v, attendu %v", tt.trajet, got, tt.want)
			}
		})
	}
}

func TestAvionNormaliser(t *testing.T) {
	creeLe := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		in   Avion
		want Avion
	}{
		{
			name: "espaces et minuscules",
			in:   Avion{Immatriculation: "  f-gkxa ", Modele: " A320 ", NombrePlaces: 180, Compagnie: " Air France "},
			want: Avion{Immatriculation: "F-GKXA", Modele: "A320", NombrePlaces: 180, Compagnie: "Air France"},
		},
		{
			name: "déjà normalisé, horodatage conservé",
			in:   Avion{Immatriculation: "F-HBXC", Modele: "B737", Compagnie: "Transavia", Horodatage: Horodatage{CreeLe: creeLe}},
			want: Avion{Immatriculation: "F-HBXC", Modele: "B737", Compagnie: "Transavia", Horodatage: Horodatage{CreeLe: creeLe}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in
			got.Normaliser()
			if got != tt.want {
				t.Errorf("Normaliser() = %+v, attendu %+v", got, tt.want)
			}
		})
	}
}

func TestVolNormaliser(t *testing.T) {
	tests := []struct {
		name string
		in   Vol
		want Vol
	}{
		{
			name: "espaces et minuscules",
			in: Vol{NumeroVol: " af1234 ", Trajet: Trajet{Depart: " CDG ", Arrivee: " JFK "},
				Date: " 2026-12-01 ", Immatriculation: " f-gkxa ", Statut: " annule "},
			want: Vol{NumeroVol: "AF1234", Trajet: Trajet{Depart: "CDG", Arrivee: "JFK"},
				Date: "2026-12-01", Immatriculation: "F-GKXA", Statut: StatutAnnule},
		},
		{
			name: "les aéroports gardent leur casse",
			in:   Vol{NumeroVol: "AF1", Trajet: Trajet{Depart: "Paris CDG", Arrivee: "Nice"}},
			want: Vol{NumeroVol: "AF1", Trajet: Trajet{Depart: "Paris CDG", Arrivee: "Nice"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in
			got.Normaliser()
			if got != tt.want {
				t.Errorf("Normaliser() = %+v, attendu %+v", got, tt.want)
			}
		})
	}
}
