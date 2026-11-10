# Versioning and Minimal Version Selection

## Semantic versioning

Go modules use `vMAJOR.MINOR.PATCH` - the `v` prefix is mandatory:

- **MAJOR** - breaking changes to the public API.
- **MINOR** - backward-compatible additions.
- **PATCH** - backward-compatible fixes.

| Version | Stability |
| --- | --- |
| `v0.x.x` | unstable - no compatibility promises |
| `v1.x.x+` | stable - compatible within the major |
| Pre-release (`v1.5.0-beta.1`) | unstable, opt-in |

### Major version suffix

From `v2` onward the module path must carry `/vN`. Different majors become entirely separate modules, which is what lets `v1` and `v2` coexist in one build:

```go
// go.mod
module github.com/example/pkg/v2
```

Tags follow: `v2.0.0`, `v2.1.0`, ... `v0` and `v1` have no suffix.

### Special cases

- **Pseudo-versions** - for untagged commits: `v0.0.0-20210101120000-abcdef123456` (base version, timestamp, commit hash).
- **`+incompatible`** - marks `v2+` modules that never adopted the `/vN` convention.
- **gopkg.in** - always suffix-based with a dot, e.g. `gopkg.in/yaml.v3`.

## Minimal Version Selection (MVS)

Go resolves dependencies differently from npm, pip, or cargo: instead of picking the latest compatible version, it picks the **minimum version satisfying every requirement** in the graph. If module A requires `pkg@v1.2.0` and module B requires `pkg@v1.3.0`, MVS selects `v1.3.0` - the highest minimum, not the latest release.

### Why this design

- **Deterministic without a lockfile** - same `go.mod`, same build list, every machine. `go.sum` only verifies integrity.
- **High fidelity** - builds match what authors tested against, since the nearest compatible versions win rather than the freshest.
- **No solver** - the algorithm is a straightforward graph walk, not a constraint-satisfaction search with NP-hard corners.
- **Reproducible** - no "works on my machine" divergence between environments.

### Upgrade and downgrade semantics

- `go get pkg@v1.5.0` adds an edge to that version and re-runs MVS; only the minimally necessary versions move.
- `go get pkg@v1.2.0` prunes every version above `v1.2.0` from the graph, then walks backward to the latest remaining versions of the affected dependencies.

Because upgrades propagate minimally, expect `go get` to touch the fewest modules it can - a patch bump on one dependency should not ripple the tree.
