package handler

import (
	"net/http"
	"net/url"

	"compagnie-aerienne/internal/model"
)

func (h *Handler) listAvions(w http.ResponseWriter, r *http.Request) {
	avions, err := h.svc.ListerAvions(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, avions)
}

func (h *Handler) createAvion(w http.ResponseWriter, r *http.Request) {
	var a model.Avion
	if err := decodeJSON(w, r, &a); err != nil {
		h.writeError(w, r, err)
		return
	}

	cree, err := h.svc.CreerAvion(r.Context(), a)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/avions/"+url.PathEscape(cree.Immatriculation))
	h.writeJSON(w, http.StatusCreated, cree)
}

func (h *Handler) getAvion(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.Avion(r.Context(), r.PathValue("immatriculation"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, a)
}

func (h *Handler) deleteAvion(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SupprimerAvion(r.Context(), r.PathValue("immatriculation")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
