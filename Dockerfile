# Build stage: compile a static Go binary.
# Pin the Go version to match go.mod (go 1.26 toolchain).
FROM golang:1.26 AS builder

WORKDIR /src

# Copy module files first so this layer is cached unless they change.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source and build a fully static, stripped binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /bin/task-api .

# Final stage: minimal distroless image (no shell, no package manager).
# -nonroot variant runs as uid 65532 by default → satisfies the non-root requirement.
FROM gcr.io/distroless/static:nonroot

# Copy the static binary from the builder.
COPY --from=builder --chown=nonroot:nonroot /bin/task-api /usr/local/bin/task-api

# Document the port (informational only; does not publish it).
EXPOSE 8080

# HEALTHCHECK uses the binary's built-in -healthcheck flag so the container
# can become "healthy" even though distroless has no shell, curl, or wget.
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/task-api", "-healthcheck"]

# Run the API server (distroless:nonroot already sets USER nonroot).
ENTRYPOINT ["/usr/local/bin/task-api"]
