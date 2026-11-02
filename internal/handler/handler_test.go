package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"compagnie-aerienne/internal/apperr"
	"compagnie-aerienne/internal/model"
	"compagnie-aerienne/internal/service"
	"compagnie-aerienne/internal/store"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatalf("store.Open : %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("Close : %v", err)
		}
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(service.New(st), logger)
}

func executer(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// etape décrit une requête du scénario et la réponse attendue.
type etape struct {
	name       string
	method     string
	path       string
	body       string
	wantStatus int
	wantChamps map[string]any // champs attendus dans l'objet JSON renvoyé
	wantListe  int            // taille attendue si la réponse est un tableau JSON
	estListe   bool
}

func TestScenarioAPI(t *testing.T) {
	routes := newTestHandler(t).Routes()

	const (
		avionGKXA = `{"immatriculation":"f-gkxa","modele":"A320","nombre_places":180,"compagnie":"Air France"}`
		avionHBXC = `{"immatriculation":"F-HBXC","modele":"B737","nombre_places":189,"compagnie":"Transavia"}`
		volAF1    = `{"numero_vol":"AF1","aeroport_depart":"CDG","aeroport_arrivee":"JFK","date":"2026-12-01"}`
	)

	etapes := []etape{
		{name: "flotte vide", method: "GET", path: "/avions", wantStatus: http.StatusOK, estListe: true, wantListe: 0},
		{name: "création d'un avion", method: "POST", path: "/avions", body: avionGKXA, wantStatus: http.StatusCreated,
			wantChamps: map[string]any{"immatriculation": "F-GKXA", "nombre_places": 180.0, "compagnie": "Air France"}},
		{name: "doublon", method: "POST", path: "/avions", body: avionGKXA, wantStatus: http.StatusConflict},
		{name: "zéro place", method: "POST", path: "/avions", wantStatus: http.StatusBadRequest,
			body:       `{"immatriculation":"F-TEST","modele":"A320","nombre_places":0,"compagnie":"X"}`,
			wantChamps: map[string]any{"champ": "nombre_places"}},
		{name: "champ inconnu", method: "POST", path: "/avions", body: `{"couleur":"rouge"}`,
			wantStatus: http.StatusBadRequest, wantChamps: map[string]any{"champ": "couleur"}},
		{name: "JSON invalide", method: "POST", path: "/avions", body: `{"modele":`,
			wantStatus: http.StatusBadRequest, wantChamps: map[string]any{"champ": "body"}},
		{name: "corps vide", method: "POST", path: "/avions", body: "",
			wantStatus: http.StatusBadRequest, wantChamps: map[string]any{"champ": "body"}},
		{name: "détail d'un avion", method: "GET", path: "/avions/F-GKXA", wantStatus: http.StatusOK,
			wantChamps: map[string]any{"modele": "A320"}},
		{name: "détail en minuscules", method: "GET", path: "/avions/f-gkxa", wantStatus: http.StatusOK,
			wantChamps: map[string]any{"immatriculation": "F-GKXA"}},
		{name: "avion inconnu", method: "GET", path: "/avions/F-NOPE", wantStatus: http.StatusNotFound},
		{name: "création d'un vol", method: "POST", path: "/avions/F-GKXA/vols", body: volAF1, wantStatus: http.StatusCreated,
			wantChamps: map[string]any{"numero_vol": "AF1", "statut": "programme", "immatriculation": "F-GKXA", "aeroport_depart": "CDG"}},
		{name: "vol en double", method: "POST", path: "/avions/F-GKXA/vols", body: volAF1, wantStatus: http.StatusConflict},
		{name: "vol avec date invalide", method: "POST", path: "/avions/F-GKXA/vols", wantStatus: http.StatusBadRequest,
			body:       `{"numero_vol":"AF2","aeroport_depart":"CDG","aeroport_arrivee":"NCE","date":"2026-13-01"}`,
			wantChamps: map[string]any{"champ": "date"}},
		{name: "vol pour un avion inconnu", method: "POST", path: "/avions/F-NOPE/vols", body: volAF1, wantStatus: http.StatusNotFound},
		{name: "vol annulé", method: "POST", path: "/avions/F-GKXA/vols", wantStatus: http.StatusCreated,
			body:       `{"numero_vol":"AF2","aeroport_depart":"CDG","aeroport_arrivee":"NCE","date":"2026-12-02","statut":"annule"}`,
			wantChamps: map[string]any{"statut": "annule"}},
		{name: "vols d'un avion", method: "GET", path: "/avions/F-GKXA/vols", wantStatus: http.StatusOK, estListe: true, wantListe: 2},
		{name: "vols d'un avion inconnu", method: "GET", path: "/avions/F-NOPE/vols", wantStatus: http.StatusNotFound},
		{name: "statistiques", method: "GET", path: "/stats", wantStatus: http.StatusOK,
			wantChamps: map[string]any{"nombre_avions": 1.0, "nombre_vols": 2.0,
				"vols_par_statut": map[string]any{"programme": 1.0, "en_cours": 0.0, "annule": 1.0}}},
		{name: "suppression refusée (vols liés)", method: "DELETE", path: "/avions/F-GKXA", wantStatus: http.StatusConflict},
		{name: "création d'un second avion", method: "POST", path: "/avions", body: avionHBXC, wantStatus: http.StatusCreated},
		{name: "suppression autorisée", method: "DELETE", path: "/avions/F-HBXC", wantStatus: http.StatusNoContent},
		{name: "suppression répétée", method: "DELETE", path: "/avions/F-HBXC", wantStatus: http.StatusNotFound},
		{name: "méthode non autorisée", method: "PUT", path: "/stats", wantStatus: http.StatusMethodNotAllowed},
		{name: "route inconnue", method: "GET", path: "/aeroports", wantStatus: http.StatusNotFound},
		{name: "flotte finale", method: "GET", path: "/avions", wantStatus: http.StatusOK, estListe: true, wantListe: 1},
	}

	for _, e := range etapes {
		t.Run(e.name, func(t *testing.T) {
			rec := executer(routes, e.method, e.path, e.body)
			verifierReponse(t, rec, e)
		})
	}
}

func verifierReponse(t *testing.T, rec *httptest.ResponseRecorder, e etape) {
	t.Helper()

	if rec.Code != e.wantStatus {
		t.Fatalf("code = %d, attendu %d (corps : %s)", rec.Code, e.wantStatus, rec.Body)
	}
	if e.wantStatus == http.StatusNoContent {
		if rec.Body.Len() != 0 {
			t.Errorf("corps = %q, attendu vide pour un 204", rec.Body)
		}
		return
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, attendu application/json", ct)
	}

	if e.estListe {
		var liste []any
		if err := json.Unmarshal(rec.Body.Bytes(), &liste); err != nil {
			t.Fatalf("corps non décodable en tableau : %v (%s)", err, rec.Body)
		}
		if liste == nil || len(liste) != e.wantListe {
			t.Errorf("tableau de %d élément(s), attendu %d (%s)", len(liste), e.wantListe, rec.Body)
		}
		return
	}

	var objet map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &objet); err != nil {
		t.Fatalf("corps non décodable en objet : %v (%s)", err, rec.Body)
	}
	if e.wantStatus >= 400 {
		if msg, ok := objet["erreur"].(string); !ok || msg == "" {
			t.Errorf("champ \"erreur\" absent ou vide : %s", rec.Body)
		}
	}
	for cle, want := range e.wantChamps {
		if got := objet[cle]; !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %#v, attendu %#v", cle, got, want)
		}
	}
}

