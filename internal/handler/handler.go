// Package handler expose la logique métier via HTTP : routage, décodage et
// encodage JSON, et traduction des erreurs métier en codes de réponse.
package handler

import (
	"log/slog"
	"net/http"

	"compagnie-aerienne/internal/service"
)

// Handler regroupe les dépendances des handlers HTTP.
type Handler struct {
	svc    *service.Service
	logger *slog.Logger
}

// New crée un Handler.
func New(svc *service.Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// Routes construit le routeur de l'API, middlewares compris.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /avions", h.listAvions)
	mux.HandleFunc("POST /avions", h.createAvion)
	mux.HandleFunc("GET /avions/{immatriculation}", h.getAvion)
	mux.HandleFunc("DELETE /avions/{immatriculation}", h.deleteAvion)
	mux.HandleFunc("POST /avions/{immatriculation}/vols", h.createVol)
	mux.HandleFunc("GET /avions/{immatriculation}/vols", h.listVols)
	mux.HandleFunc("GET /stats", h.stats)

	return h.logRequests(h.recoverPanic(h.jsonFallback(mux)))
}
