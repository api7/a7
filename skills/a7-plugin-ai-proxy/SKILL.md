---
name: a7-plugin-ai-proxy
description: >-
  Skill for configuring the API7 Enterprise Edition ai-proxy plugin via the a7 CLI.
  Covers proxying requests to LLM providers (OpenAI, Azure OpenAI, DeepSeek,
  Anthropic, Gemini, Vertex AI, and more), authentication per provider,
  model configuration, streaming, logging, and service template usage.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: plugin
  apisix_version: ">=3.9.0"
  plugin_name: ai-proxy
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 service-template create
    - a7 config sync
---

# a7-plugin-ai-proxy

## Overview

The `ai-proxy` plugin turns API7 Enterprise Edition (API7 EE) into an AI gateway. It proxies requests in
OpenAI-compatible format to LLM providers, handling authentication, endpoint
routing, and response streaming. Clients send a standard chat-completion
request; the plugin translates and forwards it to the configured provider.

## When to Use

- Proxy chat-completion or embedding requests to any supported LLM provider
- Centralize API keys at the gateway instead of distributing to clients
- Add observability (token counts, latency) to LLM calls
- Combine with `ai-prompt-template`, `ai-prompt-decorator`, or content
  moderation plugins for a full AI gateway pipeline
- Apply consistent AI proxy configurations across services using **Service Templates**

## Supported Providers

| Provider | Value | Default Endpoint |
|----------|-------|------------------|
| OpenAI | `openai` | `https://api.openai.com/v1/chat/completions` |
| DeepSeek | `deepseek` | `https://api.deepseek.com/chat/completions` |
| Azure OpenAI | `azure-openai` | Custom via `override.endpoint` |
| Anthropic | `anthropic` | `https://api.anthropic.com/v1/chat/completions` |
| AIMLAPI | `aimlapi` | `https://api.aimlapi.com/v1/chat/completions` |
| OpenRouter | `openrouter` | `https://openrouter.ai/api/v1/chat/completions` |
| Gemini | `gemini` | `https://generativelanguage.googleapis.com/v1beta/openai/chat/completions` |
| Vertex AI | `vertex-ai` | `https://aiplatform.googleapis.com` |
| OpenAI-Compatible | `openai-compatible` | Custom via `override.endpoint` |

## Plugin Configuration Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `provider` | string | **Yes** | — | One of the 9 supported providers |
| `auth` | object | **Yes** | — | Authentication config (see below) |
| `options` | object | No | — | Model and generation parameters |
| `options.model` | string | No | — | Model name (provider-specific) |
| `options.temperature` | number | No | — | Sampling temperature |
| `options.top_p` | number | No | — | Nucleus sampling |
| `options.max_tokens` | integer | No | — | Maximum tokens to generate |
| `options.stream` | boolean | No | `false` | Enable SSE streaming |
| `override` | object | No | — | Override default endpoint |
| `override.endpoint` | string | No | — | Full URL for the provider API |
| `provider_conf` | object | No | — | Provider-specific config (Vertex AI) |
| `provider_conf.project_id` | string | No | — | GCP project ID (Vertex AI) |
| `provider_conf.region` | string | No | — | GCP region (Vertex AI) |
| `logging` | object | No | — | Logging options |
| `logging.summaries` | boolean | No | `false` | Log model, duration, tokens |
| `logging.payloads` | boolean | No | `false` | Log request/response bodies |
| `timeout` | integer | No | `30000` | Request timeout (ms) |
| `keepalive` | boolean | No | `true` | Keep connection alive |
| `keepalive_timeout` | integer | No | `60000` | Keepalive timeout (ms) |
| `keepalive_pool` | integer | No | `30` | Keepalive pool size |
| `ssl_verify` | boolean | No | `true` | Verify SSL certificate |

## Authentication by Provider

### OpenAI / DeepSeek / Anthropic / AIMLAPI / OpenRouter

```json
{
  "auth": {
    "header": {
      "Authorization": "Bearer sk-your-api-key"
    }
  }
}
```

### Azure OpenAI

