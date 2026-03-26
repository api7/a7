# Bulk Operations (Planned)

> **Note**: Bulk operations are planned for **Phase 9** of the `a7` CLI development and are **Not Yet Implemented**. This document describes the planned interface.

Bulk operations will allow you to perform actions on multiple API7 Enterprise Edition (EE) resources with a single command, scoped to a gateway group.

## Planned Interface

Every bulk operation will require the `--gateway-group` (or `-g`) flag.

### Bulk Delete

The `delete` command for each resource type will support bulk operations via the `--all` or `--label` flags.

#### Delete All Resources in a Group

```bash
# Planned
a7 route delete --all -g default --force
```

#### Delete by Label

```bash
# Planned
a7 service delete --label env=staging -g prod --force
```

### Bulk Export

Each resource will have an `export` command to dump multiple resources to a file or stdout.

#### Export All Resources of a Type

```bash
# Planned
a7 upstream export -g default
```

#### Export by Label

```bash
# Planned
a7 route export --label team=payments -g prod -o json -f payments-routes.json
```

## Planned Flag Reference

### Bulk Delete Flags

| Flag | Description |
|------|-------------|
| `--all` | Delete all resources of the specified type within the gateway group |
| `--label` | Delete only resources matching the specified label (`key=value`) |
| `--gateway-group` | Target gateway group (required) |
| `--force` | Skip confirmation prompt |

### Bulk Export Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--label` | | | Export only resources matching the label |
| `--gateway-group` | `-g` | | Target gateway group (required) |
| `--output` | `-o` | `yaml` | Export format (`yaml`, `json`) |
| `--file` | `-f` | | Write to file instead of stdout |

## Planned Resource Support

Bulk operations are planned for the following resources:

- `route`
- `service`
- `consumer`
- `ssl`
- `global-rule`
- `stream-route`
- `proto`
- `secret`
- `plugin-metadata`
