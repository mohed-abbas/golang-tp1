package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestValidationError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantMsg   string
		wantChamp string
	}{
		{
			name:      "directe",
			err:       NewValidation("date", "format attendu YYYY-MM-DD"),
			wantMsg:   "date : format attendu YYYY-MM-DD",
			wantChamp: "date",
		},
		{
			name:      "enveloppée avec %w",
			err:       fmt.Errorf("création du vol : %w", NewValidation("statut", "inconnu")),
			wantMsg:   "création du vol : statut : inconnu",
			wantChamp: "statut",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, attendu %q", got, tt.wantMsg)
			}
			if !errors.Is(tt.err, ErrInvalid) {
				t.Error("errors.Is(err, ErrInvalid) = false, attendu true")
			}

			var ve *ValidationError
			if !errors.As(tt.err, &ve) {
				t.Fatal("errors.As(err, *ValidationError) = false, attendu true")
			}
			if ve.Champ != tt.wantChamp {
				t.Errorf("Champ = %q, attendu %q", ve.Champ, tt.wantChamp)
			}
		})
	}
}

func TestSentinelles(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{"enveloppée une fois", fmt.Errorf("avion %q : %w", "F-GKXA", ErrNotFound), ErrNotFound, true},
		{"enveloppée deux fois", fmt.Errorf("création : %w", fmt.Errorf("vol : %w", ErrConflict)), ErrConflict, true},
		{"sentinelle différente", fmt.Errorf("avion : %w", ErrNotFound), ErrConflict, false},
		{"%v perd la cause", fmt.Errorf("avion : %v", ErrNotFound), ErrNotFound, false},
		{"validation n'est pas une absence", NewValidation("date", "obligatoire"), ErrNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is(%v, %v) = %v, attendu %v", tt.err, tt.target, got, tt.want)
			}
		})
	}
}
