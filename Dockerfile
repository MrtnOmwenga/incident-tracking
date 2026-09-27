# Build the console, then a static binary with the console embedded, then ship the binary alone on
# a distroless base: no shell, no package manager, runs as an unprivileged user.
# The console and the Go build run natively on the build machine; Go cross-compiles for the
# target, so an arm64 image needs no emulation.
FROM --platform=$BUILDPLATFORM docker.io/library/node:26-slim AS console
WORKDIR /src/console
COPY console/package.json console/package-lock.json ./
RUN npm ci
COPY console/ ./
RUN mkdir -p ../internal/web/console && npm run build

FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27 AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=console /src/internal/web/console/ ./internal/web/console/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /lighthouse ./cmd/lighthouse

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /lighthouse /lighthouse
# The portfolio's content ships in the image; mount another folder over /site to replace it.
COPY deploy/site /site
ENV SITE_DIR=/site
USER nonroot:nonroot
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=5s --retries=3 CMD ["/lighthouse", "healthcheck"]
ENTRYPOINT ["/lighthouse"]
CMD ["serve"]
