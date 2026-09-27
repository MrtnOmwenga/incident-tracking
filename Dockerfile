# Build a static binary, then ship it alone on a distroless base: no shell, no package manager,
# runs as an unprivileged user.
FROM docker.io/library/golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /lighthouse ./cmd/lighthouse

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /lighthouse /lighthouse
USER nonroot:nonroot
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=5s --retries=3 CMD ["/lighthouse", "healthcheck"]
ENTRYPOINT ["/lighthouse"]
CMD ["serve"]
