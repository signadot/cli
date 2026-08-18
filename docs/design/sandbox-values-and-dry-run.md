# `sandbox apply`: values documents and dry run

- Status: proposed, implemented on this branch
- Tracking: [ENG-1187](https://linear.app/signadot/issue/ENG-1187/signadot-sandbox-github-action-poc)
- Split in two: `--dry-run` is [#362](https://github.com/signadot/cli/pull/362), which
  stands on its own and is not contingent on any of this. The values document and the
  built-in template are [#361](https://github.com/signadot/cli/pull/361), stacked on it,
  and are a decision rather than a proposal to merge. This document covers both halves
  because they were designed together; the sections on the dry run describe what #362
  already does.
- Consumer: the [Signadot Sandbox GitHub Action](https://github.com/signadot/hackspace/tree/joe/ENG-1187/sandbox-github-action-poc/sandbox-action),
  whose design overview explains why this exists
- Supersedes an earlier plan to extract the CLI's templating and validation into a
  shared `libspec` module. That module is deleted; this is what replaced it.

## The problem

CI wants two things the CLI could not do.

**Describe a sandbox without authoring a template.** Every customer adopting PR
validation writes a `@{var}` spec template and maintains it. For the common case —
fork these workloads, with these images, in this namespace — the template is
boilerplate whose only variable content is the bit `--set` already provides. Worse,
the interesting parts are exactly the parts a template cannot express: the fork list
has a length that depends on the input, and the templating engine cannot iterate.

**See the spec without applying it.** `sandbox apply` renders and submits in one
step. A workflow that wants to show the spec in a PR comment, diff it against the
last run, or fail review before touching the API has nowhere to stand. Every other
tool in this space has a dry run; we did not.

Both were previously solved outside the CLI, by a Go program in the GitHub Action
that built specs itself. That put the rules for image resolution, env merging, and
naming in two places, and left a spec the Action accepted possibly different from one
`sandbox apply` accepts.

## What changes

Four things, all on `sandbox apply` and all additive. Existing invocations behave
exactly as before.

### 1. `-f` accepts a values document

A values document says what to fork, not how a spec is shaped:

```yaml
cluster: prod-eks
defaults:
  namespace: hotrod
  imageTemplate: ghcr.io/acme/{workload}:{short-sha}
forks:
  - workload: route
  - workload: frontend
    env:
      LOG_LEVEL: debug
```

`-f` routes on a discriminator: a document with a top-level `spec` is a sandbox spec
and takes the existing path untouched; anything else is values. Unknown fields are
rejected, so `envs:` fails rather than being silently dropped.

The schema is flat, and deliberately so: a CI integration assembles it from string
inputs without needing a YAML library or knowledge of the API's shape.

### 2. Values render through a built-in template

The values document is compiled into variables and rendered by the same `@{var}`
engine that renders a template of your own, against a template compiled into the
binary:

```yaml
name: "@{name[yaml]}"
spec:
  cluster: "@{cluster[yaml]}"
  forks: "@{forks[yaml]}"
  # ...
```

`signadot sandbox template show` prints it. Nothing about it is privileged: save it,
edit it, pass it to `-f`, and you have taken over the structure — which is also the
upgrade path off the values schema when it stops fitting.

This is a hybrid on purpose. The structural shape lives in a template, where it is
inspectable and where the existing engine already does the work. The parts a template
cannot compute — the fork list, resolved images, merged env, labels — are computed in
Go and injected as `@{var[yaml]}` values. `[yaml]` embeds a parsed structure rather
than a string, so a whole fork list arrives as a list. It is the mechanism that makes
one template serve any number of forks.

### 3. `--dry-run=none|client|server`

Follows `kubectl`, so the distinction between rendering locally and asking the server
to validate is the one people already know.

`--dry-run=client` renders, validates and prints the spec instead of applying it. It
runs **before** authentication, so it needs no credentials and fails identically
whether or not you are logged in. The output is a spec, so it can be reviewed,
diffed, and passed straight back to `-f`: rendering is a fixed point, which is what
lets a caller render in one step and apply exactly those bytes in another.

Local validation covers what can be checked without the cluster: a name that is
missing or too long, a fork with no workload or namespace, a field a patch
misspelled, and the API's rules for the reserved `signadot/` label prefix. The last
of those came out of dog-fooding, where a built-in label the API forbids surfaced
as a `400` on apply rather than as a message naming the key.

`--dry-run=server` is accepted and rejected with a message. It needs an apiserver
change — a `validateOnly` parameter on the create/update endpoints, or a validate
endpoint — since nothing supports it today. Scoped as a follow-up rather than
designed in absentia here.

### 4. Flags for the things CI knows and a file does not

| Flag | Why |
|---|---|
| `--name`, `--ttl` | Override what the document says, for values a workflow computes |
| `--patch FILE` | YAML merge patch after rendering: one odd field without owning the whole spec |
| `--ci-context auto\|github\|none` | Where the derived name and built-in labels come from |
| `--default-labels` | Opt out of the `signadot/*` labels |

CI context detection is what makes the templateless path complete. In a GitHub
Actions environment, `--ci-context=auto` derives a deterministic sandbox name from
the repository and PR — so every push updates the same sandbox in place — and stamps
the `signadot/github-repo` and `signadot/github-pull-request` labels the Signadot
GitHub App uses to delete the sandbox when the PR closes. `--ci-context=none` turns
detection off, which is how a two-step render-then-apply avoids deriving anything
twice.

Those two labels are a pair the API insists on having whole, and they are the only
keys it allows under the reserved `signadot/` prefix, so they are stamped as a set or
not at all: a build with no pull request gets neither. Both rules are enforced while
rendering, which is where dog-fooding found them — a third built-in label was enough
for the API to refuse the whole spec.

`--ci-context` is the seam other CI systems would arrive through, and it is worth
noting what that costs, since porting was a motivation for putting this work here.
Adding a provider means adding the variables it reads, not new rendering. The labels
describe the repository rather than the pipeline, so a CircleCI job on a GitHub repo
can carry the same two and keep the GitHub App integration. What does not port is a
VCS with no Signadot integration, where teardown falls back to TTL or explicit delete.

Names are normalised (lowercased, invalid characters replaced, hashed when too long)
on their way in from a flag, the CI context, or a values document — all of which are
inputs we compile. A name written in a spec document is left exactly as authored, so
adopting this cannot silently retarget an existing sandbox.

## Where the code lives

```
internal/render/            values → spec, independent of cobra
  values.go                 the values schema and its strict decoder
  compile.go                image templates, env merging, resource sigils, defaults
  context.go                CI detection, naming, built-in labels
  doc.go                    merge patch, null pruning, stable YAML, validation
  fork-deployment.yaml      the built-in template, embedded with go:embed
  template.go               compile + render
internal/command/sandbox/
  render.go                 -f routing, patching, output
  template.go               `sandbox template show`
```

`internal/utils/template.go` gained a `RenderTemplate` that takes bytes and an
explicit base directory, because a compiled-in template has no path for `@{embed:}`
to resolve against. `LoadUnstructuredTemplate` now sits on top of it, unchanged in
behaviour.

## Testing

`internal/command/sandbox/testdata/render/*` are golden fixtures: a values or spec
document, the environment and flags, and the exact rendered output. They came from
the Action's own golden tests, so the rendering they pin is rendering that was
already reviewed against a real repository's workflow. `TestRenderedSpecIsAFixedPoint`
asserts that re-rendering a rendered spec changes nothing, which is the property the
two-step CI flow depends on.

## Alternatives considered

**A separate `sandbox render` command.** Rejected: it would have to duplicate every
flag `apply` has, and it invites the two commands to drift. `--dry-run` on `apply` is
one code path with a switch at the end.

**A `libspec` shared module.** The rendering rules were extracted into a library so
the CLI and the Action could depend on the same code. It worked, but it bought
coordination across two repositories, a `replace` directive during development, and a
release process, to serve one consumer. Putting the logic in the CLI and having the
consumer shell out to it gets the same single implementation for none of that. The
extraction is deleted; its test coverage was salvaged into `internal/utils`.

**Templating that can iterate.** Adding loops to the `@{var}` engine would let a
template build the fork list itself, and remove the need for a values schema. That is
a language design project with a compatibility surface, to avoid a Go function that
builds a list. Not now.

**Keep the CLI minimal and compile in each integration.** The CLI gets `--dry-run` and
nothing else; each CI integration embeds its own template and computes the fork list,
the name and the labels itself. This is the live alternative to everything after
`--dry-run`, and the reason the two are separate PRs.

What it buys: no values schema and no built-in template committed to a CLI release
before we know whether the Action is the right shape, and nothing to keep compatible if
it is not. What it costs: roughly the 550 lines in `compile.go` and `context.go`
reimplemented per integration, in whatever language that integration is written in. A
CircleCI orb is YAML and shell, so it cannot reuse the Action's TypeScript — the second
provider either writes this a third time or moves it here after all, by which point two
integrations have shipped doing it differently.

## Follow-ups

1. **`--dry-run=server`.** Needs `validateOnly` on the apiserver's sandbox
   create/update, or a validate endpoint. The CLI flag already exists and reports
   that it is unavailable, so wiring it up later is a small change here. Scoped in
   [ENG-1203](https://linear.app/signadot/issue/ENG-1203/server-side-dry-run-for-sandbox-apply-validateonly).
2. **Values schema coverage.** It models what one dog-fooding pass needed. A second
   consumer will find gaps; the escape hatch (`template show`, edit, `-f`) is what
   makes that survivable in the meantime.
3. **Documentation.** The values schema and the built-in template need a page on
   docs.signadot.com before this is announced.
4. **Other CI providers.** `--ci-context circleci` and friends, once we decide which
   integration is next. CircleCI's PR number is the only genuinely fiddly part:
   `CIRCLE_PR_NUMBER` is set on forked PRs only, so it has to come from the trailing
   segment of the `CIRCLE_PULL_REQUEST` URL, which is absent on commits pushed before
   the PR existed. Naming already falls back to a short SHA, so that degrades rather
   than fails.
