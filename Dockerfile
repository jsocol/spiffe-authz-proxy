FROM golang:1.26-alpine AS builder

WORKDIR /build/src

COPY go.mod go.sum ./

RUN \
  --mount=type=cache,target=/go/pkg/mod \
  go mod download

COPY . .

RUN \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 go build -v -trimpath -ldflags="-s -w" -o spiffe-authz-proxy cmd/proxy/main.go

FROM scratch

LABEL org.opencontainers.image.title="SPIFFE AuthZ Proxy"
LABEL org.opencontainers.image.description="A reverse proxy sidecare to provide SPIFFE-based AuthN and AuthZ"
LABEL org.opencontainers.image.source="https://github.com/jsocol/spiffe-authz-proxy"
LABEL org.opencontainers.image.url="https://github.com/jsocol/spiffe-authz-proxy"
LABEL org.opencontainers.image.licenses=Apache-2.0

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

USER nobody

COPY --from=builder /build/src/spiffe-authz-proxy /spiffe-authz-proxy

ENV LOG_FORMAT=json
ENV LOG_LEVEL=info
EXPOSE 8443

CMD [ "/spiffe-authz-proxy" ]
