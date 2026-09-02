# Security Policy

## Reporting a Vulnerability

**This is a public repository. Do not open a public GitHub issue for a security
vulnerability** — that discloses it to everyone before a fix is available.

Instead, report privately through GitHub's **Private Vulnerability Reporting**:

1. Go to the **Security** tab of this repository.
2. Click **Report a vulnerability**.
3. Describe the problem, including steps to reproduce, affected version(s), and
   the impact.

We will acknowledge the report, keep you informed as we investigate, and
coordinate the disclosure timeline and a fixed release with you. Please give us
a reasonable window to ship a fix before any public disclosure.

## Supported versions

| Version | Supported |
|---|---|
| `v2.x` (`github.com/MarketDataApp/sdk-go/v2`) | ✅ |
| `v1.x` (`github.com/MarketDataApp/sdk-go`) | ❌ — superseded by v2; see `docs/MIGRATION.md` |

## Scope

This repo is the **Market Data Go SDK** — a client library that consumers add to
their own Go modules. It runs on the consumer's machine or their servers, not on
Market Data infrastructure. The security concerns that matter here are therefore
about how the library treats *its consumers*:

- **Credential handling** — the caller's API token must never be logged
  verbatim, leaked in error messages, or written to disk. Tokens are redacted
  (`redactToken`) and query strings are stripped from logged URLs
  (`redactQuery`). Regressions here are in scope.
- **Transport security** — TLS is validated by default and the SDK exposes no
  skip-verify option. Anything that weakens this is in scope.
- **Injection into outbound requests** — request building that lets caller input
  smuggle headers, path segments, or query parameters it shouldn't. The configured
  API version must be a single path segment for this reason — `/`, `\` and `..` are
  rejected at client construction (`marketdata/config.go`).
- **Decoding safety** — the `encoding/json` response path handling hostile or
  malformed API responses without resource exhaustion, unbounded allocation, or
  panics a consumer cannot defend against. Response size limits
  (`ResponseTooLargeError`) are part of this.
- **File output** — `SaveToFile` writing where the caller did not intend, or with
  unsafe permissions.
- **Supply-chain integrity of the published module** — the release pipeline
  (`.github/workflows/tag-and-release.yml`) and the contents of the tagged tree.

Out of scope:

- **The Market Data API backend** itself. Report API/server vulnerabilities
  through the API's own channel, not here.
- **The `examples/` directory.** The TUI example apps (`stockterm`, `optionterm`,
  `tuitest`) are separate modules with their own `go.mod` and their own
  third-party dependencies. They are reference code, are never imported by the
  SDK, and are not part of the published module.

> **The published module has no third-party runtime dependencies.** It is built
> entirely on the Go standard library, so there is no dependency tree to audit
> and no transitive advisory surface. A vulnerability in the Go standard library
> is fixed by upgrading your Go toolchain, not by a release here — though we will
> raise the minimum Go version if a stdlib fix requires it. `govulncheck` runs on
> every push and pull request and must be clean.

## How this module is published

Go has no package registry. `proxy.golang.org` reads the git tag directly, so
**the tag is the publication**, and `sum.golang.org` records the module hash in a
public transparency log.

Two consequences matter for security:

- **A published version cannot be withdrawn.** Deleting a tag on GitHub does not
  remove the version from the proxy or the checksum database. If a release ever
  ships a vulnerability, the fix is a *new* version, plus a `retract` directive in
  `go.mod` so the toolchain stops selecting the bad one. The bad version stays
  downloadable forever. See `.github/RELEASE_PROCESS.md` §6.
- **The checksum database is a defence.** Because `sum.golang.org` pins the hash
  of every version, a tag that is moved or a repository that is tampered with
  after publication causes a verification failure on the consumer's machine
  rather than a silent substitution. This is also why the `v1.2.0` tag must never
  be moved or deleted.

## Security Fix Policy

This policy governs how security fixes are applied to this repository, including
fixes made by automated agents (e.g. Claude Code) working in the repo. It sorts
every security fix into one of two tiers.

The dividing line for a **library** is *consumer compatibility*. A fix that any
consumer can pick up by upgrading, with no source or behavior change on their
side, is low-risk. A fix that forces consumers to change their code, recompile
against a changed API, or adapt to changed runtime behavior is a breaking change
and follows SemVer — those get the maintainer gate.

### Tier 1 — Fix immediately (no approval needed)

Security fixes that are **API- and behavior-compatible for legitimate
consumers**. Existing callers keep compiling and keep working the same way after
upgrading; only the vulnerability is closed.

These may be fixed, tested, and committed right away. Every Tier 1 fix must be
called out in its commit message, in `CHANGELOG.md`, and in the summary reported
to the maintainer, so nothing ships silently.

Typical Tier 1 fixes:

- Tightening credential redaction, or plugging a token/secret/PII leak into logs
  or error messages
- Fixing injection in request building (header/path/query smuggling) where valid
  caller input is unaffected
- Hardening the response-decoding path against malformed or hostile API responses
  (bounds, size limits, nil handling)
- Correcting a logic flaw in an existing security check without changing its
  public contract
- Hardening `internal/` packages (transport, retry, rate limit, status cache,
  dotenv) that consumers cannot import or depend on
- Fixing the release pipeline or CI workflows

### Tier 2 — Requires maintainer approval first

Any security fix that **breaks consumer compatibility or changes observable
runtime behavior**. These must NOT be applied unilaterally. The agent or
contributor stops, writes up the issue, the proposed fix, and the specific
consumer impact, and waits for the maintainer's approval before proceeding.

A fix is **Tier 2** if it does any of the following:

- Removes, renames, or changes the signature of any **exported** identifier in
  `marketdata/...` (SemVer major)
- **Adds a third-party dependency to the published module.** Being standard-library
  only is a deliberate property of this SDK and a security property in its own
  right; taking on a dependency is never an agent's call
- Tightens input validation so that requests the SDK previously accepted are now
  rejected (could break existing callers)
- Changes a user-visible default (request or connect timeout, retry count or
  backoff, rate-limit behavior, base URL, API version, startup validation,
  `.env` loading)
- Changes a response type's shape, or the error taxonomy that consumers match on
  with `errors.Is` / `errors.As`
- Changes a sealed union used for compile-time parameter exclusivity (ADR-017),
  since that breaks callers at compile time
- Raises the minimum Go version in `go.mod`
- Changes the module path — which for Go means a new major version and a new
  import path for every consumer

### Classification rules

- **When in doubt, it's Tier 2.** If it is unclear which tier a fix falls into,
  treat it as Tier 2 and ask for approval.
- **No urgency exception.** Even for a critical, actively-exploitable
  vulnerability, a compatibility-breaking (Tier 2) fix waits for maintainer
  approval. Flag the urgency loudly, propose the fix, and wait. The maintainer is
  always the gate for changes that break consumers. (If a break is genuinely
  unavoidable to close a critical hole, that's a maintainer decision about cutting
  a major version — not an agent's.)

### Release of security fixes

Tiering governs *what* may be changed; the repo's normal release rules govern
*what ships to consumers*. A Tier 1 fix may be committed to a branch and merged
via the usual PR flow. **Cutting the tag that publishes it** — running
`tag-and-release.yml` — requires explicit maintainer confirmation, exactly like
every other release. Automated agents never cut or publish a release on their
own.

This matters more in Go than elsewhere: there is no staging feed and no unlist.
The moment the tag exists and one person fetches it, the version is public and
permanent.
