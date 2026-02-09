# --- STAGE 1: Builder ---
# Usamos una imagen reciente. Si Go 1.25 no está oficial en DockerHub, usamos 'latest' o '1.24-rc'.
FROM golang:alpine AS builder

# Instalar certificados CA y herramientas básicas
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Primero copiamos dependencias para aprovechar el caché de Docker
COPY go.mod ./
# COPY go.sum ./  <-- Descomenta si ya tienes go.sum
RUN go mod download

# Copiamos el código fuente
COPY . .

# Compilamos el binario estático (CGO_ENABLED=0 es vital para Alpine)
# -ldflags="-s -w" quita símbolos de debug para reducir tamaño
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o conciliador-web ./cmd/web

# --- STAGE 2: Runner ---
FROM alpine:latest

# Instalar certificados de seguridad (necesarios para llamar al SAT https)
RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copiamos el binario del stage anterior
COPY --from=builder /app/conciliador-web .

# Copiamos los templates y estáticos (VITAL: Docker no copia carpetas solas)
COPY --from=builder /app/web ./web
COPY --from=builder /app/internal/adapters/sat/templates ./internal/adapters/sat/templates

# Puerto que expone el contenedor (Railway usa la variable PORT, pero documentamos 8080)
EXPOSE 8080

# Comando de arranque
CMD ["./conciliador-web"]