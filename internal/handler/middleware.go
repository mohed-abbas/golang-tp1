package handler

import (
	"net/http"
	"runtime/debug"
	"time"
)

// statusRecorder mémorise le code HTTP envoyé, pour la journalisation.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader enregistre le code avant de le transmettre.
func (rec *statusRecorder) WriteHeader(status int) {
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}

// Unwrap donne accès au ResponseWriter d'origine (http.ResponseController).
func (rec *statusRecorder) Unwrap() http.ResponseWriter {
	return rec.ResponseWriter
}

// logRequests journalise chaque requête avec son code de réponse et sa durée.
func (h *Handler) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debut := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		h.logger.Info("requête",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duree", time.Since(debut))
	})
}

// muetWriter retient le code envoyé par http.ServeMux sans transmettre son
// corps en texte brut.
type muetWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader mémorise le code sans l'envoyer.
func (mw *muetWriter) WriteHeader(status int) {
	mw.status = status
}

// Write ignore le corps produit par http.ServeMux.
func (mw *muetWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

// jsonFallback répond en JSON aux routes inconnues (404) et aux méthodes non
// autorisées (405), à la place des réponses texte de http.ServeMux.
func (h *Handler) jsonFallback(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}

		mw := &muetWriter{ResponseWriter: w, status: http.StatusNotFound}
		mux.ServeHTTP(mw, r)

		message := "route introuvable"
		if mw.status == http.StatusMethodNotAllowed {
			message = "méthode non autorisée"
		}
		h.writeJSON(w, mw.status, errorResponse{Erreur: message})
	})
}

// recoverPanic transforme une panique inattendue en réponse 500 JSON au lieu
// d'interrompre la connexion.
func (h *Handler) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				h.logger.Error("panique récupérée", "panic", v, "stack", string(debug.Stack()))
				h.writeJSON(w, http.StatusInternalServerError, errorResponse{Erreur: "erreur interne du serveur"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
