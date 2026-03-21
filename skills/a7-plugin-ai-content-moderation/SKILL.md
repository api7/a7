---
name: a7-plugin-ai-content-moderation
description: >-
  Skill for configuring API7 Enterprise Edition AI content moderation plugins via the a7 CLI.
  Covers both ai-aws-content-moderation (AWS Comprehend, request-only) and
  ai-aliyun-content-moderation (Aliyun, request + response with streaming),
  toxicity thresholds, category filtering, and integration with ai-proxy.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: plugin
  apisix_version: ">=3.9.0"
  plugin_name: ai-aws-content-moderation
  related_plugins:
    - ai-aliyun-content-moderation
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 service-template create
    - a7 config sync
---

# a7-plugin-ai-content-moderation

## Overview

API7 Enterprise Edition (API7 EE) provides two content moderation plugins that filter harmful content
in LLM requests and responses:

| Plugin | Provider | Request | Response | Streaming |
|--------|----------|---------|----------|-----------|
| `ai-aws-content-moderation` | AWS Comprehend | ✅ | ❌ | ❌ |
| `ai-aliyun-content-moderation` | Aliyun Moderation Plus | ✅ | ✅ | ✅ |

Both must be used alongside `ai-proxy` or `ai-proxy-multi`.

## When to Use

- Block toxic, hateful, or sexual content before it reaches the LLM
- Filter harmful LLM responses before they reach clients (Aliyun only)
- Enforce content policies with configurable thresholds
- Apply consistent moderation policies using **Service Templates**

## Plugin Execution Order

```
ai-prompt-template           (priority 1071)
ai-prompt-decorator          (priority 1070)
ai-aws-content-moderation    (priority 1050) ← runs BEFORE ai-proxy
ai-proxy                     (priority 1040)
ai-aliyun-content-moderation (priority 1029) ← runs AFTER ai-proxy
```

---

## Plugin 1: ai-aws-content-moderation

Uses the AWS Comprehend `detectToxicContent` API to score request content.

### Configuration Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `comprehend.access_key_id` | string | **Yes** | — | AWS access key ID |
| `comprehend.secret_access_key` | string | **Yes** | — | AWS secret access key |
| `comprehend.region` | string | **Yes** | — | AWS region (e.g. `us-east-1`) |
| `moderation_categories` | object | No | — | Per-category thresholds (0-1) |
| `moderation_threshold` | number | No | `0.5` | Overall toxicity threshold (0-1) |

### Step-by-Step: AWS Content Moderation

All runtime resources must be scoped to a gateway group using `--gateway-group` or `-g`.

```bash
a7 route create -g default -f - <<'EOF'
{
  "id": "moderated-chat",
  "uri": "/v1/chat/completions",
  "methods": ["POST"],
  "plugins": {
    "ai-aws-content-moderation": {
      "comprehend": {
        "access_key_id": "AKIAIOSFODNN7EXAMPLE",
        "secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
        "region": "us-east-1"
      },
      "moderation_categories": {
        "HATE_SPEECH": 0.3,
        "VIOLENCE_OR_THREAT": 0.2
      }
    },
    "ai-proxy": {
      "provider": "openai",
      "auth": {
        "header": {
          "Authorization": "Bearer sk-your-key"
        }
      },
      "options": {
        "model": "gpt-4"
      }
    }
  }
}
EOF
```

---

## Plugin 2: ai-aliyun-content-moderation

Uses Aliyun Machine-Assisted Moderation Plus. Supports request moderation,
response moderation, and real-time streaming moderation.

### Configuration Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `endpoint` | string | **Yes** | — | Aliyun service endpoint URL |
| `region_id` | string | **Yes** | — | Aliyun region (e.g. `cn-shanghai`) |
| `access_key_id` | string | **Yes** | — | Aliyun access key ID |
| `access_key_secret` | string | **Yes** | — | Aliyun access key secret |
| `check_request` | boolean | No | `true` | Enable request moderation |
| `check_response` | boolean | No | `false` | Enable response moderation |
| `risk_level_bar` | string | No | `high` | Threshold: `none`, `low`, `medium`, `high`, `max` |

### Step-by-Step: Aliyun Request + Response Moderation

```bash
a7 route create -g default -f - <<'EOF'
{
  "id": "aliyun-moderated-chat",
  "uri": "/v1/chat/completions",
  "methods": ["POST"],
  "plugins": {
    "ai-proxy": {
      "provider": "openai",
      "auth": {
        "header": {
          "Authorization": "Bearer sk-your-key"
        }
      },
      "options": {
        "model": "gpt-4"
      }
    },
    "ai-aliyun-content-moderation": {
      "endpoint": "https://green.cn-shanghai.aliyuncs.com",
      "region_id": "cn-shanghai",
      "access_key_id": "your-aliyun-key-id",
      "access_key_secret": "your-aliyun-key-secret",
      "check_request": true,
      "check_response": true,
      "risk_level_bar": "high"
    }
  }
}
EOF
```

## Using Service Templates

You can define standard moderation policies on a **Service Template** to reuse them across multiple services.

```bash
a7 service-template create -f - <<'EOF'
{
  "id": "standard-moderation",
  "name": "Standard Moderation",
  "plugins": {
    "ai-aws-content-moderation": {
      "comprehend": {
        "access_key_id": "...",
        "secret_access_key": "...",
        "region": "us-east-1"
      },
      "moderation_threshold": 0.5
    }
  }
}
EOF
```

## Config Sync Example

Config sync is scoped by gateway group:

```bash
a7 config sync -f config.yaml --gateway-group default
```

```yaml
version: "1"
routes:
  - id: moderated-chat
    uri: /v1/chat/completions
    methods:
      - POST
    plugins:
      ai-aws-content-moderation:
        comprehend:
          access_key_id: AKIA...
          secret_access_key: wJal...
          region: us-east-1
        moderation_threshold: 0.5
      ai-proxy:
        provider: openai
        auth:
          header:
            Authorization: Bearer sk-your-key
        options:
          model: gpt-4
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| "no ai instance picked" | Aliyun plugin used without ai-proxy | Always configure ai-proxy on the same route |
| 404 Not Found | Missing `--gateway-group` | Ensure runtime commands include `-g <group>` |
| AWS not blocking | Threshold too permissive | Lower `moderation_threshold` |
| Aliyun response inactive | `check_response` defaults to `false` | Set `check_response: true` |
| Signature mismatch | Wrong Aliyun credentials | Verify credentials |
