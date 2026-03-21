---
name: a7-plugin-kafka-logger
description: >-
  Skill for configuring the API7 Enterprise Edition (API7 EE) kafka-logger plugin via the a7 CLI.
  Covers pushing access logs to Apache Kafka topics, broker configuration,
  SASL authentication, and gateway group scoping.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: plugin
  apisix_version: ">=3.0.0"
  plugin_name: kafka-logger
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 config sync
---

# a7-plugin-kafka-logger

## Overview

The `kafka-logger` plugin pushes request/response logs to Apache Kafka
topics. It supports multiple brokers, SASL authentication, async/sync
producing, custom log formats, and batch processing for efficient delivery.

## When to Use

- Stream access logs to Kafka for downstream processing
- Feed real-time API analytics pipelines
- Integrate with Kafka-based logging infrastructure
- Need SASL-authenticated Kafka clusters

## Plugin Configuration Reference

### Core Parameters

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `brokers` | array | **Yes** | — | Kafka broker list |
| `brokers[].host` | string | **Yes** | — | Broker hostname or IP |
| `brokers[].port` | integer | **Yes** | — | Broker port (1-65535) |
| `kafka_topic` | string | **Yes** | — | Target Kafka topic |
| `key` | string | No | — | Partition key for routing |
| `timeout` | integer | No | `3` | Connection timeout in seconds |

### SASL Authentication

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `brokers[].sasl_config` | object | No | — | SASL config per broker |
| `brokers[].sasl_config.mechanism` | string | No | `"PLAIN"` | `PLAIN`, `SCRAM-SHA-256`, or `SCRAM-SHA-512` |
| `brokers[].sasl_config.user` | string | Yes* | — | SASL username (*if sasl_config set) |
| `brokers[].sasl_config.password` | string | Yes* | — | SASL password (*if sasl_config set) |

### Producer Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `producer_type` | string | `"async"` | `async` (batched) or `sync` (immediate) |
| `required_acks` | integer | `1` | `1` (leader ack) or `-1` (all replicas) |
| `producer_batch_num` | integer | `200` | Messages per Kafka batch |
| `producer_batch_size` | integer | `1048576` | Batch size in bytes (1MB) |

### Log Format Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `meta_format` | string | `"default"` | `default` (JSON) or `origin` (raw HTTP) |
| `log_format` | object | — | Custom log format with `$variable` syntax |
| `include_req_body` | boolean | `false` | Include request body |
| `include_req_body_expr` | array | — | Conditional request body logging |

## Step-by-Step: Ship Logs to Kafka

### 1. Create a route with kafka-logger

Enable logging for gateway group `default`:

```bash
a7 route create --gateway-group default -f - <<'EOF'
{
  "id": "kafka-logged-api",
  "uri": "/api/*",
  "plugins": {
    "kafka-logger": {
      "brokers": [
        {"host": "kafka-1", "port": 9092},
        {"host": "kafka-2", "port": 9092}
      ],
      "kafka_topic": "api7-logs",
      "batch_max_size": 100
    }
  },
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "backend:8080": 1
    }
  }
}
EOF
```

### 2. Global logging for a gateway group

Apply a Global Rule for all traffic in the `prod` group:

```bash
a7 global_rule create --gateway-group prod -f - <<'EOF'
{
  "id": "kafka-logger-global",
  "plugins": {
    "kafka-logger": {
      "brokers": [{"host": "kafka-broker", "port": 9092}],
      "kafka_topic": "prod-logs",
      "batch_max_size": 500
    }
  }
}
EOF
```

## Common Patterns

### SASL-authenticated Kafka cluster

```json
{
  "plugins": {
    "kafka-logger": {
      "brokers": [
        {
          "host": "kafka.example.com",
          "port": 9092,
          "sasl_config": {
            "mechanism": "SCRAM-SHA-256",
            "user": "api7-user",
            "password": "secret-password"
          }
        }
      ],
      "kafka_topic": "secure-logs",
      "required_acks": -1
    }
  }
}
```

### Custom log format

```json
{
  "plugins": {
    "kafka-logger": {
      "brokers": [{"host": "kafka", "port": 9092}],
      "kafka_topic": "api-logs",
      "log_format": {
        "@timestamp": "$time_iso8601",
        "client_ip": "$remote_addr",
        "method": "$request_method",
        "uri": "$request_uri",
        "status": "$status",
        "latency": "$request_time"
      }
    }
  }
}
```

### Partition by route ID

```json
{
  "plugins": {
    "kafka-logger": {
      "brokers": [{"host": "kafka", "port": 9092}],
      "kafka_topic": "api-logs",
      "key": "$route_id"
    }
  }
}
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| No messages in Kafka | Broker unreachable | Verify broker host/port; check firewall from gateway nodes |
| SASL auth failure | Wrong credentials or mechanism | Verify user/password; ensure mechanism matches Kafka config |
| Messages delayed | Large batch/timeout settings | Reduce `inactive_timeout` and `producer_time_linger` |
| Messages dropped | Buffer overflow | Increase `producer_max_buffering`; add more brokers |
| Topic not found | Topic doesn't exist | Create topic manually in Kafka or enable auto-creation |
| Config not applied | Wrong gateway group specified | Ensure `--gateway-group` matches the desired cluster |

## Config Sync Example

```yaml
version: "1"
gateway_group: default
routes:
  - id: kafka-logged-api
    uri: /api/*
    plugins:
      kafka-logger:
        brokers:
          - host: kafka-1
            port: 9092
          - host: kafka-2
            port: 9092
        kafka_topic: api7-logs
        producer_type: async
        required_acks: 1
        batch_max_size: 200
        inactive_timeout: 5
    upstream_id: my-upstream
```
