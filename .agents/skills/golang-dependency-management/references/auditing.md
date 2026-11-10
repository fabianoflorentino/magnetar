# Auditing Dependencies

An audit answers three questions: what is linked into the binary, what has known vulnerabilities, and what is stale. Separate tools answer each.

## Test-only vs binary dependencies

`go.mod` does not distinguish test-only from production dependencies; every module appears in the same requirement list, `// indirect` marking the transitive ones. The binary itself tells the truth: packages imported only by `*_test.go` files are compiled by `go test` but never linked into `go build` output - their modules still stay in `go.mod`.

With `go 1.17+` in `go.mod`, Go prunes the module graph: dependencies needed only by other modules' tests are excluded from the build graph, shrinking `go.mod` and skipping needless downloads.

Upgrades respect the same split:

```bash
go get -u ./...     # upgrade deps, EXCLUDING test-only
go get -u -t ./...  # upgrade deps, INCLUDING test-only
```

To confirm whether a big dependency actually lands in the binary, look at a size breakdown - if it never appears, it is test-only and costs you nothing at runtime.

## Vulnerability scanning with govulncheck

`govulncheck` reports known vulnerabilities that reach code you actually call. Unlike generic CVE scanners that flag every dependency regardless of usage, it traces from your code to the vulnerable function and reports only reachable findings.

```bash
govulncheck ./...                 # scan source
go tool govulncheck ./...         # same, when pinned as a Go 1.24+ tool
govulncheck -mode=binary ./bin/myapp
govulncheck -format json ./...    # machine-readable for CI
govulncheck -test ./...           # include test code in the analysis
```

The report names the GO-*/CVE id, the affected module, the fixed version, and the call path from your code to the vulnerable function. A vuln in a dependency your code never reaches is not reported, keeping the noise low. For CI wiring, see `golang-continuous-integration`.

## Stale dependencies with go-mod-outdated

`psampaz/go-mod-outdated` lists outdated direct dependencies with available updates:

```bash
go list -u -m -json all | go-mod-outdated -update -direct
go list -u -m -json all | go-mod-outdated -update -direct -ci     # fail CI when outdated
go list -u -m -json all | go-mod-outdated -update -direct -style markdown
```

Columns tell the story: CURRENT, WANTED (latest minor/patch), and LATEST (overall). The VALID TIMESTAMPS warning catches a telltale sign of a hijacked date: an "update" that is chronologically older than the version you hold.

## Binary bloat with goweight

`jondot/goweight` lists every package linked into the binary sorted by size contribution:

```bash
goweight          # by size
goweight --json   # for CI tracking
```

Modern alternative - `Zxilly/go-size-analyzer` (`gsa`) covers ELF, Mach-O, PE, and WebAssembly and renders interactive HTML/SVG reports:

```bash
go get -tool github.com/Zxilly/go-size-analyzer/cmd/gsa@latest
go build -o ./myapp ./cmd/myapp
go tool gsa -f html -o size-report.html ./myapp
```

Finding a fat dependency paying for 10% of the binary is usually the cue to evaluate a stdlib or lighter replacement.