```json
{
  "auth": {
    "header": {
      "api-key": "your-azure-key"
    }
  },
  "override": {
    "endpoint": "https://YOUR-RESOURCE.openai.azure.com/openai/deployments/gpt-4/chat/completions?api-version=2024-02-15-preview"
  }
}
```

### Vertex AI (GCP Service Account)

```json
{
  "auth": {
    "gcp": {
      "service_account_json": "{ ... }",
      "max_ttl": 3600,
      "expire_early_secs": 60
    }
  },
  "provider_conf": {
    "project_id": "your-project-id",
    "region": "us-central1"
  }
}
```

## Step-by-Step: Route to OpenAI

### 1. Create a route with ai-proxy

All runtime resources like routes must be scoped to a gateway group using `--gateway-group` or `-g`.

```bash
a7 route create -g default -f - <<'EOF'
{
  "id": "openai-chat",
  "uri": "/v1/chat/completions",
  "methods": ["POST"],
  "plugins": {
    "ai-proxy": {
      "provider": "openai",
      "auth": {
        "header": {
          "Authorization": "Bearer sk-your-openai-key"
        }
      },
      "options": {
        "model": "gpt-4",
        "temperature": 0.7,
        "max_tokens": 1024
      }
    }
  }
}
EOF
```

### 2. Send a request

```bash
curl http://127.0.0.1:9080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "What is 1+1?"}
    ]
  }'
```

## Using Service Templates

In API7 EE, you can enable `ai-proxy` on a **Service Template** to standardize AI gateway configurations across multiple services.

```bash
a7 service-template create -f - <<'EOF'
{
  "id": "standard-ai-proxy",
  "name": "Standard AI Proxy",
  "plugins": {
    "ai-proxy": {
      "provider": "openai",
      "auth": {
        "header": {
          "Authorization": "Bearer sk-global-key"
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

## Common Patterns

### Streaming responses

```json
{
  "plugins": {
    "ai-proxy": {
      "provider": "openai",
      "auth": {
        "header": {
          "Authorization": "Bearer sk-your-key"
        }
      },
      "options": {
        "model": "gpt-4",
        "stream": true
      }
    }
  }
}
```

### Model Routing with Multiple Routes

The plugin does not natively route by model. Use separate routes with `vars` matching on request body fields:

```bash
# Route requests for gpt-4 to OpenAI
a7 route create -g default -f - <<'EOF'
{
  "id": "openai-gpt4",
  "uri": "/v1/chat/completions",
  "methods": ["POST"],
  "vars": [["post_arg.model", "==", "gpt-4"]],
  "plugins": {
    "ai-proxy": {
      "provider": "openai",
      "auth": { "header": { "Authorization": "Bearer sk-openai-key" } },
      "options": { "model": "gpt-4" }
    }
  }
}
EOF
```

## Access Log Variables

| Variable | Description |
|----------|-------------|
| `$request_type` | `traditional_http`, `ai_chat`, or `ai_stream` |
| `$llm_time_to_first_token` | Time to first token (ms) |
| `$llm_model` | Actual model used by provider |
| `$request_llm_model` | Model requested by client |
| `$llm_prompt_tokens` | Prompt token count |
| `$llm_completion_tokens` | Completion token count |

## Config Sync Example

Config sync is scoped by gateway group:

```bash
a7 config sync -f config.yaml --gateway-group default
```

```yaml
version: "1"
routes:
  - id: openai-chat
    uri: /v1/chat/completions
    methods:
      - POST
    plugins:
      ai-proxy:
        provider: openai
        auth:
          header:
            Authorization: Bearer sk-your-openai-key
        options:
          model: gpt-4
          max_tokens: 1024
          temperature: 0.7
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| 502 Bad Gateway | Wrong endpoint or provider value | Verify `provider` matches; check `override.endpoint` |
| 401 from upstream | Invalid API key | Check `auth.header` value |
| 404 Not Found | Missing `--gateway-group` | Ensure all runtime commands include `-g <group>` |
| Azure 404 | Missing api-version in URL | Include `?api-version=YYYY-MM-DD-preview` in `override.endpoint` |
