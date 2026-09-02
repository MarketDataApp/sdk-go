# Go SDK Release Process

This document defines the release process for `MarketDataApp/sdk-go`. The module is
[`github.com/MarketDataApp/sdk-go/v2`](https://pkg.go.dev/github.com/MarketDataApp/sdk-go/v2).

## 1. What "publishing" means for Go

There is no package registry. The other Market Data SDKs push a built artifact to npm,
PyPI, Maven Central, NuGet or Packagist. Go has none of that:
[proxy.golang.org](https://proxy.golang.org) clones the repository and reads the git tag
the first time anyone asks for a version.

**The tag is the publication.** Nothing is uploaded. That has three consequences that
shape everything below.

| | |
|---|---|
| **Publishing is instant** | The moment `v2.0.1` exists and one person fetches it, it is public. |
| **Publishing is irreversible** | The proxy is append-only and `sum.golang.org` records the hash in a transparency log. Deleting the tag on GitHub does **not** withdraw the version. |
| **Validation must happen before the tag** | There is no "unlist", no staging feed, and no window in which to reconsider. |

The only remedy for a bad release is to publish a later version that
[`retract`](https://go.dev/ref/mod#go-mod-file-retract)s it. The bad version stays
downloadable forever; `retract` only stops the toolchain from selecting it.

## 2. Scope and versioning

`2.0.0` is published. The public API is covered by semantic versioning from that version
on, so the version number is a promise:

| Change | Version |
|---|---|
| Bug fix, no API change | `2.0.Z` |
| New API, nothing removed or altered | `2.Y.0` |
| Anything a caller must react to | `3.0.0` |

Removing an exported identifier, renaming one, changing a signature, tightening a return
type, or altering documented behaviour all count.

> **A major bump is expensive in Go.** From v2 onward the major version lives in the
> module path itself. `3.0.0` means editing `go.mod` to
> `github.com/MarketDataApp/sdk-go/v3`, rewriting every internal import, and leaving v2
> users on a path that never receives the change. Treat a major bump as a fork of the
> import path, not as a version number.
>
> The release workflow refuses a version whose major disagrees with the module path, so
> dispatching `3.0.0` against a `/v2` module fails in the first job rather than cutting an
> unusable tag.

## 3. Release preparation

1. Confirm `main` is current and CI is green for the commit you intend to release.

2. **Promote the CHANGELOG section.** `CHANGELOG.md` is the single source of truth for
   release notes. In a normal PR:

   - Change `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD`.
   - Add a fresh, empty `## [Unreleased]` section above it.
   - Update the link-reference block at the bottom of the file: point `[Unreleased]` at
     `compare/vX.Y.Z...HEAD` and add an `[X.Y.Z]` entry.
   - Confirm every breaking change has migration guidance in `docs/MIGRATION.md`.

   > **The release workflow matches `## [X.Y.Z]` exactly** (Keep a Changelog bracket
   > format). A `## vX.Y.Z` heading will not be found, and the release fails in the
   > `validate` job before any tag is created.

3. Confirm `README.md` and `docs/` describe the behaviour you are about to ship.

4. Merge that PR to `main`.

## 4. Cut the release

One workflow drives the whole release: **Tag and Release**
(`.github/workflows/tag-and-release.yml`). Go to Actions → "Tag and Release" → "Run
workflow", and fill in:

| Input | Value |
|---|---|
| **version** | `X.Y.Z` (no `v` prefix) |
| **ref** | `main` (or a specific commit SHA) |
| **prerelease** | `false` unless this is a prerelease |
| **confirm** | `RELEASE` exactly |

Nothing else triggers this workflow. Pushing commits, merging PRs and pushing branches
never reach it.

The run proceeds through four gated stages. Each must pass before the next starts.

1. **`validate`** — seconds, and deliberately first so a typo costs no runner time. Checks
   that `confirm` is exactly `RELEASE`, that the version is semver without a `v` prefix,
   that the major agrees with the module path in `go.mod`, that the tag does not already
   exist on origin, and that `CHANGELOG.md` has a `## [X.Y.Z]` section. It prints the
   release notes it extracted so you can read them in the log before anything happens.

2. **`gate`** — calls `tests.yml`, the ordinary CI suite, against the exact ref: unit
   tests on Go 1.22 and stable, the 100% coverage gate, lint, `govulncheck`, the
   negative-compile corpus, the three example apps, and the **live integration suite**.
   The suite is called, not copied, so the release gate and everyday CI can never drift
   apart.

   > Integration is forced on. On an ordinary push `tests.yml` skips it; a release is
   > exactly the moment it must not be skipped. A missing `MARKETDATA_TOKEN` fails the
   > release rather than quietly reporting a suite that ran nothing.

3. **`release`** — **the point of no return.** Resolves `ref` to a concrete SHA, re-checks
   that the tag still does not exist, re-extracts the CHANGELOG section, then creates the
   tag and the GitHub Release `vX.Y.Z` pointing at that SHA.

4. **`verify`** — polls `proxy.golang.org` until it serves the new version (the first
   request is what makes the proxy fetch it), then creates a throwaway module, resolves
   the SDK from the public proxy, and asserts `marketdata.Version()` reports the version
   just released. Before the tag existed that call returned `unknown`, so this genuinely
   proves the published module, not the local checkout.

### Why no workflow run appears for the release

`tests.yml` has a `release: published` trigger, and it stays silent here. GitHub does not
start a workflow run from an event raised by the default `GITHUB_TOKEN`, as a guard
against recursive triggering.

That is fine and intended: the `gate` job already ran that same suite against the same
commit, minutes earlier. Do not "fix" the silent trigger by re-running tests after the
tag — by then the version is public, so a failure would be a discovery, not a gate.

### The CHANGELOG is written by hand, never by a workflow

There is deliberately no workflow that writes back to `CHANGELOG.md`. Promote
`## [Unreleased]` yourself in the release PR, as §3 describes. That is the only supported
path.

## 5. Post-release checks

The `verify` job already proves the first two. Check the rest by hand.

1. The GitHub Release exists with the notes taken from `CHANGELOG.md`.
2. A clean module resolves it:

   ```bash
   cd "$(mktemp -d)" && go mod init smoke
   go get github.com/MarketDataApp/sdk-go/v2@vX.Y.Z
   ```

3. `https://pkg.go.dev/github.com/MarketDataApp/sdk-go/v2@vX.Y.Z` renders the godoc.
   pkg.go.dev indexes asynchronously and can lag the proxy by up to an hour; a 404 shortly
   after a release is normal and needs no action.
4. `go get github.com/MarketDataApp/sdk-go` (no `/v2`) still resolves to `v1.2.0`. The v1
   line is served from the `v1` branch and the immutable `v1.2.0` tag, and no v2 release
   should ever disturb it.

## 6. Rollback and hotfix

A published version cannot be replaced, unpublished, or hidden.

1. Stop any promotion messaging.
2. Ship a patch release `vX.Y.(Z+1)` from `main` with the targeted fix, through the normal
   process above.
3. If the bad version is actively harmful, add a `retract` block to `go.mod` in that patch
   release so the toolchain stops selecting it:

   ```go
   retract (
       v2.0.1 // Published in error: <reason>.
   )
   ```

   `retract` is advisory. It stops `go get` from *choosing* the version; anyone who pins it
   explicitly still gets it, and the module cache and proxy keep serving it forever.
4. Record the root cause and the remediation in the next `CHANGELOG.md` entry.

## 7. Repository state this process assumes

| Item | State |
|---|---|
| `MARKETDATA_TOKEN` secret | required — the release gates on the live integration suite |
| `CODECOV_TOKEN` secret | optional; its absence leaves the coverage badge empty and breaks nothing |
| `main` branch protection | none at time of writing; the `confirm: RELEASE` input and the gated stages are the controls |
| `v1` branch | must never be deleted — it is what keeps the v1 line browsable |
| `v1.2.0` tag | must never be moved or deleted — it is what keeps v1 installable |

> **Never create another repository named `sdk-go` under the `MarketDataApp` account.**
> The module path is frozen at `github.com/MarketDataApp/sdk-go/v2`; changing it later
> would require a v3.
