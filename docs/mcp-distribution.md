# MCP Server Distribution

The CLI ships an MCP server (`signadot mcp`). This is the runbook for getting
it listed — and keeping it current — across the various MCP directories.
Statuses below reflect the sweep done in September 2026.

## Official MCP Registry

The entry on [registry.modelcontextprotocol.io](https://registry.modelcontextprotocol.io)
is named `io.github.signadot/cli` and is published per release with
`./scripts/gen-mcp-server-json.sh` and `mcp-publisher` — see
[Post-release: publish to the MCP Registry](../README.md#post-release-publish-to-the-mcp-registry)
in the README for the steps and PAT requirements. Several other directories
key off this entry, so publish here first.

## Bulk submission: mcp-submit

[`mcp-submit`](https://www.npmjs.com/package/mcp-submit) submits one
`server.json` to several directories at once. Run it from a checkout after
generating `server.json`:

```sh
npx mcp-submit --dry-run
npx mcp-submit --skip official,docker
```

Skip `official` (published directly, above) and `docker` (needs a manual PR,
below). The September 2026 run filed the mcp.so submission
([chatmcp/mcpso#3885](https://github.com/chatmcp/mcpso/issues/3885)) and
confirmed that both awesome-mcp-servers lists
([punkpeye](https://github.com/punkpeye/awesome-mcp-servers) and
[appcypher](https://github.com/appcypher/awesome-mcp-servers)) already list us.

## Claude Desktop Extensions (.mcpb bundle)

The Claude Desktop extensions directory takes a `.mcpb` bundle — a zip with a
`manifest.json` and the server binary. Manual build, first used for v1.8.0:

1. Download the `darwin_amd64` and `darwin_arm64` release archives and combine
   the binaries into a universal one:

   ```sh
   lipo -create signadot_amd64 signadot_arm64 -output bundle/server/signadot
   ```

2. Add a 512x512 `icon.png` at the bundle root.
3. Write `manifest.json` per the [MCPB spec](https://github.com/anthropics/mcpb/blob/main/MANIFEST.md):
   - `manifest_version`: `"0.3"`
   - `server.type`: `"binary"`, `command`: `"${__dirname}/server/signadot"`,
     `args`: `["mcp"]`
   - `SIGNADOT_API_KEY` in the server env, sourced from a `user_config` field
     marked `sensitive` and `required`
   - `platforms`: `["darwin"]`
4. Zip the bundle *contents* (so `manifest.json` sits at the zip root) into
   `signadot-mcp-<version>.mcpb`.
5. Submit via [Anthropic's directory form](https://forms.gle/tyiAZvch1kDADKoP9).

## Other directories

| Directory | How to submit |
|---|---|
| [GitHub MCP Registry](https://github.com/mcp) (also powers the VS Code MCP gallery) | Email partnerships@github.com, referencing the official-registry entry |
| [Docker MCP Catalog](https://hub.docker.com/mcp) | PR adding `servers/signadot/server.yaml` to [docker/mcp-registry](https://github.com/docker/mcp-registry) |
| [mcpservers.org](https://mcpservers.org) | Web form |
| [Glama](https://glama.ai/mcp/servers) | Claim the auto-indexed listing |
| [MCPCentral](https://mcpcentral.io) | Manual submission |
| [PulseMCP](https://www.pulsemcp.com) | Submissions paused as of September 2026 |
| [Smithery](https://smithery.ai) | Requires a public remote Streamable HTTP endpoint with OAuth — not available yet |
| Claude Connectors Directory | Same remote-endpoint requirement as Smithery — not available yet |
