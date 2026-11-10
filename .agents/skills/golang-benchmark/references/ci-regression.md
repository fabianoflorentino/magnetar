# CI Benchmark Regression Detection

Wire benchmark comparison into CI — not onto developer machines. Local runs carry background-process, thermal, and frequency-scaling noise that makes local regression claims unreliable. Even shared runners show 5–10% variance; treat that as the floor and use statistical or relative methods to cut through it, or invest in dedicated runners for critical paths.

## benchdiff

Benchmarks two git refs and reports the delta through `benchstat`. Non-worktree refs are cached, so re-runs only re-measure the working tree. It also prevents macOS sleep during a run.

```bash
go install filippo.io/mostly-harmless/benchdiff@latest
```

```bash
# Compare the worktree against HEAD (default)
benchdiff -- -benchmem

# Compare two explicit refs
benchdiff -base-ref main -head-ref feature-branch

# Compare against a tag or commit
benchdiff -base-ref v1.2.0

# Everything after -- goes to go test
benchdiff -- -benchmem -count=10 -benchtime=3s

# Filter to one benchmark
benchdiff -- -benchmem -count=10 -bench=BenchmarkParse

# Target one package
benchdiff -- -benchmem -count=10 ./pkg/parser/...

# Drop stale cache after a rebase
benchdiff -clear-cache

# Combined: main vs worktree, ten iterations, critical benchmarks
benchdiff -base-ref main -- -benchmem -count=10 -bench='BenchmarkParse|BenchmarkEncode'
```

Best for: PR-vs-base comparisons in git workflows, where `benchstat` rigor plus caching keeps re-runs cheap.

## cob

Compares HEAD against HEAD~1 and fails the job when performance degrades beyond a threshold (20% by default).

```bash
go install github.com/knqyf263/cob@latest
```

```bash
# Default 20% threshold, HEAD vs HEAD~1
cob

# Tighten for critical paths
cob -threshold 10

# Compare against a named base
cob -base main

# Flag regressions only, ignore improvements
cob -only-degression

# Choose the compared metrics (default ns/op,B/op)
cob -compare "ns/op,B/op,allocs/op"

# Custom go test invocation
cob -bench-args "test -run '^$' -bench . -benchmem -benchtime=3s ./..."

# Opt out per-commit: include [skip cob] in the commit message
```

Caution and limits:

- `cob` runs `git reset` internally — commit your work before invoking it, and run it in CI only.
- It gates only if every benchmark passes; a failing benchmark skips the check.
- It compares single runs without `benchstat`-style statistics, so it is more noise-sensitive than `benchdiff`.

Best for: a simple post-commit gate where fast feedback outweighs statistical rigor.

## gobenchdata

A GitHub Action plus CLI that collects benchmark JSON, publishes it to `gh-pages`, and renders an interactive dashboard for long-term trends.

```bash
go install go.bobheadxi.dev/gobenchdata@latest
```

### CLI

```bash
# Parse go test -bench output into JSON
go test -bench=. -benchmem -count=5 ./... | gobenchdata --json bench.json

# Parse an existing capture
gobenchdata --json bench.json < bench.txt

# Tag the run with the git commit
gobenchdata --json bench.json --tag "$(git rev-parse --short HEAD)" < bench.txt

# Evaluate regression checks
gobenchdata checks eval bench.txt --checks-config .gobenchdata-checks.yml

# Generate and serve the static dashboard app
gobenchdata web generate ./dashboard-app
gobenchdata web serve ./dashboard-app

# Merge captures, prune to the latest N
gobenchdata merge old-bench.json new-bench.json > combined.json
gobenchdata prune --count 30 bench.json
```

### GitHub Action

