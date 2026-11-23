FROM node:22-alpine AS webbuild
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=webbuild /internal/web/static/next ./internal/web/static/next
# Bundle the web shell (hashed script + stylesheet); the server falls back to
# the plain sources when this output is absent.
RUN CGO_ENABLED=0 go run ./cmd/assetbuild
ARG VERSION=0.0.0-dev
ARG COMMIT=
RUN GOMAXPROCS=1 CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w -X mochil/internal/version.Version=${VERSION} -X mochil/internal/version.Commit=${COMMIT}" -o /out/mochil ./cmd/mochil \
    && mkdir -p /out/data \
    && chown 65532:65532 /out/data

FROM alpine:3.22
RUN apk add --no-cache ca-certificates ffmpeg mesa-dri-gallium mesa-va-gallium su-exec
COPY --from=build /out/mochil /mochil
COPY --from=build --chown=65532:65532 /out/data /data
COPY --chmod=0755 deployment/container-entrypoint.sh /container-entrypoint.sh
# The managed-config template. The container starts in config.json mode from the first run, so the
# settings page can save; -library/-cache remain available for a single-folder install.
COPY deployment/bootstrap.json /bootstrap/config.json
EXPOSE 8080
VOLUME ["/books", "/config", "/data"]
ENTRYPOINT ["/container-entrypoint.sh"]
CMD ["-config", "/config/config.json", "-config-template", "/bootstrap/config.json"]
