# Signadot Agent Skills

Signadot has an [MCP server](https://www.signadot.com/docs/integrations/mcp) but
we also provide agent skills here.  Agent skills do not require authentication
or MCP. Unlike the authenticated MCP case, agent skills do not directly know about your
authenticated environment such as clusters, services, workloads, sandboxes, etc.  Instead
the agent skills are generic guidance for agent's to drive the Signadot CLI surface.

These skills work as slash commands in Claude Code and can also be used as
context for other AI coding assistants (Cursor, Windsurf, etc.).

## Skills

| Skill | Audience | What it covers |
|-------|----------|----------------|
| `signadot-cli` | Platform + Developers | Sandbox, route group, cluster, resource plugin, job, and smart test management via the `signadot` CLI |
| `signadot-local` | Developers | `signadot local connect/disconnect/status/proxy/override`, traffic recording and inspection |

## Try It Out

### Claude Code

**Option A: Per-project** — copy the skills into any project where you want them available:

```bash
# From your project root
mkdir -p .claude/skills
cp -r /path/to/agent-skills/signadot-* .claude/skills/
```

**Option B: Global** — make them available in all projects:

```bash
cp -r /path/to/agent-skills/signadot-* ~/.claude/skills/
```

Then in Claude Code, type:

- `/signadot-cli` — help with sandbox/routegroup/job management
- `/signadot-local` — help with local development workflow

Each skill accepts an optional argument to focus on a specific area:

```
/signadot-cli sandbox
/signadot-local override
```

### Cursor

Copy the SKILL.md files into your project's `.cursor/rules/` directory:

```bash
mkdir -p .cursor/rules
cp agent-skills/signadot-cli/SKILL.md .cursor/rules/signadot-cli.md
cp agent-skills/signadot-local/SKILL.md .cursor/rules/signadot-local.md
```

Cursor will use these as context when you ask Signadot-related questions.

### Other AI Agents

The SKILL.md files are self-contained markdown. Feed them as system prompts or
context to any LLM-based agent. The frontmatter (`name`, `description`) can be
stripped or used for routing.

## Structure

```
agent-skills/
├── README.md
├── signadot-cli/
│   └── SKILL.md          # CLI resource management
└── signadot-local/
    └── SKILL.md          # Local development workflow
```

## Maintenance

These skills encode CLI behavior and command syntax that changes with releases.
When CLI behavior changes, the corresponding SKILL.md should be updated in the
same PR.