func TestCreationRenvoieLocation(t *testing.T) {
	routes := newTestHandler(t).Routes()

	rec := executer(routes, "POST", "/avions",
		`{"immatriculation":"f-gkxa","modele":"A320","nombre_places":180,"compagnie":"Air France"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d, attendu 201 (%s)", rec.Code, rec.Body)
	}
	if loc := rec.Header().Get("Location"); loc != "/avions/F-GKXA" {
		t.Errorf("Location = %q, attendu /avions/F-GKXA", loc)
	}

	var avion model.Avion
	if err := json.NewDecoder(rec.Body).Decode(&avion); err != nil {
		t.Fatalf("décodage de la réponse : %v", err)
	}
	if avion.CreeLe.IsZero() {
		t.Error("cree_le n'est pas renseigné")
	}
}

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantChamp string
		wantTrop  bool
	}{
		{name: "objet valide", body: `{"modele":"A320"}`},
		{name: "corps vide", body: "", wantChamp: "body"},
		{name: "JSON incomplet", body: `{"modele":`, wantChamp: "body"},
		{name: "erreur de syntaxe", body: `{modele}`, wantChamp: "body"},
		{name: "mauvais type", body: `{"nombre_places":"beaucoup"}`, wantChamp: "nombre_places"},
		{name: "champ inconnu", body: `{"couleur":"rouge"}`, wantChamp: "couleur"},
		{name: "deux objets", body: `{}{}`, wantChamp: "body"},
		{name: "tableau au lieu d'un objet", body: `[]`, wantChamp: "body"},
		{name: "corps trop volumineux", body: `{"modele":"` + strings.Repeat("a", maxBodyBytes) + `"}`, wantTrop: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/avions", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			var avion model.Avion
			err := decodeJSON(rec, req, &avion)

			var (
				ve       *apperr.ValidationError
				tropGros *http.MaxBytesError
			)
			switch {
			case tt.wantTrop:
				if !errors.As(err, &tropGros) {
					t.Errorf("erreur = %v, attendu *http.MaxBytesError", err)
				}
			case tt.wantChamp != "":
				if !errors.As(err, &ve) || ve.Champ != tt.wantChamp {
					t.Errorf("erreur = %v, attendu une ValidationError sur %q", err, tt.wantChamp)
				}
			case err != nil:
				t.Errorf("erreur inattendue : %v", err)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	h := newTestHandler(t)

	tests := []struct {
		name       string
		status     int
		valeur     any
		wantStatus int
		wantBody   string
	}{
		{"objet", http.StatusOK, map[string]int{"a": 1}, http.StatusOK, `{"a":1}`},
		{"création", http.StatusCreated, errorResponse{Erreur: "x"}, http.StatusCreated, `{"erreur":"x"}`},
		{"champ omis si vide", http.StatusBadRequest, errorResponse{Erreur: "x", Champ: ""}, http.StatusBadRequest, `{"erreur":"x"}`},
		{"valeur non encodable", http.StatusOK, make(chan int), http.StatusInternalServerError, `{"erreur":"erreur interne du serveur"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.writeJSON(rec, tt.status, tt.valeur)

			if rec.Code != tt.wantStatus {
				t.Errorf("code = %d, attendu %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, attendu application/json", ct)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Errorf("corps = %s, attendu %s", got, tt.wantBody)
			}
		})
	}
}

func TestRecoverPanic(t *testing.T) {
	h := newTestHandler(t)

	tests := []struct {
		name   string
		valeur any
	}{
		{"panique avec une chaîne", "boum"},
		{"panique avec une erreur", errors.New("boum")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paniqueur := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(tt.valeur)
			})

			rec := executer(h.recoverPanic(paniqueur), http.MethodGet, "/", "")
			verifierReponse(t, rec, etape{wantStatus: http.StatusInternalServerError})
		})
	}
}

func TestCorpsTropVolumineux(t *testing.T) {
	routes := newTestHandler(t).Routes()

	body := `{"modele":"` + strings.Repeat("a", 2*maxBodyBytes) + `"}`
	rec := executer(routes, http.MethodPost, "/avions", body)
	verifierReponse(t, rec, etape{wantStatus: http.StatusRequestEntityTooLarge})
}
