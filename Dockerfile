FROM golang:1.25-alpine AS builder

WORKDIR /build/src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags=-s -o spiffe-authz-proxy main.go

FROM scratch

EXPOSE 8443

COPY --from=builder /build/src/spiffe-authz-proxy /usr/bin/spiffe-authz-proxy

ENTRYPOINT [ "/usr/bin/spiffe-authz-proxy" ]
