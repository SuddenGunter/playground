# syntax=docker/dockerfile:1

# Build stage runs natively on the build host and cross-compiles for the target,
# so multi-arch builds don't need QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/tcapp .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/tcapp /tcapp

ENV PORT=8080
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/tcapp"]
