# Visualizing the Dependency Graph

Go ships the raw graph; the ecosystem renders it.

## go mod graph - built-in

One line per edge, `module` then `requirement`, in `path@version` format:

```text
example.com/main github.com/google/uuid@v1.6.0
example.com/main golang.org/x/text@v0.3.7
github.com/google/uuid@v1.6.0 golang.org/x/sys@v0.0.0-20210615035016
```

## go mod why

The shortest import path from your code to a module - the tool for "why is this even here":

```bash
go mod why -m github.com/some/module
```

A common payoff: the module exists because of a single transitive import you can often drop.

## modgraphviz - static image

Pin it as a module tool and pipe `go mod graph` through it:

```bash
go get -tool golang.org/x/exp/cmd/modgraphviz@latest
go mod graph | go tool modgraphviz | dot -Tpng -o deps.png
```

Green nodes are versions MVS actually selected; grey nodes exist in the requirement graph but are not used. The visual split instantly shows which declared versions are dead weight.

## go-mod-graph - interactive explorer

`samber/go-mod-graph` is a web-based viewer with a zoomable graph, per-module weight indicators, a searchable module list, and a step-through MVS animation. Good for a meeting-room walkthrough of a dependency-heavy module.

## digraph - graph queries

`golang.org/x/tools`' `digraph` runs generic graph queries over `go mod graph` output:

```bash
go get -tool golang.org/x/tools/cmd/digraph@latest
go mod graph | go tool digraph reverse example.com/some/module
```

`reverse` names everything that (transitively) depends on the target - the exact input to a "what breaks if I remove this?" decision.
