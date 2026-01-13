FROM golang:1.25-alpine AS builder

WORKDIR /build/src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags=-s -o spiffe-authz-proxy main.go

FROM scratch

LABEL org.opencontainers.image.description="A reverse proxy sidecare to provide SPIFFE-based AuthN and AuthZ"
LABEL org.opencontainers.image.licenses=Apache-2.0
EXPOSE 8443

COPY --from=builder /build/src/spiffe-authz-proxy /usr/bin/spiffe-authz-proxy

ENTRYPOINT [ "/usr/bin/spiffe-authz-proxy" ]
