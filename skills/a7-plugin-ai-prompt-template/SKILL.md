---
name: a7-plugin-ai-prompt-template
description: >-
  Skill for configuring the API7 Enterprise Edition ai-prompt-template plugin via the
  a7 CLI. Covers defining reusable prompt templates with variable placeholders,
  enforcing prompt structure, and combining with ai-proxy for a complete AI gateway pipeline.
version: "1.0.0"
author: API7.ai Contributors
license: Apache-2.0
metadata:
  category: plugin
  apisix_version: ">=3.9.0"
  plugin_name: ai-prompt-template
  a7_commands:
    - a7 route create
    - a7 route update
    - a7 service-template create
    - a7 config sync
---

# a7-plugin-ai-prompt-template

## Overview

The `ai-prompt-template` plugin pre-configures prompt templates with
`{{variable}}` placeholders. Clients submit only the template name and
variable values; the plugin fills the template and produces a complete
chat-completion request. This enforces prompt structure and prevents
clients from sending arbitrary system prompts.

**Priority**: 1071 (runs before `ai-prompt-decorator` at 1070 and
`ai-proxy` at 1040).

## When to Use

- Enforce a fixed prompt structure across all clients
- Accept user inputs only for specific fields (fill-in-the-blank)
- Prevent prompt injection by controlling the system message
- Build prompt libraries that clients select by name
- Standardize prompt templates across services using **Service Templates**

## Plugin Configuration Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `templates` | array | **Yes** | Array of template objects (min 1) |
| `templates[].name` | string | **Yes** | Template identifier (min length 1) |
| `templates[].template` | object | **Yes** | Template specification |
| `templates[].template.model` | string | **Yes** | AI model name |
| `templates[].template.messages` | array | **Yes** | Array of message objects (min 1) |
| `templates[].template.messages[].role` | string | **Yes** | `system`, `user`, or `assistant` |
| `templates[].template.messages[].content` | string | **Yes** | Prompt content with `{{variable}}` placeholders |

## Client Request Format

Instead of sending a standard `messages` array, clients send:

```json
{
  "template_name": "my-template",
  "variable1": "value1",
  "variable2": "value2"
}
```

## Step-by-Step: Create a Templated Route

### 1. Create a route with ai-prompt-template and ai-proxy

All runtime resources must be scoped to a gateway group using `--gateway-group` or `-g`.

```bash
a7 route create -g default -f - <<'EOF'
{
  "id": "templated-chat",
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
    "ai-prompt-template": {
      "templates": [
        {
          "name": "code-review",
          "template": {
            "model": "gpt-4",
            "messages": [
              {
                "role": "system",
                "content": "You are an expert {{language}} code reviewer. Review the code for bugs, performance issues, and style."
              },
              {
                "role": "user",
                "content": "Review this code:\n\n{{code}}"
              }
            ]
          }
        }
      ]
    }
  }
}
EOF
```

### 2. Send a request with template variables

```bash
curl http://127.0.0.1:9080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "code-review",
    "language": "Python",
    "code": "def add(a, b): return a + b"
  }'
```

## Using Service Templates

You can define standard prompt templates on a **Service Template** to reuse them across multiple services.

```bash
a7 service-template create -f - <<'EOF'
{
  "id": "global-ai-prompts",
  "name": "Global AI Prompts",
  "plugins": {
    "ai-prompt-template": {
      "templates": [
        {
          "name": "summarize",
          "template": {
            "model": "gpt-4",
            "messages": [
              {"role": "system", "content": "Summarize this in one sentence."},
              {"role": "user", "content": "{{text}}"}
            ]
          }
        }
      ]
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
  - id: templated-chat
    uri: /v1/chat/completions
    methods:
      - POST
    plugins:
      ai-proxy:
        provider: openai
        auth:
          header:
            Authorization: Bearer sk-your-key
        options:
          model: gpt-4
      ai-prompt-template:
        templates:
          - name: code-review
            template:
              model: gpt-4
              messages:
                - role: system
                  content: "You are an expert {{language}} code reviewer."
                - role: user
                  content: "Review this code:\n\n{{code}}"
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| 400 "template not found" | `template_name` doesn't match configured template | Check spelling and case |
| Unfilled `{{variable}}` | Variable key missing from request body | Include all variables in the request JSON |
| 404 Not Found | Missing `--gateway-group` | Ensure all runtime commands include `-g <group>` |
| Plugin not transforming | Wrong plugin name | Verify plugin name is `ai-prompt-template` |
