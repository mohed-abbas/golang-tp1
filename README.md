# API de gestion de compagnie aérienne

API HTTP écrite en Go pour gérer la flotte d'avions d'une compagnie et ses vols.

- Serveur HTTP de la bibliothèque standard (`net/http`), sans framework ni routeur externe
- Persistance SQLite avec le pilote pur Go [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) (aucune dépendance C)
- Conteneurisée avec Docker, port **8080**

---

## Démarrage rapide (Docker)

**Prérequis :** Docker avec Docker Compose v2.

```bash
docker compose up --build -d
```

L'API est disponible sur <http://localhost:8080>. Vérification :

```bash
curl http://localhost:8080/stats
```

```json
{"nombre_avions":0,"nombre_vols":0,"vols_par_statut":{"annule":0,"en_cours":0,"programme":0}}
```

| Action | Commande |
|---|---|
| Suivre les logs | `docker compose logs -f api` |
| Arrêter (données conservées) | `docker compose down` |
| Arrêter et effacer les données | `docker compose down -v` |

La base SQLite est stockée dans le volume Docker `db-data` : les données survivent aux redémarrages du conteneur.

### Le port 8080 est déjà utilisé ?

Le port publié sur la machine hôte se change avec la variable `API_PORT` (le conteneur écoute toujours sur 8080) :

```bash
API_PORT=18080 docker compose up --build -d
```

Il est aussi possible de créer un fichier `.env` à la racine contenant `API_PORT=18080` ; Docker Compose le lit automatiquement.

---

## Démarrage sans Docker

**Prérequis :** Go 1.26 ou plus récent.

```bash
go run ./cmd/api
```

La base `compagnie.db` est créée dans le dossier courant.

### Configuration

| Variable | Défaut | Rôle |
|---|---|---|
| `ADDR` | `:8080` | Adresse d'écoute du serveur |
| `DB_PATH` | `compagnie.db` | Chemin du fichier SQLite (`/data/compagnie.db` dans Docker) |
| `API_PORT` | `8080` | Port publié sur l'hôte (Docker Compose uniquement) |

---

## Endpoints

| Méthode | Route | Description | Succès | Erreurs possibles |
|---|---|---|---|---|
| `GET` | `/avions` | Lister tous les avions | 200 | |
| `POST` | `/avions` | Ajouter un avion | 201 | 400, 409, 413 |
| `GET` | `/avions/{immatriculation}` | Détails d'un avion | 200 | 404 |
| `DELETE` | `/avions/{immatriculation}` | Supprimer un avion sans vol associé | 204 | 404, 409 |
| `POST` | `/avions/{immatriculation}/vols` | Enregistrer un vol | 201 | 400, 404, 409, 413 |
| `GET` | `/avions/{immatriculation}/vols` | Lister les vols d'un avion | 200 | 404 |
| `GET` | `/stats` | Nombre d'avions et de vols par statut | 200 | |

Toutes les réponses (y compris les erreurs, les routes inconnues et les méthodes non autorisées) sont en JSON avec l'en-tête `Content-Type: application/json`, sauf le `204 No Content` qui n'a pas de corps.

### Données

**Avion**

| Champ | Type | Règles |
|---|---|---|
| `immatriculation` | string | identifiant unique, 2 à 10 caractères (lettres, chiffres, tirets), mise en majuscules |
| `modele` | string | obligatoire, 100 caractères max. |
| `nombre_places` | int | entre 1 et 1000 |
| `compagnie` | string | obligatoire, 100 caractères max. |
| `cree_le` | date-heure | renseignée par le serveur |

**Vol**

| Champ | Type | Règles |
|---|---|---|
| `numero_vol` | string | identifiant unique, 2 à 10 caractères (lettres, chiffres, tirets), mis en majuscules |
| `aeroport_depart` | string | obligatoire |
| `aeroport_arrivee` | string | obligatoire, différent de l'aéroport de départ |
| `date` | string | format `YYYY-MM-DD`, date valide |
| `immatriculation` | string | issue de l'URL ; l'avion doit exister |
| `statut` | string | `programme` (par défaut), `en_cours` ou `annule` |
| `cree_le` | date-heure | renseignée par le serveur |

