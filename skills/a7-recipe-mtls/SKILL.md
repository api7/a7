---
name: a7-recipe-mtls
description: >-
  Recipe skill for configuring mutual TLS (mTLS) using the a7 CLI in API7 Enterprise Edition.
  Covers SSL certificate management, upstream mTLS to backend services,
  client certificate verification, and end-to-end mTLS setup from
  client through API7 EE to upstream.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: recipe
  apisix_version: ">=3.0.0"
  a7_commands:
    - a7 ssl create
    - a7 ssl update
    - a7 ssl list
    - a7 ssl get
    - a7 ssl delete
    - a7 upstream create
    - a7 upstream update
    - a7 route create
---

# a7-recipe-mtls

## Overview

Mutual TLS (mTLS) ensures both the client and server verify each other's
identity via TLS certificates. Standard TLS only verifies the server; mTLS
adds client certificate verification.

With API7 Enterprise Edition (API7 EE) and the a7 CLI, you can configure:
1. **Client → API7 EE mTLS**: Require clients to present valid certificates.
2. **API7 EE → Upstream mTLS**: Present client certificates when connecting to backends.
3. **End-to-end mTLS**: Both directions simultaneously.

## When to Use

- Zero-trust networking between services.
- Secure service-to-service communication in microservices.
- Compliance requirements mandating mutual authentication.
- Replace or supplement API key authentication with certificate-based auth.
- Internal APIs that should only be accessible by authorized services.

## Prerequisites

- API7 EE Control Plane and at least one Gateway Group.
- a7 CLI configured with a valid token and server address.
- Certificates and private keys in PEM format.

## Part 1: Client → API7 EE mTLS

Require clients to present a valid TLS certificate when connecting to API7 EE.

### 1. Create SSL resource with CA for client verification

```bash
a7 ssl create --gateway-group default -f - <<'EOF'
{
  "id": "mtls-domain",
  "cert": "<SERVER_CERTIFICATE_PEM>",
  "key": "<SERVER_PRIVATE_KEY_PEM>",
  "snis": ["api.example.com"],
  "client": {
    "ca": "<CA_CERTIFICATE_PEM>"
  }
}
EOF
```

**Fields**:
- `cert` / `key`: Server certificate and private key (presented to clients).
- `snis`: Server Name Indications — domain names this certificate covers.
- `client.ca`: CA certificate used to verify client certificates.
- `client.depth`: (optional) Maximum certificate chain depth for verification.

### 2. Create a route on the protected domain

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "secure-api",
  "uri": "/api/*",
  "host": "api.example.com",
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "backend:8080": 1
    }
  }
}
EOF
```

### 3. Test with client certificate

```bash
# With valid client cert — succeeds
curl --cert client.crt --key client.key --cacert ca.crt \
  https://api.example.com:9443/api/health

# Without client cert — fails with SSL handshake error
curl --cacert ca.crt https://api.example.com:9443/api/health
```

## Part 2: API7 EE → Upstream mTLS

Configure API7 EE to present a client certificate when connecting to backends.

### 1. Create upstream with TLS client certificate

```bash
a7 upstream create --gateway-group default -f - <<'EOF'
{
  "id": "mtls-backend",
  "type": "roundrobin",
  "scheme": "https",
  "nodes": {
    "secure-backend:443": 1
  },
  "tls": {
    "client_cert": "<CLIENT_CERTIFICATE_PEM>",
    "client_key": "<CLIENT_PRIVATE_KEY_PEM>"
  }
}
EOF
```

**Fields**:
- `scheme`: Must be `"https"` for TLS connections to upstream.
- `tls.client_cert`: Client certificate API7 EE presents to the upstream.
- `tls.client_key`: Private key for the client certificate.
- `pass_host`: Set to `"pass"` (default) or `"rewrite"` if upstream expects a specific Host header.

### 2. Create route using this upstream

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "api",
  "uri": "/api/*",
  "upstream_id": "mtls-backend"
}
EOF
```

## Part 3: End-to-End mTLS

