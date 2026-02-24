---
name: signadot-local
description: Connect your local machine to a Kubernetes cluster for local development with Signadot — local connect, proxy, traffic override, and traffic recording/inspection. Use when a developer wants to run services locally that participate in cluster traffic, or record/inspect sandbox traffic.
argument-hint: "[command: connect|proxy|override|traffic]"
---

# Signadot Local Development

Help the user connect their local machine to a Kubernetes cluster, run workloads locally that participate in cluster traffic, override sandbox traffic, and record/inspect traffic. If the user specifies a command ($ARGUMENTS), focus on that section.

**Important**: This skill covers two separate CLI subcommands:
- `signadot local` — connect, proxy, override, status, disconnect
- `signadot traffic` — record and inspect sandbox traffic (does NOT require `signadot local connect`)

For traffic recording/inspection, go directly to `signadot traffic record` or `signadot traffic inspect`. Do not use `signadot local` for traffic operations.

## Overview

Signadot local development enables developers to:
1. Run a service on their laptop that receives real cluster traffic
2. Access cluster services (DNS, networking) from local code
3. Intercept and override specific sandbox traffic with a local implementation
4. Record and inspect HTTP/gRPC traffic flowing through sandboxes

## Prerequisites

- Signadot CLI installed and authenticated (`signadot auth status`)
- Signadot Operator installed in the target cluster
- Local configuration in `~/.signadot/config.yaml` (for `local connect`). Use `--config` to switch between config files:

```yaml
local:
  connections:
  - cluster: <cluster-name>       # from Signadot dashboard
    kubeContext: <kube-context>    # from your kubeconfig
```

```bash
# Use a specific config for a different cluster/org
signadot --config ~/configs/staging.yaml local connect --cluster staging
```

### Connection Types

Configure in `~/.signadot/config.yaml` under each connection:

- **PortForward** (default): Uses kubectl port-forwarding. Requires `kubeContext`.
- **ControlPlaneProxy**: Routes through Signadot control plane. No kubectl access needed. Add `type: ControlPlaneProxy`.
- **ProxyAddress**: For exposed SOCKS5 proxy (e.g. VPN). Add `type: ProxyAddress` and `proxyAddress: <host>:<port>`.

```yaml
# ControlPlaneProxy — simplest setup, no kubectl required
local:
  connections:
  - cluster: staging
    type: ControlPlaneProxy
```

## signadot local connect

Establishes bidirectional network connectivity between your machine and the cluster. Requires root privileges.

```bash
sudo signadot local connect --cluster <cluster-name>
```

This will:
- Update `/etc/hosts` with cluster service DNS names
- Configure networking so your machine can reach cluster services
- Enable cluster traffic to reach local workloads via sandboxes

Once connected, your local code can resolve and call cluster services by their Kubernetes DNS names (e.g. `my-service.my-namespace.svc`).

### Unprivileged Mode

```bash
signadot local connect --cluster <cluster-name> --unprivileged
```

Limited functionality: no `/etc/hosts` updates or system networking changes.

## signadot local status

Shows current connection status, active sandboxes, and health:

```bash
signadot local status
signadot local status -o json
signadot local status -o yaml
```

## signadot local disconnect

Tears down the cluster connection:

```bash
# Keep sandbox configurations
signadot local disconnect

# Also remove all local sandbox configurations
signadot local disconnect --clean-local-sandboxes
```

## Working with Local Sandboxes

After connecting, create a sandbox with local mappings to route cluster traffic to your local service:

```yaml
# local-sandbox.yaml
name: my-local-dev
spec:
  cluster: staging
  description: "Local dev for my-service"
  local:
  - name: local-my-service
    from:
      kind: Deployment
      namespace: my-namespace
      name: my-service
    mappings:
    - port: 8080
      toLocal: localhost:3000
```

```bash
signadot sandbox apply -f local-sandbox.yaml
```

### Extract Environment and Files

Get the environment variables and config files your local service needs to match the cluster workload:

```bash
# Print environment variables (can be eval'd)
signadot sandbox get-env my-local-dev
# Example output:
# export MYSQL_HOST="localhost"
# export MYSQL_PORT="3306"
# export DEBUG="true"

# Extract config files
signadot sandbox get-files my-local-dev
# Files are saved under ~/.signadot/sandboxes/<name>/local/files/
```

### Typical Local Development Workflow

```bash
# 1. Connect to cluster
sudo signadot local connect --cluster staging

# 2. Create sandbox with local mapping
signadot sandbox apply -f local-sandbox.yaml

# 3. Get environment for your local service
eval $(signadot sandbox get-env my-local-dev)

# 4. Get config files
signadot sandbox get-files my-local-dev

# 5. Run your service locally
go run ./cmd/server --port 3000

# 6. Test via preview URL or cluster traffic
# Your local service now receives traffic routed to the sandbox

# 7. Clean up
signadot sandbox delete my-local-dev
signadot local disconnect
```

## signadot local proxy

Proxy cluster services to local ports. Similar to `kubectl port-forward` but with Signadot routing key injection.

```bash
signadot local proxy --sandbox <name> --map <remote>@<local>
signadot local proxy --routegroup <name> --map <remote>@<local>
signadot local proxy --cluster <name> --map <remote>@<local>
```

