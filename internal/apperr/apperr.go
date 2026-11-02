// Package apperr définit les erreurs métier de l'application.
//
// Les couches basses les enveloppent avec fmt.Errorf et %w pour ajouter du
// contexte ; la couche HTTP les reconnaît avec errors.Is et errors.As afin de
// choisir le code de réponse approprié.
package apperr

import (
	"errors"
	"fmt"
)

// Erreurs sentinelles.
var (
	// ErrNotFound signale qu'une ressource demandée n'existe pas.
	ErrNotFound = errors.New("ressource introuvable")
	// ErrConflict signale qu'une ressource avec le même identifiant existe déjà.
	ErrConflict = errors.New("ressource déjà existante")
	// ErrAvionAvecVols signale qu'un avion ne peut pas être supprimé car des
	// vols y sont rattachés.
	ErrAvionAvecVols = errors.New("des vols sont associés à cet avion")
	// ErrInvalid signale des données d'entrée invalides.
	ErrInvalid = errors.New("données invalides")
)

// ValidationError décrit une donnée d'entrée invalide et le champ concerné.
type ValidationError struct {
	Champ   string
	Message string
}

// NewValidation crée une ValidationError pour le champ indiqué.
func NewValidation(champ, message string) *ValidationError {
	return &ValidationError{Champ: champ, Message: message}
}

// Error implémente l'interface error.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s : %s", e.Champ, e.Message)
}

// Unwrap rattache toute ValidationError à ErrInvalid, ce qui rend
// errors.Is(err, ErrInvalid) vrai.
func (e *ValidationError) Unwrap() error {
	return ErrInvalid
}
