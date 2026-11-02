package model

import "strings"

// FormatDate est le format attendu pour la date d'un vol (YYYY-MM-DD).
const FormatDate = "2006-01-02"

// Statut représente l'état d'un vol.
type Statut string

// Valeurs possibles d'un Statut.
const (
	StatutProgramme Statut = "programme"
	StatutEnCours   Statut = "en_cours"
	StatutAnnule    Statut = "annule"
)

// Statuts renvoie la liste des statuts autorisés, dans un ordre stable.
func Statuts() []Statut {
	return []Statut{StatutProgramme, StatutEnCours, StatutAnnule}
}

// Valide indique si le statut fait partie des valeurs autorisées.
func (s Statut) Valide() bool {
	switch s {
	case StatutProgramme, StatutEnCours, StatutAnnule:
		return true
	}
	return false
}

// Trajet décrit les aéroports de départ et d'arrivée d'un vol.
type Trajet struct {
	Depart  string `json:"aeroport_depart"`
	Arrivee string `json:"aeroport_arrivee"`
}

// Boucle indique si le départ et l'arrivée désignent le même aéroport.
func (t Trajet) Boucle() bool {
	return strings.EqualFold(t.Depart, t.Arrivee)
}

// Vol représente un vol assuré par un avion de la flotte.
type Vol struct {
	NumeroVol string `json:"numero_vol"`
	Trajet
	Date            string `json:"date"`
	Immatriculation string `json:"immatriculation"`
	Statut          Statut `json:"statut"`
	Horodatage
}

// Normaliser nettoie les champs saisis : espaces superflus retirés,
// numéro de vol et immatriculation mis en majuscules.
func (v *Vol) Normaliser() {
	v.NumeroVol = strings.ToUpper(strings.TrimSpace(v.NumeroVol))
	v.Depart = strings.TrimSpace(v.Depart)
	v.Arrivee = strings.TrimSpace(v.Arrivee)
	v.Date = strings.TrimSpace(v.Date)
	v.Immatriculation = NormaliserImmatriculation(v.Immatriculation)
	v.Statut = Statut(strings.TrimSpace(string(v.Statut)))
}
