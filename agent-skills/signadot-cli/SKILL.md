---
name: signadot-cli
description: Manage Signadot sandboxes, route groups, clusters, resource plugins, jobs, and smart tests using the signadot CLI. Use when a developer or platform engineer needs to create, update, inspect, or delete Signadot resources.
argument-hint: "[resource: sandbox|routegroup|cluster|job|resourceplugin|smart-test]"
---

# Signadot CLI

Help the user manage Signadot resources using the `signadot` CLI. If the user specifies a resource type ($ARGUMENTS), focus on that section. Otherwise, provide guidance based on what they're trying to accomplish.

## Core Concepts

- **Sandbox**: An isolated environment comprising forked workloads, local workloads, or virtual workloads plus ephemeral resources. Each sandbox gets a unique routing key for request routing.
- **Route Group**: Dynamically groups sandboxes by label criteria and provides shared preview URLs. Useful for combining multiple sandboxes into a single test environment.
- **Preview URL**: An authenticated URL that routes traffic to a specific sandbox or route group by automatically injecting the routing key.
- **Resource Plugin**: A reusable template for provisioning ephemeral resources (databases, queues, etc.) within sandboxes.
- **Templates**: YAML files with `@{variable}` directives, expanded at apply time with `--set variable=value`.

## Configuration

The CLI reads `$HOME/.signadot/config.yaml` by default. Use `--config` to switch:

```bash
signadot --config ~/configs/staging.yaml sandbox list
```

See the `signadot-install` skill for full config file reference.

## Authentication

```bash
# Interactive browser login
signadot auth login

# API key login
signadot auth login --with-api-key <key>

# Check status
signadot auth status

# Logout
signadot auth logout
```

For CI, use environment variables: `SIGNADOT_ORG` and `SIGNADOT_API_KEY`.

## Sandbox Management (alias: sb)

### Create or Update a Sandbox

Write a YAML spec file and apply it:

```yaml
# my-sandbox.yaml
name: my-sandbox
spec:
  cluster: my-cluster
  description: Testing new feature
  forks:
  - forkOf:
      kind: Deployment
      namespace: example
      name: my-app
    customizations:
      images:
      - image: example.com/my-app:dev-abcdef
        container: my-app
      env:
      - name: FEATURE_FLAG
        container: my-app
        operation: upsert
        value: "true"
  defaultRouteGroup:
    endpoints:
    - name: my-endpoint
      target: http://my-app.example.svc:8080
```

```bash
signadot sandbox apply -f my-sandbox.yaml
```

### With Templates

```yaml
# sandbox-template.yaml
name: "@{dev}-feature-x"
spec:
  cluster: "@{cluster}"
  description: "Feature X sandbox for @{dev}"
  forks:
  - forkOf:
      kind: Deployment
      namespace: "@{namespace}"
      name: my-app
    customizations:
      images:
      - image: "example.com/my-app:@{tag}"
        container: my-app
```

```bash
signadot sandbox apply -f sandbox-template.yaml \
  --set dev=jane \
  --set cluster=staging \
  --set namespace=default \
  --set tag=pr-42-abc123
```

### Template Features

- **Variables**: `@{variable}` — replaced via `--set variable=value`
- **Embeddings**: `@{embed: file.txt}` — inline file contents
- **Encodings**: `@{var[yaml]}` for YAML expansion, `@{var[binary]}` for binary, `@{var[raw]}` (default) for string interpolation

### List, Get, Delete

```bash
# List all sandboxes
signadot sandbox list
signadot sb list -o json

# Get details
signadot sandbox get my-sandbox
signadot sb get my-sandbox -o yaml

# Delete by name
signadot sandbox delete my-sandbox

# Delete by spec file
signadot sandbox delete -f my-sandbox.yaml
```

### Sandbox with Local Mappings

Requires `signadot local connect` first (see the signadot-local skill).

```yaml
name: my-local-sandbox
spec:
  cluster: staging
  local:
  - name: local-my-app
    from:
      kind: Deployment
      namespace: hotrod
      name: my-app
    mappings:
    - port: 8080
      toLocal: localhost:3000
```

### Working with Local Sandbox Configuration

After creating a sandbox with local mappings, extract the environment and files the workload needs:

```bash
# Get environment variables (can be eval'd or sourced)
signadot sandbox get-env my-local-sandbox

# Get mounted files (ConfigMaps, Secrets)
signadot sandbox get-files my-local-sandbox
```

### Sandbox TTL (Auto-deletion)

