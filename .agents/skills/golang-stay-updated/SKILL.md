---
name: golang-stay-updated
description: "Golang ecosystem watch list — official sources (go.dev/blog, pkg.go.dev, tour.golang.org, golang-nuts), newsletters (Golang Weekly, Awesome Go Newsletter), communities (r/golang, gophers.slack.com, Go Forum, go.dev/wiki), blogs (Dave Cheney, Ardan Labs, Rob Pike), YouTube channels (Gopher Academy, GopherCon EU/UK), conferences, and Go contributors to follow on GitHub, X and Bluesky. Use when seeking Golang learning resources, discovering new libraries or tools, finding community channels or meetups, picking Go people to follow, or keeping up with Go language changes and releases. Not for querying a specific module's versions, docs, or vulnerabilities from the CLI (→ See `fabianoflorentino/golang-agent-skills@golang-pkg-go-dev` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "📰"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch WebSearch
---

# Staying current with Go

**Persona:** You are a Go engineer who keeps a lean, high-signal watch. A handful of curated sources beats fifty tabs: official announcements, one or two newsletters, a small set of people worth following.

**Modes:**

- **Recommend** — given the visitor's goals, suggest the smallest useful set: newsletters, accounts, channels. Sequential.
- **Research** — verify a Go change or release against the official sources (go.dev), and cross-check people/tools with `golang-pkg-go-dev`. Sequential.

**When to use:** finding learning resources, discovering libraries, picking people to follow, or catching up on changes. Not for querying versions/docs/CVEs (that is `golang-pkg-go-dev`).

## Official sources (the ground truth)

| Resource | Use |
| --- | --- |
| [go.dev](https://go.dev) | Tutorials, toolchain, releases |
| [pkg.go.dev](https://pkg.go.dev) | Package docs — query from the CLI with `golang-pkg-go-dev` |
| [go.dev/blog](https://go.dev/blog) | Release notes, design docs, language announcements |
| [tour.golang.org](https://tour.golang.org) | Interactive primer |
| [play.golang.org](https://play.golang.org) | Run snippets |
| [Discuss Go (golang-nuts)](https://groups.google.com/g/golang-nuts) | Official discussion |

## Newsletters

| Newsletter | Why |
| --- | --- |
| [Golang Weekly](https://golangweekly.com/) | The one to keep — curated articles, releases, tools |
| [Awesome Go Newsletter](https://go.libhunt.com/) | New libraries and tools signal |

## Communities

| Community | Why |
| --- | --- |
| [r/golang](https://www.reddit.com/r/golang) | 300K+ members; active Q&A and announcements |
| [Gophers Slack](https://invite.slack.golangbridge.org) | Real-time help across topic channels |
| [Go Forum](https://forum.golangbridge.org) | Slower, deeper threads |
| [go.dev/wiki](https://go.dev/wiki/) | Living index of resources and FAQs |

## People worth following

| Person | Where | Why |
| --- | --- | --- |
| Russ Cox (`rsc`) | [X @_rsc](https://x.com/_rsc), [Bluesky](https://bsky.app/profile/swtch.com) | Design decisions and release rationale |
| Brad Fitzpatrick (`bradfitz`) | [X @bradfitz](https://x.com/bradfitz), [Bluesky](https://bsky.app/profile/bradfitz.com) | The stdlib and ecosystem takes |
| Dave Cheney (`davecheney`) | [X @davecheney](https://x.com/davecheney), [blog](https://dave.cheney.net) | Performance and internals writing |
| Katherine Cox-Buday (`kat-co`) | [blog](https://www.ardanlabs.com) | Author of *Concurrency in Go* |
| Bill Kennedy | [Ardan Labs blog](https://www.ardanlabs.com/blog) | Production-grade training content |
| Mitchell Hashimoto (`mitchellh`) | [X @mitchellh](https://x.com/mitchellh), [Bluesky](https://bsky.app/profile/mitchellh.com) | Systems tooling |
| Steve Francia (`spf13`) | [X @spf13](https://x.com/spf13) | Cloud Native + Go project standards |
| Samuel Berthe (`samber`) | [X @samuelberthe](https://x.com/samuelberthe), [Bluesky](https://bsky.app/profile/samber.bsky.social) | The lo/mo/oops/slog library ecosystem and agent skills |
| Jaana Dogan (`rakyll`) | [X @rakyll](https://x.com/rakyll) | Runtime and observability |
| Mat Ryer (`matryer`) | [X @matryer](https://x.com/matryer) | Go Time founder, clear writing |
| Erik St. Martin (`erikstmartin`) | [X @erikstmartin](https://x.com/erikstmartin) | GopherCon co-creator |
| Brian Ketelsen (`bketelsen`) | [Bluesky](https://bsky.app/profile/brian.dev) | GopherCon co-creator, teaching |
| Carlisia Campos (`carlisia`) | [X @carlisia](https://x.com/carlisia) | Community and conference organizer |

Filter to 10–20 active accounts; the list is a starting point, not a quota.

## Blogs

| Blog | Author | Why |
| --- | --- | --- |
| [The Go Blog](https://go.dev/blog) | Go team | The changelog of the language |
| [dave.cheney.net](https://dave.cheney.net) | Dave Cheney | Internals and performance |
| [commandcenter.blogspot.com](https://commandcenter.blogspot.com) | Rob Pike | Language philosophy firsthand |
| [Ardan Labs](https://www.ardanlabs.com/blog) | Bill Kennedy | Team-scale Go practice |

## Video

| Channel | Content |
| --- | --- |
| [Go (official)](https://www.youtube.com/@golang) | The Go team |
| [Gopher Academy](https://www.youtube.com/@GopherAcademy) | Talks and tutorials |
| [GopherCon Europe](https://www.youtube.com/@GopherConEurope) | EU conference talks |
| [GopherCon UK](https://www.youtube.com/@GopherConUK) | UK conference talks |
| [Ardan Labs](https://www.youtube.com/@ArdanLabs) | Training and tips |
| [Golang Singapore](https://www.youtube.com/@golangSG) | Regional meetup/conf talks |
| [Learn Go Programming](https://youtube.com/learn_goprogramming) | Beginner path |

Attend or watch a GopherCon yearly — the release and ecosystem announcements concentrate there.

## Sustainable routine

1. One newsletter (Golang Weekly), read weekly.
2. go.dev/blog weekly — official announcements first.
3. 10–20 followed accounts; prune the silent.
4. pkg.go.dev (or `godig`) for library discovery on demand.
5. Gopher Slack for real-time questions.

Suggest additions or corrections via a GitHub issue on this repo.
