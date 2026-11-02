// Package model définit les structures de données manipulées par l'API :
// avions, vols et statistiques de la flotte.
package model

import (
	"strings"
	"time"
)

// Horodatage regroupe les informations temporelles communes aux ressources.
// Il est embarqué dans Avion et Vol.
type Horodatage struct {
	CreeLe time.Time `json:"cree_le"`
}

// Avion représente un appareil de la flotte.
type Avion struct {
	Immatriculation string `json:"immatriculation"`
	Modele          string `json:"modele"`
	NombrePlaces    int    `json:"nombre_places"`
	Compagnie       string `json:"compagnie"`
	Horodatage
}

// Normaliser nettoie les champs saisis : espaces superflus retirés et
// immatriculation mise en majuscules.
func (a *Avion) Normaliser() {
	a.Immatriculation = NormaliserImmatriculation(a.Immatriculation)
	a.Modele = strings.TrimSpace(a.Modele)
	a.Compagnie = strings.TrimSpace(a.Compagnie)
}

// NormaliserImmatriculation renvoie la forme canonique d'une immatriculation.
func NormaliserImmatriculation(immatriculation string) string {
	return strings.ToUpper(strings.TrimSpace(immatriculation))
}