```yaml
name: Benchmark
on: [push]
jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - name: Run benchmarks
        run: go test -bench=. -benchmem -count=5 ./... | tee bench.txt
      - uses: bobheadxi/gobenchdata@v1
        with:
          PRUNE_COUNT: 30
          GO_TEST_PKGS: ./...
          BENCHMARKS_OUT: bench.txt
          PUBLISH: true
          PUBLISH_BRANCH: gh-pages
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Regression checks on PRs

```yaml
checks:
  - name: "No major regressions"
    package: ./...
    benchmarks: [".*"]
    thresholds:
      - metric: NsPerOp
        max: 1.2 # fail if more than 20% slower
      - metric: AllocedBytesPerOp
        max: 1.3 # fail if more than 30% more allocations
  - name: "Critical path stability"
    package: ./pkg/parser
    benchmarks: ["BenchmarkParse.*"]
    thresholds:
      - metric: NsPerOp
        max: 1.1 # stricter: 10% cap for the hot path
```

### Dashboard configuration

```yaml
title: "My Project Benchmarks"
description: "Performance tracking dashboard"
chartGroups:
  - name: Parser
    charts:
      - name: Parse Performance
        package: myapp/pkg/parser
        benchmarks: ["BenchmarkParse.*"]
        metrics: [NsPerOp, AllocedBytesPerOp, AllocsPerOp]
  - name: Encoding
    charts:
      - name: Encode/Decode
        package: myapp/pkg/encoding
        benchmarks: ["Benchmark(Encode|Decode).*"]
        metrics: [NsPerOp, MBPerS]
```

Best for: trend tracking and visualization; pair it with benchdiff or cob for immediate gating.

## Tool selection

| Tool | Statistical rigor | Dashboard | Best for |
| --- | --- | --- | --- |
| benchdiff | High (`benchstat`) | No | Local dev and CI PR comparisons |
| cob | Low (single comparison) | No | Quick post-commit gate |
| gobenchdata | Medium (configurable checks) | Yes (Vue on gh-pages) | Long-term trends |
| benchstat raw | High | No (CSV-capable) | Custom pipelines, full control |

## Taming noisy neighbors

Shared CI hardware competes with other jobs, so expect 5–10% variance even when the machine looks idle. Sources: other jobs on the same CPU and memory, thermal throttling, differing runner hardware, kernel scheduling jitter, and disk I/O contention.

Counter-measures:

- **Statistical rigor** — `-count=10` or more through `benchstat`; a single run is meaningless.
- **Relative, same-job comparison** — run base and head in one CI job on one machine instead of comparing to historical absolutes. benchdiff does this by checking out both refs.
- **Dedicated runners** — self-hosted, nothing else running: eliminates noisy neighbors at infrastructure cost.
- **Conservative thresholds** — 20%+ on shared runners, ~10% on dedicated ones. GitHub-hosted runners show roughly 2–3% coefficient of variation in the best case; a <1% false-positive rate needs a gate of 7% or wider.
- **No retry-until-pass** — rerunning flaky benchmarks until green is selection bias; fix the noise source instead.

## Tuning self-hosted runners

These kernel and CPU settings are for dedicated benchmark runners only — never developer machines or shared servers.

```bash
# Fixed maximum frequency
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Disable Turbo Boost — Intel
echo 1 | sudo tee /sys/devices/system/cpu/intel_pstate/no_turbo

# Disable Turbo Boost — AMD
echo 0 | sudo tee /sys/devices/system/cpu/cpufreq/boost

# Pin benchmarks to cores, leaving headroom for the OS
taskset -c 2,3 go test -bench=. -count=10 ./...

# Disable SMT so logical cores don't share execution units
echo off | sudo tee /sys/devices/system/cpu/smt/control
```

A consolidated pre-run script:

```bash
#!/bin/bash
set -euo pipefail

echo "=== Configuring CPU for stable benchmarks ==="
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
echo 1 | sudo tee /sys/devices/system/cpu/intel_pstate/no_turbo 2>/dev/null || true
echo off | sudo tee /sys/devices/system/cpu/smt/control 2>/dev/null || true

echo "=== Running benchmarks on isolated cores ==="
taskset -c 2,3 go test -bench=. -benchmem -count=10 ./... | tee bench.txt
```