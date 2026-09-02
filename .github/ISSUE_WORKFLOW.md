# Issue Workflow

This document defines the process for triaging and resolving bug reports in
`MarketDataApp/sdk-go`. It is written to be followed by a maintainer, human or
automated.

Companion document: [BUG_FINDING.md](./BUG_FINDING.md) finds bugs proactively. This
document processes bugs that users report.

## Overview

```
Verify Permissions → New Issue → Validate → [Valid]      → Reproduce → Accept → Fix → Close
                                          → [Needs Info] → Request Info → Wait 7 days → Close
                                          → [Not a Bug]  → Explain → Close
```

---

## Step 0: Verify permissions

Before processing issues, confirm you can manage them.

```bash
gh api repos/MarketDataApp/sdk-go/collaborators/$(gh api user --jq '.login')/permission --jq '.permission'
```

| Result | Meaning | Action |
|---|---|---|
| `admin`, `maintain`, `write`, `triage` | Sufficient permission | Go to Step 1 |
| `read` | Read-only access | Stop. Ask a maintainer to elevate your access |
| Error: `404 Not Found` | Not a collaborator | Stop. You cannot manage issues |
| Error: `401 Unauthorized` | Not authenticated | Run `gh auth login` first |

Quick check — exits 0 when you can manage issues:

```bash
gh api repos/MarketDataApp/sdk-go/collaborators/$(gh api user --jq '.login')/permission --jq '.permission' \
  | grep -qE '^(admin|maintain|write|triage)$'
```

---

## Step 1: Validate the bug report

Run this checklist against every new report. The fields map directly to
[`ISSUE_TEMPLATE/bug.yml`](./ISSUE_TEMPLATE/bug.yml).

| # | Criterion | How to check | Pass | Fail |
|---|---|---|---|---|
| 1 | **API docs verified** | "API documentation verification" checkboxes | Both checked | Either unchecked |
| 2 | **Has reproduction code** | "Reproduction code" field | A real Go code block | Empty, pseudocode, or prose only |
| 3 | **Code is complete** | Look for client construction | Has `marketdata.NewClient(...)` plus the `import` block | Missing client setup or imports |
| 4 | **Names the resource and method** | "SDK resource" + "Method" | Both present, e.g. `stocks` / `Stocks.Candles` | Empty or vague |
| 5 | **Specifies SDK version** | "SDK version" | A concrete version, e.g. `v2.0.0` | Empty or "latest" |
| 6 | **Specifies Go version** | "Go version" | A concrete version, e.g. `go1.24.4` | Empty or vague, e.g. "1.x" |
| 7 | **Describes expected behavior** | "Expected behavior" | A clear statement | Empty or unclear |
| 8 | **Describes actual behavior** | "Actual behavior" | A clear statement, ideally with the error text or panic trace | Empty or unclear |

**Bonus signal, not required:** the "Support info" field. When the SDK returned an
API-produced error, the `SupportInfo()` block carries `request_id`, `request_url`,
`status_code` and `timestamp`. That identifies the exact upstream request and usually
settles whether the fault is in the SDK or the API. Ask for it whenever an error is
involved and the block is missing.

**Check the import path first.** A surprising share of reports are really "I am on v1".
If the reproduction imports `github.com/MarketDataApp/sdk-go` without `/v2`, they are
using the v1 SDK, whose API is entirely different. Point them at `docs/MIGRATION.md`.

### Decision

- **All 8 pass** → Step 2 (Reproduce)
- **Any fail** → Step 4 (Request more information)

---

## Step 2: Reproduce the bug

1. Create a scratch module, or add a test alongside the affected package.

   ```bash
   cd "$(mktemp -d)" && go mod init repro
   go get github.com/MarketDataApp/sdk-go/v2@vX.Y.Z   # the reported version
   ```

2. Use the reported SDK version. `go get ...@vX.Y.Z` pins it exactly; do not test against
   `main` and assume it matches.
3. Run it and compare against the reported "Actual behavior".
4. If it involves the live API, check `integration/discrepancy_test.go` first — several
   API-vs-documentation differences are already tracked there with links to the API's own
   issue tracker, and the answer may already be recorded.

### Decision

| Outcome | Next step |
|---|---|
| **Reproduces** — output matches the report | Step 3A (Accept) |
| **Does not reproduce** — the code works | Step 3B (Cannot reproduce) |
| **Different error** — fails, but not as reported | Step 4 (Request more information) |
| **API error, not SDK error** — the API itself returns the error | Step 3C (Not an SDK bug) |
| **Expected API behavior** — the SDK faithfully returns what the API sent | Step 3C (Not an SDK bug) |
| **User error** — the reproduction code is wrong | Step 3C (Not an SDK bug) |

> **Reproduces on Go 1.22 but not on stable (or the reverse)?** That is a real bug, not a
> non-repro. CI runs both legs for exactly this reason. Record which toolchain is affected
> and go to Step 3A.

---

## Step 3A: Accept as a bug

