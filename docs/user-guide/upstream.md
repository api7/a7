# Upstream Management

> **⚠️ Standalone upstreams are NOT exposed via the API7 Enterprise Edition Admin API.** In API7 EE, upstreams exist only as inline objects within services and routes. The `/apisix/admin/upstreams` endpoint returns "resource not found".

## How Upstreams Work in API7 EE

Instead of creating standalone upstream resources, define upstreams inline when creating a service or route:

```yaml
# service-with-upstream.yaml
name: my-service
upstream:
  type: roundrobin
  nodes:
    - host: 127.0.0.1
      port: 8080
      weight: 1
```

```bash
a7 service create -g default -f service-with-upstream.yaml
```

Or inline in a route:

```yaml
# route-with-upstream.yaml
name: my-route
paths:
  - /api/v1/*
service_id: "<service-id>"
upstream:
  type: roundrobin
  nodes:
    - host: 127.0.0.1
      port: 8080
      weight: 1
```

```bash
a7 route create -g default -f route-with-upstream.yaml
```

## CLI Commands (Not Functional in API7 EE)

The `a7 upstream` commands exist in the CLI for APISIX compatibility but will return errors when used against an API7 EE instance. Use `a7 service` or `a7 route` with inline upstream configurations instead.

## See Also

- [Service Management](service.md) — Create services with inline upstreams
- [Route Management](route.md) — Create routes with inline upstreams