Combine both: clients verify themselves to API7 EE, and API7 EE verifies
itself to the upstream.

### 1. SSL for client → API7 EE mTLS

```bash
a7 ssl create --gateway-group default -f - <<'EOF'
{
  "id": "frontend-mtls",
  "cert": "<API7_SERVER_CERT>",
  "key": "<API7_SERVER_KEY>",
  "snis": ["api.example.com"],
  "client": {
    "ca": "<CLIENT_CA_CERT>"
  }
}
EOF
```

### 2. Upstream for API7 EE → backend mTLS

```bash
a7 upstream create --gateway-group default -f - <<'EOF'
{
  "id": "secure-backend",
  "type": "roundrobin",
  "scheme": "https",
  "nodes": {
    "internal-service:443": 1
  },
  "tls": {
    "client_cert": "<API7_CLIENT_CERT>",
    "client_key": "<API7_CLIENT_KEY>"
  }
}
EOF
```

### 3. Route connecting both

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "e2e-mtls-api",
  "uri": "/api/*",
  "host": "api.example.com",
  "upstream_id": "secure-backend"
}
EOF
```

## Common Patterns

### Multiple domains with different CAs

```bash
# Domain A: internal services
a7 ssl create --gateway-group default -f - <<'EOF'
{
  "id": "internal-mtls",
  "cert": "<INTERNAL_CERT>",
  "key": "<INTERNAL_KEY>",
  "snis": ["internal.example.com"],
  "client": {
    "ca": "<INTERNAL_CA>"
  }
}
EOF

# Domain B: partner services
a7 ssl create --gateway-group default -f - <<'EOF'
{
  "id": "partner-mtls",
  "cert": "<PARTNER_CERT>",
  "key": "<PARTNER_KEY>",
  "snis": ["partner.example.com"],
  "client": {
    "ca": "<PARTNER_CA>"
  }
}
EOF
```

### Using Secrets for certificate management

API7 EE supports certificate storage in external secret managers. Configure secrets via Dashboard or a7 CLI first.

```bash
# Create a secret reference
a7 secret create --gateway-group default -f - <<'EOF'
{
  "id": "vault/mtls-certs",
  "uri": "https://vault.example.com/v1/secret/data/mtls"
}
EOF
```

## Config Sync Example

```yaml
version: "1"
gateway_group: default
ssls:
  - id: api-mtls
    cert: |
      -----BEGIN CERTIFICATE-----
      <server certificate>
      -----END CERTIFICATE-----
    key: |
      -----BEGIN RSA PRIVATE KEY-----
      <server private key>
      -----END RSA PRIVATE KEY-----
    snis:
      - api.example.com
    client:
      ca: |
        -----BEGIN CERTIFICATE-----
        <CA certificate for client verification>
        -----END CERTIFICATE-----
upstreams:
  - id: secure-backend
    type: roundrobin
    scheme: https
    nodes:
      "backend:443": 1
    tls:
      client_cert: |
        -----BEGIN CERTIFICATE-----
        <client certificate for upstream>
        -----END CERTIFICATE-----
      client_key: |
        -----BEGIN RSA PRIVATE KEY-----
        <client private key for upstream>
        -----END RSA PRIVATE KEY-----
routes:
  - id: mtls-api
    uri: /api/*
    host: api.example.com
    upstream_id: secure-backend
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| SSL handshake failure (client side) | Client cert not signed by the CA in `client.ca` | Verify CA chain; check that client cert is signed by the correct CA |
| "no required SSL certificate" | Client didn't send a certificate | Configure client to present cert (`--cert` in curl) |
| 502 to upstream | Upstream rejects API7 EE's client cert | Verify `tls.client_cert` is signed by the upstream's trusted CA |
| Certificate expired | TLS cert past validity date | Rotate certificate with `a7 ssl update` |
| SNI mismatch | Domain doesn't match `snis` list | Add the domain to the `snis` array |
| Command failed with 401 | Invalid token | Refresh your token using `a7 context create` |
| Upstream not found | Different gateway group | Ensure `--gateway-group` matches where resources were created |