1. Add the label `accepted`.
2. Comment with the template below.
3. Go to Step 5.

```markdown
Thanks for the detailed report. I've reproduced this.

**Reproduction confirmed:**
- SDK version: [version]
- Go version: [version]
- GOOS/GOARCH: [platform]
- Behavior: [what you observed]

Working on a fix.
```

---

## Step 3B: Cannot reproduce

1. Add the label `needs-info`.
2. Comment with the template below.

```markdown
I wasn't able to reproduce this with the information provided.

**My environment:**
- SDK version: [version]
- Go version: [version]
- GOOS/GOARCH: [platform]

**What I observed:**
[What actually happened — worked correctly, different output, etc.]

Could you provide:
- [ ] The `SupportInfo()` block from the error — it contains the request id and URL we need, and never includes your API token
- [ ] Your client options, and any `MARKETDATA_*` environment variables or `.env` file in play
- [ ] The complete error text, or the full panic trace
- [ ] The exact version from your `go.mod` and the output of `go version`

I'll keep this open for 7 days for additional information.
```

---

## Step 3C: Not an SDK bug

1. Add the label `wontfix`.
2. Comment with the applicable template.
3. Close the issue.

### API issue, not the SDK

```markdown
Thanks for the report. After investigation this is behavior of the Market Data API itself rather than the Go SDK.

**What's happening:**
[Explain the API behavior]

**Suggested next steps:**
- Check the [API documentation](https://www.marketdata.app/docs/api) for this endpoint
- Contact Market Data support if you believe the API behavior is wrong
- Join the [Discord](https://discord.com/invite/GmdeAVRtnT) for community help

Closing as outside the SDK's scope. Please open a new issue if you find an SDK-specific problem.
```

### Expected API behavior

```markdown
Thanks for the report. After checking the [API documentation](https://www.marketdata.app/docs/api), this matches how the API is designed to work.

**What you're seeing:**
[Describe the behavior]

**Documentation reference:**
[Link or quote]

The SDK returns data exactly as the API provides it. If you believe the documentation is wrong, or the API should behave differently, please contact Market Data support or join the [Discord](https://discord.com/invite/GmdeAVRtnT).

Closing as working-as-designed.
```

### User error

~~~markdown
Thanks for the report. Reviewing the reproduction code, this looks like an issue in the calling code rather than a bug in the SDK.

