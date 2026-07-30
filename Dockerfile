# syntax=docker/dockerfile:1

# --- builder -----------------------------------------------------------
FROM golang:1.26-bookworm AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/batocera-scrap-manager .

# --- runtime -------------------------------------------------------------
# distroless: the binary above is a static, CGO-free Go build that needs
# neither a shell nor a package manager at runtime — the nonroot variant
# also gives it a non-root user and CA certificates for free (see
# .vibe/decisions/034).
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/batocera-scrap-manager /batocera-scrap-manager

EXPOSE 8080

# Exec form so the binary runs as PID 1 and receives SIGTERM directly for
# serve's graceful shutdown; distroless has no shell to wrap it in anyway.
ENTRYPOINT ["/batocera-scrap-manager", "serve", "--addr", "0.0.0.0:8080"]
