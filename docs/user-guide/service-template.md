# Service Template

The `a7 service-template` command manages API7 Enterprise Edition (EE) service templates. Service templates are design-time service definitions stored on the Control Plane. They can be "published" to one or more gateway groups to create runtime services.

## Design-time vs Runtime

- **Service Template**: A Control Plane resource (`/api/services/template`) where you define the service's structure, plugins, and configuration.
- **Service**: A runtime resource (`/apisix/admin/services`) that exists within a specific gateway group and actually handles traffic.

## Commands

### `a7 service-template list`

Lists all service templates on the Control Plane.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `table` | Output format (`table`, `json`, `yaml`) |

**Examples:**

```bash
a7 service-template list
```

### `a7 service-template get`

Gets detailed information about a specific service template.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `yaml` | Output format (`json`, `yaml`) |

**Examples:**

```bash
a7 service-template get st-123
```

### `a7 service-template create`

Creates a new service template from a file or flags.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | `-n` | | Template name (required) |
| `--file` | `-f` | | Path to configuration file |

**Examples:**

```bash
a7 service-template create --name "User API" -f user-service.yaml
```

### `a7 service-template update`

Updates an existing service template.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | | Path to configuration file |

**Examples:**

```bash
a7 service-template update st-123 -f updated-user-service.json
```

### `a7 service-template delete`

Deletes a service template.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | | `false` | Skip confirmation prompt |

**Examples:**

```bash
a7 service-template delete st-123 --force
```

### `a7 service-template publish`

Publishes a service template to one or more gateway groups. This operation creates or updates corresponding runtime services in the targeted groups.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group-id` | | | ID of the target gateway group (can be repeated) |

**Examples:**

Publish to a single gateway group:
```bash
a7 service-template publish st-123 --gateway-group-id default
```

Publish to multiple gateway groups:
```bash
a7 service-template publish st-123 --gateway-group-id dev --gateway-group-id staging
```

## Example Configuration

```yaml
id: st-123
name: "Order API"
desc: "Template for the ordering microservice"
upstream:
  type: roundrobin
  nodes:
    "orders.internal:8080": 1
plugins:
  proxy-rewrite:
    uri: "/v1/orders"
```
    
## Relationship with Routes

When you create a route, you can reference a published service (which may have originated from a service template):

```yaml
# In a route config
service_id: "order-service-runtime-id"
```
