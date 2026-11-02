package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"compagnie-aerienne/internal/apperr"
)

// maxBodyBytes limite la taille des corps de requête acceptés (1 Mo).
const maxBodyBytes = 1 << 20

// errorResponse est le corps JSON renvoyé en cas d'erreur.
type errorResponse struct {
	Erreur string `json:"erreur"`
	Champ  string `json:"champ,omitempty"`
}

// writeJSON encode v en JSON et l'envoie avec le code status.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		h.logger.Error("encodage JSON", "err", err)
		status = http.StatusInternalServerError
		body = []byte(`{"erreur":"erreur interne du serveur"}`)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(append(body, '\n')); err != nil {
		h.logger.Warn("écriture de la réponse", "err", err)
	}
}

// writeError traduit err en réponse JSON avec le code HTTP correspondant.
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		validationErr *apperr.ValidationError
		tropGrosErr   *http.MaxBytesError
	)

	switch {
	case errors.As(err, &validationErr):
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Erreur: validationErr.Error(), Champ: validationErr.Champ})
	case errors.As(err, &tropGrosErr):
		h.writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{
			Erreur: fmt.Sprintf("corps de requête trop volumineux (%d octets maximum)", tropGrosErr.Limit),
		})
	case errors.Is(err, apperr.ErrNotFound):
		h.writeJSON(w, http.StatusNotFound, errorResponse{Erreur: err.Error()})
	case errors.Is(err, apperr.ErrConflict), errors.Is(err, apperr.ErrAvionAvecVols):
		h.writeJSON(w, http.StatusConflict, errorResponse{Erreur: err.Error()})
	default:
		h.logger.Error("erreur interne", "method", r.Method, "path", r.URL.Path, "err", err)
		h.writeJSON(w, http.StatusInternalServerError, errorResponse{Erreur: "erreur interne du serveur"})
	}
}

// decodeJSON lit un unique objet JSON depuis le corps de la requête dans dst.
// Le corps est limité à maxBodyBytes et les champs inconnus sont refusés.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return erreurDecodage(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var tropGrosErr *http.MaxBytesError
		if errors.As(err, &tropGrosErr) {
			return fmt.Errorf("corps de requête : %w", err)
		}
		return apperr.NewValidation("body", "un seul objet JSON est attendu")
	}
	return nil
}

// erreurDecodage convertit une erreur de encoding/json en erreur métier.
func erreurDecodage(err error) error {
	var (
		syntaxErr   *json.SyntaxError
		typeErr     *json.UnmarshalTypeError
		tropGrosErr *http.MaxBytesError
	)

	switch {
	case errors.As(err, &tropGrosErr):
		return fmt.Errorf("corps de requête : %w", err)
	case errors.Is(err, io.EOF):
		return apperr.NewValidation("body", "corps de requête vide")
	case errors.Is(err, io.ErrUnexpectedEOF):
		return apperr.NewValidation("body", "JSON incomplet")
	case errors.As(err, &syntaxErr):
		return apperr.NewValidation("body", fmt.Sprintf("JSON invalide (position %d)", syntaxErr.Offset))
	case errors.As(err, &typeErr):
		champ := typeErr.Field
		if champ == "" {
			champ = "body"
		}
		return apperr.NewValidation(champ, fmt.Sprintf("type attendu : %s", typeErr.Type))
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		champ := strings.Trim(strings.TrimPrefix(err.Error(), "json: unknown field "), `"`)
		return apperr.NewValidation(champ, "champ inconnu")
	}
	return apperr.NewValidation("body", "JSON invalide")
}
