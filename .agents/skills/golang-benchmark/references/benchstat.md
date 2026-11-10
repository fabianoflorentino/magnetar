# benchstat

`benchstat` folds many `go test -bench` runs into a statistically sound A/B comparison. A single run says nothing about variance; benchstat reports a confidence interval and a p-value so a real improvement survives scrutiny and plain luck does not.

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

## The basic loop

1. Write the benchmark — `b.Loop()` on Go 1.24+, the legacy `b.N` loop otherwise — in a `_bench_test.go` file.
2. Measure the baseline with `-count=10` or more and save the output.
3. Change one thing in the code.
4. Measure again with identical flags, machine, and load.
5. Compare: `benchstat old.txt new.txt`.

```bash
go test -run='^$' -bench=BenchmarkParse -benchmem -count=10 ./pkg/parser | tee old.txt
```

`-run='^$'` restricts the run to benchmarks only, keeping the session short. At least ten samples is the practical minimum before a confidence interval means anything.

Inputs are files of `go test -bench` output; label them with `label=path` to control column headers. The **first input is the base**; every later input is compared against it:

```bash
benchstat old.txt new.txt
benchstat baseline=old.txt optimized=new.txt
```

## Reading the output

```
goos: linux
goarch: amd64
pkg: myapp/pkg/parser
cpu: AMD Ryzen 9 5950X 16-Core Processor
          │   old.txt   │              new.txt               │
          │   sec/op    │   sec/op     vs base               │
Parse-32    4.592µ ± 2%   3.041µ ± 1%  -33.78% (p=0.000 n=10)
```

| Element | Meaning | What to watch |
| --- | --- | --- |
| `4.592µ` | Median across samples — robust against outliers | The reference number for this benchmark |
| `± 2%` | Half-width of the 95% confidence interval as a percentage of the median | `≤2%` stable; `>5%` too noisy to trust |
| `-33.78%` | Change of this input relative to the base | Negative = faster or smaller |
| `p=0.000` | Mann–Whitney U-test, non-parametric | `<0.05` statistically significant |
| `n=10` | Samples used | Should match `-count`; if not, inputs are misaligned |
| `~` | No statistically significant difference | Do not claim a change |
| `geomean` | Geometric mean of all rows in the table | Overall proportional change |

Units are normalized for display: `ns/op` renders as `sec/op` (with µ/m prefixes) and `MB/s` as `B/s` (with K/M/G prefixes).

### The `~` symbol

```
Parse-32    4.592µ ± 8%   4.481µ ± 7%  ~ (p=0.089 n=10)
```

The confidence intervals overlap — benchstat cannot separate the difference from noise. Either raise `-count` to 20+ to tighten the intervals, reduce the noise sources, or accept that this benchmark shows no measurable change. A `~` result is not a regression and not an improvement; it is an inconclusive measurement.

## Projection flags

These control how results are grouped into tables, rows, and columns.

| Flag | Default | Purpose |
| --- | --- | --- |
| `-table KEYS` | `.config` | Separate tables grouped by these keys |
| `-row KEYS` | `.fullname` | One row per different value of these keys |
| `-col KEYS` | `.file` | One column per value of these keys |
| `-ignore KEYS` | (none) | Remove keys from grouping; suppresses "benchmarks vary" warnings |
| `-filter EXPR` | (none) | Keep only matching benchmarks before grouping |

| Key | Extracts | Example value |
| --- | --- | --- |
| `.name` | Base name, sub-benchmark config stripped | `Parse` from `Parse/size=4k-16` |
| `.fullname` | Full name including config | `Parse/size=4k-16` |
| `.file` | Input label or file name | `old.txt` or `baseline` |
| `.config` | All header-level keys combined (`goos`/`goarch`/`pkg`/`cpu`) | — |
| `.unit` | Metric unit | `sec/op`, `B/op`, `allocs/op` |
| `/size` | A sub-benchmark parameter | `4k` from `Parse/size=4k` |
| `/gomaxprocs` | GOMAXPROCS, from `/gomaxprocs=N` or the `-N` suffix | `16` from `Parse-16` |
| `goos` | Operating system from the header | `linux`, `darwin` |
| `goarch` | Architecture from the header | `amd64`, `arm64` |
| `pkg` | Package path from the header | `myapp/pkg/parser` |
| `cpu` | CPU model from the header | `AMD Ryzen 9 5950X` |

Sort modifiers append to any key: `@alpha` sorts alphabetically, `@num` sorts numerically (understanding `2k`, `1Mi`), and `@(gob json)` imposes a fixed order while filtering to only the listed values.

### Common projections