**The issue:**
[Explain what's wrong]

**Suggested fix:**
```go
// Corrected code
```

**Documentation reference:**
[Link if applicable]

Closing this, but reopen if you believe there is still an SDK bug.
~~~

### Wrong major version

```markdown
Thanks for the report. The reproduction imports `github.com/MarketDataApp/sdk-go`, which is the **v1** SDK. v2 lives at a different import path:

```go
import "github.com/MarketDataApp/sdk-go/v2/marketdata"
```

```bash
go get github.com/MarketDataApp/sdk-go/v2
```

v2 is a complete rewrite — no global client, context on every method, typed errors. See [docs/MIGRATION.md](https://github.com/MarketDataApp/sdk-go/blob/main/docs/MIGRATION.md).

Closing, but please reopen or file a new issue if the behavior persists on v2.
```

---

## Step 4: Request more information

1. Add the label `needs-info`.
2. Comment, keeping only the items you actually need.
3. Check back in 7 days.

```markdown
Thanks for the report. To investigate I need some additional information:

- [ ] **API documentation verification**: Please confirm you've checked the [API documentation](https://www.marketdata.app/docs/api) and that the behavior differs from what it describes
- [ ] **Complete reproduction code**: A self-contained Go program including the `import` block and `marketdata.NewClient(...)`
- [ ] **Support info**: If the SDK returned an error, paste `SupportInfo()` — it carries the request id, URL, status code and timestamp, and never includes your API token
- [ ] **SDK version**: The version in your `go.mod`, or the output of `marketdata.Version()`
- [ ] **Go version**: The output of `go version`
- [ ] **Platform**: The output of `go env GOOS GOARCH`
- [ ] **Expected behavior**: What did you expect?
- [ ] **Actual behavior**: What happened? Include the full error text or panic trace
- [ ] **Additional context**: [Specify]

I'll keep this open for 7 days. Without a response I'll close it, but you're always welcome to reopen with the details.
```

### 7-day follow-up

```markdown
Closing due to inactivity. If you can provide the requested information, feel free to reopen or open a new issue with the additional details.
```

---

## Step 5: Fix the bug

1. [ ] **Write a failing test** next to the affected package and confirm it fails. Unit
       tests serve all HTTP from an `httptest.Server` — they never reach the network.
2. [ ] **Implement the minimal fix.**
3. [ ] **Confirm the new test passes.**
4. [ ] **Run the full suite and the coverage gate.** This repo requires **100% statement
       coverage** of `marketdata/...` and `internal/...`; CI fails below it.

       ```bash
       go test -race -p 1 ./...
       go test -race -p 1 -coverprofile=cov.out ./marketdata/... ./internal/...
       go tool cover -func=cov.out | tail -1        # must read 100.0%
       ```

       > Run the race suite **sequentially** (`-p 1`). Running it in parallel across
       > several sessions has previously exhausted memory on developer machines.

5. [ ] **Check the minimum toolchain still builds**: `GOTOOLCHAIN=go1.22.12 go test ./...`.
6. [ ] **Check formatting and lint**: `gofmt -l .` (must print nothing) and
       `golangci-lint run`. Use the pinned release binary — see the note in
       `.github/workflows/tests.yml`; a locally built golangci-lint links against a
       different toolchain and reports phantom stdlib errors.
7. [ ] **If the fix changes an illegal parameter combination**, extend the corpus in
       `internal/negcompile/testdata/` so the combination is asserted not to compile.
8. [ ] **If the fix touches live-API behavior**, run the integration suite:

       ```bash
       MARKETDATA_TOKEN=... go test ./integration/... -tags=integration -count=1
       ```

9. [ ] **Add a CHANGELOG entry** under `## [Unreleased]`.
10. [ ] **Commit** as `fix: description (closes #NNN)`.
11. [ ] **Open a PR** against `main`. Integration tests run on every PR.

Examples:

- `fix(stocks): decode BulkCandles when the API omits the symbol array (closes #45)`
- `fix(options): return an empty Expirations rather than nil on an empty list (closes #67)`

---

## Step 6: Close the issue

1. GitHub auto-closes from a `closes #NNN` commit message once merged.
2. If it did not, close it by hand with a comment.

~~~markdown
Fixed in [commit or PR link].

This ships in the next release. To use it immediately, point your module at `main` with a `replace` directive:

```
replace github.com/MarketDataApp/sdk-go/v2 => github.com/MarketDataApp/sdk-go/v2 main
```

or pin the pseudo-version:

```bash
go get github.com/MarketDataApp/sdk-go/v2@main
```
~~~

---

## Labels reference

| Label | Meaning | When to apply |
|---|---|---|
| `bug` | Default label from the template | Automatic on new issues |
| `accepted` | Validated and reproduced | After successful reproduction |
| `needs-info` | Waiting on the reporter | Report incomplete, or cannot reproduce |
| `wontfix` | Not a bug, or will not be fixed | When closing as not-a-bug |
| `dependencies` | Dependency update | Automatic on Dependabot PRs |

---

## CLI reference

```bash
# Labels
gh issue edit NUMBER --add-label "accepted"
gh issue edit NUMBER --add-label "needs-info"
gh issue edit NUMBER --remove-label "bug"

# State
gh issue close NUMBER
gh issue reopen NUMBER

# Comment and inspect
gh issue comment NUMBER --body "Comment text here"
gh issue view NUMBER

# Lists
gh issue list --label "bug"
gh issue list --label "needs-info"
```

---

## Examples

### Example A: valid bug report

**Issue #42** — resource `stocks`, method `Stocks.Candles`, complete reproduction code
with imports and `NewClient`, expected "returns candle data", actual "panic: runtime
error: invalid memory address", SDK `v2.0.0`, Go `go1.24.4`.

**Action:** passes all criteria → reproduce → accept and fix.

---

### Example B: incomplete report

**Issue #43** — resource `options`, method `Options.Chain`, reproduction code reads "I
called the chain method and it broke", expected "it should work", actual "it doesn't
work", SDK version empty, Go version "1.x".

**Action:** fails criteria 2, 3, 5, 6, 7, 8 → request more information, naming each
missing item.

---

### Example C: not a bug (API behavior)

**Issue #44** — `stocks` / `Stocks.Quote`, complete code, expected "should return the
after-hours price", actual "returns the regular session price".

**Investigation:** the API returns regular-session prices by default.

**Action:** close as "Not an SDK bug" with a pointer to the API documentation.

---

### Example D: expected API behavior

**Issue #45** — `stocks` / `Stocks.Earnings`, complete code, expected "percentages like
`5.2` for 5.2%", actual "returns `0.052`", both docs checkboxes checked.

**Investigation:** the API documents percentage fields as decimals (`0.052` = 5.2%). The
SDK passes the response through unchanged.

**Action:** close as "Expected API behavior", quoting the documentation.

---

### Example E: wrong major version

**Issue #46** — reproduction imports `github.com/MarketDataApp/sdk-go` and calls
`api.StockQuote().Symbol("AAPL").Get(ctx)`, which no longer exists.

**Investigation:** that is the v1 builder API. The reporter is on the v1 module.

**Action:** close with the "Wrong major version" template and a link to
`docs/MIGRATION.md`.

---

### Example F: toolchain-specific failure

**Issue #47** — `stocks` / `Stocks.Candles`, reproduces on Go 1.22, works on Go 1.24.

**Action:** this is a real bug. `go.mod` declares `go 1.22`, and CI runs that leg
precisely so the floor keeps working. Accept it and write the regression test so it runs
on both legs.
