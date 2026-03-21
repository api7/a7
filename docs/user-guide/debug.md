# Debug and Tracing (Planned)

> **Note**: Debug and tracing commands are planned for **Phase 7** of the `a7` CLI development and are **Not Yet Implemented**. This document describes the planned interface.

The `a7 debug` command group will help diagnose request behavior and monitor logs for API7 Enterprise Edition (EE) instances.

## Planned `a7 debug trace`

The `trace` command will analyze how a request is handled by a specific route within a gateway group. It will combine Control Plane configuration with real-time Data Plane feedback.

```bash
# Planned
a7 debug trace <route-id> --gateway-group <group-name>
```

### Planned Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group (required) |
| `--method` | | `GET` | HTTP method for the probe request |
| `--path` | | `/` | Request path for the probe request |
| `--header` | | | Request header in `Key: Value` format (repeatable) |
| `--body` | | | Request body for the probe request |
| `--output` | `-o` | `table` | Output format (`table`, `json`, `yaml`) |

### Planned Example

```bash
# Planned
a7 debug trace my-route -g prod \
  --method POST \
  --path /api/v1/data \
  --header "Authorization: Bearer token123" \
  --body '{"key": "value"}'
```

The output will provide a detailed breakdown of:
- Route matching logic
- Plugin execution order and results
- Upstream selection and latency
- Response status and headers

## Planned `a7 debug logs`

The `logs` command will stream or fetch logs from API7 EE gateway instances.

```bash
# Planned
a7 debug logs --gateway-group <group-name>
```

### Planned Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--gateway-group` | `-g` | | Target gateway group (required) |
| `--follow` | `-f` | `false` | Stream logs continuously |
| `--tail` | `-n` | `100` | Number of recent lines to show |
| `--since` | | | Show logs since duration (e.g., `5m`, `1h`) |
| `--type` | `-t` | `all` | Log type: `error`, `access`, `all` |
| `--instance` | | | Filter logs from a specific gateway instance ID |

### Planned Example

Stream access logs from the `prod` gateway group:

```bash
# Planned
a7 debug logs -g prod --follow --type access
```

Fetch the last 500 error logs from a specific instance:

```bash
# Planned
a7 debug logs -g prod --type error --tail 500 --instance gw-1
```
