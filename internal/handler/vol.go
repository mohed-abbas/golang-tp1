package handler

import (
	"net/http"

	"compagnie-aerienne/internal/model"
)

func (h *Handler) createVol(w http.ResponseWriter, r *http.Request) {
	var v model.Vol
	if err := decodeJSON(w, r, &v); err != nil {
		h.writeError(w, r, err)
		return
	}

	cree, err := h.svc.CreerVol(r.Context(), r.PathValue("immatriculation"), v)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, cree)
}

func (h *Handler) listVols(w http.ResponseWriter, r *http.Request) {
	vols, err := h.svc.VolsAvion(r.Context(), r.PathValue("immatriculation"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, vols)
}
