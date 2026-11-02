# syntax=docker/dockerfile:1

# --- Étape 1 : compilation ---
FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && mkdir -p /out/data

# --- Étape 2 : image d'exécution minimale ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/api /api
COPY --from=build --chown=65532:65532 /out/data /data

ENV DB_PATH=/data/compagnie.db
EXPOSE 8080
VOLUME ["/data"]
USER 65532:65532

ENTRYPOINT ["/api"]