```yaml
spec:
  ttl:
    duration: "2h"        # auto-delete after 2 hours
    offsetFrom: createdAt  # or updatedAt
```

## Route Group Management (alias: rg)

Route groups combine multiple sandboxes via label matching and provide shared endpoints.

```yaml
# my-routegroup.yaml
name: my-routegroup
spec:
  cluster: my-cluster
  description: "Route group for feature X"
  match:
    any:
    - label:
        key: feature
        value: "feature-x-*"
  endpoints:
  - name: frontend
    target: http://frontend.hotrod.svc:8080
```

```bash
# Apply
signadot routegroup apply -f my-routegroup.yaml

# List
signadot routegroup list
signadot rg list -o json

# Get details
signadot routegroup get my-routegroup

# Delete
signadot routegroup delete my-routegroup
```

### Match Criteria

- **Single label**: `match.label: {key: "k", value: "v"}`
- **Any (OR)**: `match.any: [{label: {key: "k1", value: "v1"}}, ...]`
- **All (AND)**: `match.all: [{label: {key: "k1", value: "v1"}}, ...]`
- Values support glob patterns: `value: "feature-*"`

## Cluster Management (alias: cl)

```bash
# Add a cluster
signadot cluster add --name my-cluster

# List clusters
signadot cluster list

# Manage cluster tokens
signadot cluster token list --cluster my-cluster
signadot cluster token create --cluster my-cluster
signadot cluster token delete --cluster my-cluster <token-id>

# Analyze DevMesh workloads
signadot cluster devmesh analyze --cluster my-cluster
```

## Devbox Management

Devboxes represent developer machines connected via `signadot local connect`.

```bash
# Register a devbox
signadot devbox register --name my-devbox

# List devboxes (own)
signadot devbox list

# List all devboxes in org
signadot devbox list --all

# Delete a devbox
signadot devbox delete <devbox-id>
```

## Resource Plugin Management (alias: rp)

Resource plugins provision ephemeral resources (databases, queues, etc.) for sandboxes.

```bash
# Apply a resource plugin spec
signadot resourceplugin apply -f my-plugin.yaml

# List plugins
signadot resourceplugin list

# Get details
signadot resourceplugin get my-plugin

# Delete
signadot resourceplugin delete my-plugin
```

## Job Management

Jobs run tests or tasks in the context of a sandbox.

```bash
# Submit a job
signadot job submit -f my-job.yaml --set sandbox=my-sandbox

# List jobs
signadot job list
signadot job list --all

# Get job details
signadot job get my-job

# Cancel/delete a job
signadot job delete my-job
```

## Smart Tests (alias: st)

```bash
# List smart tests
signadot smart-test list

# Run a smart test
signadot smart-test run <test-name>

# List executions
signadot smart-test execution list

# Get execution details
signadot smart-test execution get <execution-id>

# Cancel execution
signadot smart-test execution cancel <execution-id>
```

## Job Runner Groups (alias: jrg)

```bash
signadot jobrunnergroup list
signadot jobrunnergroup get <name>
signadot jobrunnergroup apply -f <file>
signadot jobrunnergroup delete <name>
```

## MCP Server

The CLI can run as an MCP (Model Context Protocol) server for AI-assisted workflows:

```bash
signadot mcp
```

## Output Formats

Most commands support `-o json` or `-o yaml` for machine-readable output:

```bash
signadot sandbox list -o json
signadot routegroup get my-rg -o yaml
```

## Common Patterns

### CI/CD: Create sandbox per PR

```bash
signadot sandbox apply -f sandbox.yaml \
  --set name="pr-${PR_NUMBER}" \
  --set tag="${COMMIT_SHA}" \
  --set cluster=staging
```

### CI/CD: Clean up on PR close

```bash
signadot sandbox delete "pr-${PR_NUMBER}"
```

### Test against a sandbox

```bash
# Get the preview URL
signadot sandbox get my-sandbox -o json | jq -r '.endpoints[0].url'

# Or use local proxy for direct access
signadot local proxy --sandbox my-sandbox \
  --map http://backend.staging.svc:8000@localhost:8001
```

## Troubleshooting

- **"sandbox not found"**: Check `signadot sandbox list` and verify the name
- **Sandbox stuck in non-Ready state**: Check `signadot sandbox get <name> -o yaml` for status conditions
- **Template errors**: Ensure all `@{variables}` have corresponding `--set` flags
- **Permission errors**: Verify `signadot auth status` shows authenticated