### Map Format

```
--map <scheme>://<host>:<port>@<host>:<port>
```

- **Left of `@`**: URL resolved in the cluster
- **Right of `@`**: Local bind address
- **Schemes**: `http`, `grpc`, `tcp` (tcp has no header/routing key injection)

### Examples

```bash
# Proxy a sandbox service to localhost:8001
signadot local proxy --sandbox feature-x \
  --map http://backend.staging.svc:8000@localhost:8001

# Use in a test script
export BACKEND=localhost:8001
signadot local proxy --sandbox feature-x \
  --map http://backend.staging.svc:8000@$BACKEND &
pid=$!

# Run tests against localhost:8001
npm test

kill $pid

# Proxy a route group
signadot local proxy --routegroup my-rg \
  --map http://frontend.hotrod.svc:8080@localhost:9090

# Proxy without routing key injection (raw TCP)
signadot local proxy --cluster staging \
  --map tcp://postgres.db.svc:5432@localhost:5432
```

## signadot local override

Intercept HTTP/gRPC traffic destined for a sandbox workload and route it to a local service. Requires CLI v1.3.0+ and Operator v1.2.0+.

```bash
signadot local override \
  --sandbox <sandbox> \
  --workload <workload-name> \
  --workload-port <port> \
  --with <local-address>
```

### How It Works

1. All HTTP/gRPC requests to the sandbox workload are first routed to your local service
2. If your local service responds with the header `sd-override: true` (HTTP) or metadata key `sd-override: true` (gRPC), that response is sent back to the client — the request never reaches the sandbox workload
3. If `sd-override: true` is absent, the request falls through to the original sandbox workload
4. If your local service is unavailable, all requests fall through automatically

### --except-status (inverse behavior)

Override all traffic EXCEPT when your local service returns specific status codes:

```bash
signadot local override \
  --sandbox my-sandbox \
  --workload my-workload \
  --workload-port 8080 \
  --with localhost:9999 \
  --except-status 404,503
```

With `--except-status`, your local service handles everything by default. Only when it returns one of the listed status codes does the request fall through to the sandbox workload.

### Detached Mode

Keep the override active after the CLI exits:

```bash
signadot local override \
  --sandbox my-sandbox \
  --workload my-workload \
  --workload-port 8080 \
  --with localhost:9999 \
  --detach
```

### Managing Overrides

```bash
# List active overrides
signadot local override list

# Delete a specific override
signadot local override delete <name> --sandbox <sandbox>
```

### Override with Virtual Workloads

Virtual workloads are zero-cost placeholders that point to baseline. Combined with `local override`, you can record baseline traffic at near-zero cost:

```yaml
spec:
  virtual:
  - name: my-virtual
    workload:
      kind: Deployment
      namespace: my-ns
      name: my-service
```

## signadot traffic record

Record HTTP/gRPC request/response traffic flowing through a sandbox. Requires CLI v1.3.0+ and Operator v1.2.0+.

```bash
signadot traffic record --sandbox <sandbox-name>
```

### Options

- `--inspect` — launch interactive TUI instead of log output
- `--clean` — erase previously recorded traffic before recording
- `--out-dir <dir>` — custom output directory
- `--short --to-file <file>` — record only the activity log (no request/response bodies)

### How It Works

- Temporarily adds `trafficwatch` middleware to the sandbox
- Records traffic in real time as it flows through the cluster
- On termination, removes the middleware changes it made
- Data is appended by default (use `--clean` to start fresh)

### Recorded Data Format

Each request/response pair produces:
- An entry in the activity log (JSON stream)
- A directory named by `middlewareRequestID` containing:
  - `meta.json` — request metadata
  - `request` — HTTP wire format (protocol line, headers, body)
  - `response` — HTTP wire format

### Example

```bash
# Record traffic with live TUI
signadot traffic record --sandbox my-sandbox --inspect

# Record to a clean directory
signadot traffic record --sandbox my-sandbox --clean --out-dir ./traffic-data

# Record activity log only
signadot traffic record --sandbox my-sandbox --short --to-file ./activity.json
```

## signadot traffic inspect

Browse previously recorded traffic in an interactive TUI:

```bash
signadot traffic inspect
```

Works with traffic recorded by `signadot traffic record`.

## macOS Considerations

### VPN Configuration

If your cluster is behind a VPN using a non-`en0` interface, add to `~/.signadot/config.yaml`:

```yaml
local:
  connections:
  - cluster: my-cluster
    outbound:
      macOSVPNInterface: utun6  # find with ifconfig
```

## Troubleshooting

- **"connect: permission denied"**: `signadot local connect` needs root. Use `sudo`.
- **Services not resolving**: Check `signadot local status` for connection health. Verify `/etc/hosts` was updated.
- **Traffic not reaching local service**: Ensure port mapping matches. Check `signadot sandbox get <name>` for sandbox status.
- **Override not intercepting**: Verify your local service is running on the `--with` address. Check `signadot local override list`.
- **Traffic record exits with error**: Often caused by CI overwriting the sandbox. The middleware config changed underneath.