```bash
# One row per benchmark, one column per input (the default)
benchstat old.txt new.txt

# Compare sub-benchmark parameters inside a single file
benchstat -col /format bench.txt

# Compact rows: drop sub-benchmark config from names
benchstat -col /format -row .name bench.txt

# Force column order instead of alphabetical
benchstat -col '/format@(gob json)' bench.txt

# Sweep GOMAXPROCS as columns
benchstat -col /gomaxprocs bench.txt

# One table per package
benchstat -table pkg old.txt new.txt

# Drop a dimension and silence the "varies in /gomaxprocs" warning
benchstat -row .name -ignore /gomaxprocs bench.txt

# Three versions, all against the first
benchstat v1=v1.txt v2=v2.txt v3=v3.txt

# Rows = name, columns = OS, tables = architecture
benchstat -row .name -col goos -table goarch results.txt
```

## Filter expressions

Filters apply before grouping, so they also prune the statistics. Match keys are the same `.name`, `.fullname`, `/size`, `/gomaxprocs`, `.file`, `.unit`, `goos`, `goarch`, `pkg` as above.

| Pattern | Meaning | Example |
| --- | --- | --- |
| `key:value` | Exact match | `goos:linux` |
| `key:"value"` | Exact match, spaces allowed | `pkg:"github.com/user/repo"` |
| `key:/regexp/` | Go regexp match | `.name:/Parse|Encode/` |
| `key:(a OR b)` | Any listed value | `goos:(linux OR darwin)` |
| `*` | Everything | `*` |
| `a AND b` (also `a b`) | Both must match — AND is implicit | `goos:linux goarch:amd64` |
| `-key...` | Negation | `-goos:windows` |
| `(...)` | Grouping | `(goos:linux OR goos:darwin) -pkg:/internal/` |

```bash
# Only Parse benchmarks
benchstat -filter '.name:Parse' old.txt new.txt

# Only the size=4096 sub-parameter
benchstat -filter '/size:4096' old.txt new.txt

# Exclude the Parallel family
benchstat -filter '-.name:/Parallel/' old.txt new.txt

# Linux on amd64 only
benchstat -filter 'goos:linux goarch:amd64' old.txt new.txt

# Named set of benchmarks
benchstat -filter '.name:(Parse OR Encode OR Decode)' old.txt new.txt

# Combination: the OSes you run, external packages excluded, one unit
benchstat -filter '(goos:linux OR goos:darwin) -pkg:/internal/ .unit:sec/op' old.txt new.txt
```

## Unit metadata

A `Unit` line in the benchmark output changes how a metric is compared:

```
BenchmarkSize 1 42 custom-bytes/op
Unit custom-bytes/op assume=exact
```

- `assume=exact` — for metrics that should not vary between runs (binary size, generated code size). Non-parametric statistics are disabled, any measured variance is flagged, and a single before/after pair is accepted.
- `assume=nothing` (default) — the standard median + Mann–Whitney U-test path, which needs multiple samples.

## Interleaving old and new

Sequential capture — all of old, then all of new — is exposed to systematic bias: thermal throttling accumulates, background load drifts, CPU frequency scaling adapts. Pre-compile both sides so compilation time never leaks into the numbers, then alternate:

```bash
go test -c -o old.test ./pkg/parser
# ...make the change...
go test -c -o new.test ./pkg/parser

for i in $(seq 1 10); do
    ./old.test -test.bench=BenchmarkParse -test.benchmem >> old.txt
    ./new.test -test.bench=BenchmarkParse -test.benchmem >> new.txt
done

benchstat old.txt new.txt
```

Precompilation matters: direct `go test -bench` invocations recompile first, and compilation time is unstable noise.

## How many runs?

| Scenario | Minimum `-count` | Why |
| --- | --- | --- |
| Quick local check | 6 | Rough confidence interval, fast feedback |
| Pre-merge comparison | 10 | Standard for moderate (>5%) changes |
| Small changes (<5%) | 20–30 | More samples narrow the interval |
| Shared CI runner | 20+ | Baseline variance is higher there |

Never rerun until `~` disappears — that is selection bias and it voids the statistics. Raise the count once and accept the result. Note that at α=0.05 roughly 5% of benchmark pairs will look significant by chance; do not chase false positives.

## Single-file summary

```bash
benchstat bench.txt
```

Reports the median and confidence interval of each benchmark in one capture. Use it to check measurement stability before a change, to find benchmarks needing more runs, or as a quick snapshot.

## Common pitfalls

| Pitfall | Problem | Fix |
| --- | --- | --- |
| `-count=1` | No variance, no statistics | `-count=6` minimum, prefer 10 |
| Laptop on battery | Throttling explodes variance | Wired power, fixed frequency |
| Browser/IDE open | Background CPU steals cycles | Close them or accept wider CIs |
| Rerunning until `~` goes away | P-hacking | One high-count run, accept it |
| Comparing across machines | Incomparable baselines | Same machine, same conditions |
| Sequential old-then-new | Systematic drift | Precompile, interleave runs |
| Ignoring ± >5% | Significance without stability | Fix the noise first |
| Unequal `-count` per input | Distorted p-value | Identical flags for all inputs |

For regression gating in CI pipelines (benchdiff, cob, gobenchdata), see [`ci-regression.md`](./ci-regression.md).