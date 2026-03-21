# Documentation Maintenance Specification

### Purpose
This document defines the mandatory rules for maintaining a7 documentation. Every code change that affects user-facing behavior MUST include corresponding documentation updates. This is enforced in code review.

### Documentation Structure
```
docs/
├── api7ee-api-spec.md              # API7 EE Admin API reference
├── adr/                           # Architecture Decision Records
│   └── 001-tech-stack.md         # Tech stack decisions
├── golden-example.md              # Reference implementation template
├── coding-standards.md            # Code style guide
├── testing-strategy.md            # Test patterns and requirements
├── documentation-maintenance.md   # THIS file
├── roadmap.md                     # Development roadmap
├── skills.md                      # AI agent skills reference
└── user-guide/                    # End-user documentation
    ├── getting-started.md         # Installation + first commands
    ├── configuration.md           # Context management, config file format
    ├── gateway-group.md           # Gateway group commands (EE)
    ├── service-template.md        # Service template commands (EE)
    ├── route.md                   # Route command reference
    ├── upstream.md                # Upstream command reference
    ├── service.md                 # Service command reference
    ├── consumer.md                # Consumer command reference
    ├── ssl.md                     # SSL command reference
    ├── plugin.md                  # Plugin command reference
    ├── global-rule.md             # Global rule command reference
    ├── stream-route.md            # Stream route command reference
    ├── plugin-config.md           # Plugin config command reference
    ├── plugin-metadata.md         # Plugin metadata command reference
    ├── consumer-group.md          # Consumer group command reference
    ├── credential.md              # Credential command reference
    ├── secret.md                  # Secret command reference
    ├── proto.md                   # Proto command reference
    ├── rbac.md                    # Users, Roles, Policies (EE)
    ├── portal.md                  # Developer portal (EE)
    ├── audit-log.md               # Audit logs (EE)
    ├── custom-plugin.md           # Custom plugins (EE)
    ├── service-registry.md        # Service registries (EE)
    ├── token.md                   # Access tokens (EE)
    └── ...                        # Total 29 files in user-guide
```

### Mandatory Documentation Rules

#### Rule 1: New Command → New/Updated User Guide
When adding a new command (e.g., `a7 gateway-group list`):
- Create or update `docs/user-guide/<resource>.md`
- Include command syntax, all flags with descriptions, examples, and common use cases
- Format: every command section must have Synopsis, Description, Flags table (including `--gateway-group`), and Examples

#### Rule 2: New Flag → Update Command Reference
When adding a new flag to an existing command:
- Update the flags table in the corresponding user guide
- Add an example showing the flag in use

#### Rule 3: Behavior Change → Update Affected Docs
When changing command behavior (output format, error messages, defaults):
- Update all affected user guide pages
- If the change affects the golden example pattern, update `docs/golden-example.md`

#### Rule 4: API Client Change → Update API Spec
When the API7 EE Admin API changes (new field, endpoint, parameter):
- Update `docs/api7ee-api-spec.md`
- Ensure dual-API prefix and gateway group scoping are correctly documented

#### Rule 5: Architecture Change → New ADR
When making a significant architectural decision:
- Create `docs/adr/NNN-<title>.md` following the ADR format
- Link from AGENTS.md

### User Guide Page Template
Resource command reference pages follow this template:

```markdown
# <Resource> Commands

Manage API7 EE <resources>.

## a7 <resource> list

List all <resources>.

### Synopsis
\`\`\`
a7 <resource> list [flags]
\`\`\`

### Flags
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| --gateway-group | | string | (context) | Gateway group ID for scoping |
| --output | -o | string | (auto) | Output format: table, json, yaml |
| --page | | int | 1 | Page number |
| --page-size | | int | 10 | Items per page (10-500) |
| --name | | string | | Filter by name |

### Examples
\`\`\`bash
# List all resources in the default group
a7 <resource> list

# List as JSON for a specific group
a7 <resource> list --gateway-group prod-group -o json

# Filter by name
a7 <resource> list --name "my-resource"
\`\`\`
```

### Documentation Quality Checklist
Before approving any PR, verify:
- [ ] All new commands have user guide entries
- [ ] All flags are documented with types and defaults
- [ ] Gateway group scoping is explained for runtime resources
- [ ] At least 2 examples per command
- [ ] Examples are realistic and tested
- [ ] No broken internal links
- [ ] AGENTS.md document map is up to date

### Who Updates Documentation
- AI coding agents: MUST update docs as part of every feature PR
- Human reviewers: MUST check docs checklist before approving
- If docs are missing from a code PR, the PR is NOT ready for merge