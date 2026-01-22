# SPIFFE AuthZ Proxy

A proxy designed to run as a sidecar container that provides SPIFFE-based X509
SVID authentication and authorization for inbound requests to an upstream HTTP
server.

## Before

The application has to interact with the workloadapi and SVIDs to serve
terminate TLS with an SVID and authenticate and authorize inbound requests:

```mermaid
graph TD
    svc[kubernetes service] -->app
    app -->|get own SVID for tls|workloadapi
    workloadapi -->app
    app -->|get bundles|workloadapi
    workloadapi -->app
    app -->|investigate client certs|openssl["ssl library"]
```

The application also needs to manage certificate expiry and rotation, which can
introduce latency or concurrency challenges.

## After

The application doesn't need to know anything about SPIFFE, SVIDs, or the
workloadapi. The proxy terminates TLS with an appropriate SVID. Route (HTTP
method + path) based authorization is performed based on authenticated X509
SVIDs, and the SPIFFE ID is passed along as a plain HTTP header.

```mermaid
graph TD
    svc[kubernetes service] -->proxy
    proxy -->|get and manage updated svids and bundles|workloadapi
    workloadapi -->proxy
    proxy -->app
```

The application offloads managing SPIFFE authentication and route-based
authorization. Requests it processes are already authenticated and authorized.
It does not need to handle TLS termination.

## Configuration

Configuration is via environment variables. Most of them have reasonable
defaults. Only `AUTHZ_CONFIG` is required.

|env var|description|default|
|---|---|---|
| `AUTHZ_CONFIG` | The authoziration config source ([see below](#authz-config)). **Required**. | |
| `LOG_LEVEL` | Set the log level. Accepts Golang log/slog levels. | `INFO` |
| `LOG_FORMAT` | Set the log format. Accepts either `json` or `text`. | `json` |
| `BIND_ADDR` | The IP and port to bind and listen on. | `:8443` |
| `WORKLOAD_API` | The address (either `tcp://` with a network address and port, or `unix://` with a path to a socket) of the Workload API endpoint. | `unix:///tmp/spire-agent/public/agent.sock` |
| `UPSTREAM_ADDR` | The address (either `tcp://` with a network address and port, or `unix://` with a path to a socket) of the upstream server. | `tcp://127.0.0.1:8000` |

## AuthZ Config

### Sources

The `AUTHZ_CONFIG` variable typically looks like a URL, with structure that
depends on the scheme. The supported schemes are `file:` and `configmap:`.

#### `file:` sources

With a `file:` URL, the path component is the filesystem path to a file
containing the AuthZ configuration. For example:

```sh
AUTHZ_CONFIG=file:///path/to/file.conf
```

#### `configmap:` sources

Within Kubernetes, you can specify a ConfigMap that the workload can read
containing the AuthZ configuration. The authority is the name of the ConfigMap,
and the path is the filename within the Data section of the ConfigMap. For
example:

```sh
AUTHZ_CONFIG=configmap://some-configmap/authz-file.conf
```

Would look for a ConfigMap like

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: some-configmap
data:
  authz-file.conf: |
    spiffeid "spiffe://example.org/foo/bar" {}
```

The ConfigMap must be in the same Namespace as the workload.

If it can, `spiffe-authz-proxy` will attempt to watch the specified ConfigMap
for changes and reload the rules when necessary. In order for this to work, the
ServiceAccount for the workload needs ot have both `get` and `watch`
permissions on the ConfigMap.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: config-map-watcher
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "watch"]
  resourceNames: ["some-configmap"]
```

### Syntax

The authorization rules are defined with HCL. `spiffeid` blocks are the top
level config, granting access to SVIDs with the given SPIFFE ID.  Under each
`spiffeid` block, `path` blocks allow specific paths or patterns.  Within a
`path` block, the `methods` array allows specific HTTP methods.

```hcl
# rules for requests with SVIDs for the spiffeid spiffe://example.org/workloads/workload-a
spiffeid "spiffe://example.org/workloads/workload-a" {
    # allows requests to exactly `/foo`
    path "/foo" {
        # allows GET and POST requests
        methods = ["GET", "POST"]
    }

    # allows requests to /widgets/1 but not /widgets/1/details
    path "/widgets/*" {
        # allows all request methods
        methods = ["*"]
    }

    # allows requests to any path starting with the prefix /sprockets/
    path "/sprockets/**" {
        # allow only GET, HEAD, and OPTIONS requests
        methods = ["GET", "HEAD", "OPTIONS"]
    }
}
```
