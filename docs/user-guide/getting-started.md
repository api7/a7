# Getting Started

This guide helps you set up and use the a7 CLI to manage your API7 Enterprise Edition (API7 EE) instance.

## Prerequisites

Before installing a7, ensure you have:

- Go 1.22 or higher.
- An API7 Enterprise Edition instance with the Admin API enabled. The default Admin API port is 7443 (HTTPS).

## Installation

Install the a7 CLI directly with the Go command:

```bash
go install github.com/api7/a7/cmd/a7@latest
```

Alternatively, you can build from source:

```bash
git clone https://github.com/api7/a7.git
cd a7
make build
```

The resulting binary will be located in the current directory.

## Configuring Your First Context

The a7 CLI uses "contexts" to manage different API7 EE environments. A context stores the server address, the token, and the target gateway group. Use the `context create` command to set up your first connection.

```bash
a7 context create local \
  --server https://localhost:7443 \
  --token your-api7-token \
  --gateway-group default \
  --tls-skip-verify
```

Example output:

```bash
✓ Context "local" created and saved.
✓ Context "local" set as current context.
```

Your configuration is stored in `~/.config/a7/config.yaml` by default. You can override this location by setting the `A7_CONFIG_DIR` environment variable.

### Using Environment Variables

If you prefer not to use a context, you can set the following environment variables:

- `A7_SERVER`: The Admin API server address (e.g., `https://localhost:7443`).
- `A7_TOKEN`: Your API7 EE token.
- `A7_GATEWAY_GROUP`: The target gateway group name.

## Verifying the Connection

Check if a7 can communicate with your API7 EE instance by listing the available gateway groups:

```bash
a7 gateway-group list
```

If the connection is successful, you will see a list of gateway groups.

## Your First Route

Once you have a working context, you can start managing routes within a gateway group.

### 1. Create a Route Configuration

Create a file named `route.json` with the following content:

```json
{
  "id": "getting-started",
  "name": "getting-started-route",
  "uri": "/get",
  "methods": ["GET"],
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "httpbin.org:80": 1
    }
  }
}
```

### 2. Apply the Route

Use the `route create` command to send this configuration to API7 EE. Note that `--gateway-group` (or `-g`) is required for runtime resource commands if not set in the current context.

```bash
a7 route create -g default -f route.json
```

### 3. Verify the Route

List your routes to see the new entry:

```bash
a7 route list -g default
```

You can also get the full details of the route you just created:

```bash
a7 route get getting-started -g default
```

### 4. Test the Route

Assuming your API7 gateway is running and listening for data plane traffic (default port 9443), you can test the route with `curl`:

```bash
curl -ik https://localhost:9443/get
```

### 5. Clean Up

When you are done, you can delete the route:

```bash
a7 route delete getting-started -g default --force
```

## Interactive Mode

When you run a command that requires a resource ID without providing one, a7 presents an interactive fuzzy-filterable list:

```bash
# Instead of remembering the route ID...
a7 route get -g default

# a7 fetches all routes and presents a picker:
# > Select a route
#   my-api (1)
#   auth-service (2)
#   health-check (3)
```

This works for resource commands that support ID-based get, delete, and update. Interactive mode requires a terminal. In scripts or pipes, provide the ID explicitly.

## Managing Multiple Contexts

You can create multiple contexts for different environments or gateway groups.

```bash
a7 context create staging \
  --server https://staging.api:7443 \
  --token YOUR_STAGING_TOKEN \
  --gateway-group staging-group
```

To see all available contexts:

```bash
a7 context list
```

Example output:

```bash
NAME     SERVER                     GATEWAY GROUP    CURRENT
local    https://localhost:7443     default          *
staging  https://staging.api:7443   staging-group
```

To switch between contexts, use `context use`:

```bash
a7 context use staging
```

Example output:

```bash
✓ Switched to context "staging".
```

You can verify the active context anytime:

```bash
a7 context current
```

## Bulk Operations

You can delete or export multiple resources with one command.

```bash
# Delete all routes in the current gateway group
a7 route delete --all --force -g default

# Export upstreams by label as JSON
a7 upstream export --label team=platform --output json -g default
```

## What's Next

- Check the [Configuration Guide](configuration.md) for detailed configuration options.
- See the [Route Management Guide](route.md) for comprehensive route CRUD operations.

## Shell Completion

The a7 CLI supports shell completion for bash, zsh, fish, and PowerShell.

### Bash

```bash
# Load completions in current session
source <(a7 completion bash)

# To load completions for every session (Linux)
a7 completion bash > /etc/bash_completion.d/a7

# To load completions for every session (macOS with Homebrew)
a7 completion bash > $(brew --prefix)/etc/bash_completion.d/a7
```

### Zsh

```bash
# Enable shell completion if not already done
echo "autoload -U compinit; compinit" >> ~/.zshrc

# Generate and install completion
a7 completion zsh > "${fpath[1]}/_a7"
```

You will need to start a new shell for this to take effect.