Les champs inconnus sont refusés et le corps des requêtes est limité à 1 Mo.

### Erreurs

Les erreurs ont toujours la même forme ; `champ` n'est présent que pour les erreurs de validation :

```json
{"erreur": "nombre_places : doit être strictement positif", "champ": "nombre_places"}
```

| Code | Cas |
|---|---|
| 400 | Données invalides, JSON mal formé, champ inconnu |
| 404 | Avion inexistant, route inconnue |
| 405 | Méthode non autorisée sur une route existante |
| 409 | Immatriculation ou numéro de vol déjà utilisé, suppression d'un avion ayant des vols |
| 413 | Corps de requête supérieur à 1 Mo |
| 500 | Erreur interne (détail journalisé côté serveur, jamais exposé) |

---

## Exemples

```bash
# Ajouter un avion
curl -X POST http://localhost:8080/avions \
  -H 'Content-Type: application/json' \
  -d '{"immatriculation":"F-GKXA","modele":"A320","nombre_places":180,"compagnie":"Air France"}'

# Détails d'un avion
curl http://localhost:8080/avions/F-GKXA

# Enregistrer un vol (statut "programme" par défaut)
curl -X POST http://localhost:8080/avions/F-GKXA/vols \
  -H 'Content-Type: application/json' \
  -d '{"numero_vol":"AF1234","aeroport_depart":"CDG","aeroport_arrivee":"JFK","date":"2026-12-01"}'

# Lister les vols de l'avion
curl http://localhost:8080/avions/F-GKXA/vols

# Statistiques
curl http://localhost:8080/stats

# Supprimer un avion (refusé avec 409 tant qu'il a des vols)
curl -X DELETE http://localhost:8080/avions/F-GKXA
```

---

## Tests

```bash
go test ./...
```

Avec le détecteur de concurrence et la couverture :

```bash
go test -race -cover ./...
```

- Tests table-driven avec sous-tests `t.Run`, au moins un fichier `_test.go` par package
- Tests HTTP avec `httptest.NewRequest` et `httptest.NewRecorder` : codes de réponse, en-têtes et corps JSON
- La couche `store` est testée sur une vraie base SQLite temporaire ; la couche `service` sur un dépôt en mémoire

---

## Organisation du code

```
.
├── cmd/api/            point d'entrée : configuration, assemblage, démarrage et arrêt propre
├── internal/
│   ├── model/          structures de données (Avion, Vol, Statut, Stats) et tags JSON
│   ├── apperr/         erreurs métier : sentinelles et ValidationError
│   ├── store/          accès SQLite : schéma, requêtes, traduction des erreurs SQL
│   ├── service/        logique métier : validation, doublons, règle de suppression, statistiques
│   └── handler/        HTTP : routage, décodage/encodage JSON, codes de réponse, middlewares
├── Dockerfile          build multi-stage, image finale distroless non-root
└── docker-compose.yml
```

Chaque couche ne dépend que des couches inférieures : `handler` → `service` → `store`. Les packages `model` et `apperr` sont partagés.

Les erreurs sont créées au plus près de leur cause (par exemple `ErrNotFound` dans `store`), enveloppées avec `%w` en remontant, puis traduites en code HTTP dans `handler` avec `errors.Is` et `errors.As`.

### Choix techniques

- **Routage** : motifs de `http.ServeMux` (Go 1.22+) avec méthode et paramètre, lus par `r.PathValue`.
- **SQLite pur Go** : compilation avec `CGO_ENABLED=0`, binaire statique, image Docker d'environ 21 Mo.
- **Robustesse** : corps limité par `http.MaxBytesReader`, timeouts du serveur, récupération des paniques en 500 JSON, arrêt propre sur `SIGINT`/`SIGTERM`.
- **Intégrité** : clés primaires et clé étrangère en base, en plus des vérifications du service.
