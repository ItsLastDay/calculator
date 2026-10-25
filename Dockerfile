# syntax=docker/dockerfile:1

# Builds the React frontend and the Go backend, then ships both in one small,
# non-root image: the Go server serves the API under /api/ and the built
# frontend from STATIC_DIR, so the browser talks to a single origin.
#
#   docker build -t calculator .
#   docker run --rm -p 8080:8080 calculator
#
# Test stages are not part of the default build; run them explicitly:
#
#   docker build --target backend-test .
#   docker build --target frontend-test .

ARG NODE_VERSION=22
ARG GO_VERSION=1.23

# ---- Frontend ---------------------------------------------------------------

FROM node:${NODE_VERSION}-alpine AS frontend-deps
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./

FROM frontend-deps AS frontend-test
RUN npm run lint && npm run format:check && npm run typecheck && npm run coverage

FROM frontend-deps AS frontend-build
RUN npm run build

# ---- Backend ----------------------------------------------------------------

FROM golang:${GO_VERSION}-alpine AS backend-base
WORKDIR /src
COPY backend/go.mod ./
RUN go mod download
COPY backend/ ./

# The race detector needs cgo and glibc, so tests run on the Debian image.
FROM golang:${GO_VERSION} AS backend-test
WORKDIR /src
COPY backend/ ./
RUN go vet ./... && go test -race -cover ./...

FROM backend-base AS backend-build
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/calculator ./cmd/server

# ---- Runtime ----------------------------------------------------------------

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend-build /out/calculator /app/calculator
COPY --from=frontend-build /app/dist /app/static
ENV ADDR=:8080 \
    STATIC_DIR=/app/static
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/calculator", "healthcheck"]
ENTRYPOINT ["/app/calculator"]
