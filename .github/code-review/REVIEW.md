@include default

# SDK pull request rules

These rules apply to every Market Data SDK. The acceptance criteria they build
on are the SDK requirements document at
https://www.marketdata.app/docs/internal/sdk-requirements/. Where that document
and this file disagree, that document wins. Where this file and a general
review rule disagree, this file wins.

## 1. The version bump is a calculation, not an opinion

Two different questions get called "breaking". Keep them apart.

1. **Is the API change breaking?** One answer for every SDK, taken from the API
   schema diff: a parameter removed, a type changed, a default changed. It is
   decided upstream in the `api` repository, and it is not yours to decide.
2. **Does this pull request break THIS SDK's public surface?** A separate answer
   for each SDK. This is the question you answer, and it decides the bump.

The two answers are independent. An API change that is purely additive can
still break an SDK: when `dte` used to accept one value and now accepts a
range, a Java `dte(String)` has to become `dte(DteFilter)` and every caller
breaks, while a dynamically typed Python signature may not change at all. A
breaking API change can also need no SDK change at all.

**Do not form an opinion. Compare.**

Build one row for every public symbol the diff touches:

| symbol | before | after | breaking |
|--------|--------|-------|----------|

Read "before" from the `-` lines of the diff and from the base ref; read
"after" from the `+` lines. Then apply this test, without judgement:

- a public symbol that disappeared or was renamed: **breaking**
- a parameter removed, renamed, reordered, or made required: **breaking**
- a parameter type or a return type that changed: **breaking**
- a default value that changed: **breaking**
- the error type a call raises, throws or returns, changed: **breaking**
- a new public symbol, a new optional parameter, a widened accepted type:
  **not breaking**

Put the table in `evidence`. A breaking verdict with no table is an opinion,
and opinions are what this rule exists to remove: an accidental major release
is the failure it prevents.

State the result in `summary`, in this form:

`Version impact: major (1.3.0 -> 2.0.0)`, or `minor`, or `patch`, or `none`.

## 2. The bump arrives with the pull request

- a non-breaking change takes the next minor: 2.5 becomes 2.6
- a breaking change takes the next major: 2.5 becomes 3.0
- the decision belongs to each SDK on its own. The same API change can be a
  major here and a minor in the SDK next door.

A pull request that breaks the public surface carries its major bump with it.
The bump arrives as three things in the same pull request. A breaking change
that is missing any of them is `blocking`:

1. **A `CHANGELOG.md` entry under `## [Unreleased]`**, under a heading that
   names it as breaking: `### Removed (BREAKING)`, `### Changed (BREAKING)`.
   The release workflow extracts the release notes from this file and matches
   the `## [X.Y.Z]` heading exactly, so an entry that is absent here never
   reaches the release notes.
2. **A migration line** on every removal and every changed signature: the
   concrete replacement call, and every difference a caller will notice. The
   shape of the answer, a dropped column, a parameter that now has to be
   passed.
3. **The version impact, stated in the pull request description**: the word
   `major` and the two numbers. If the description claims a smaller bump than
   your table computes, that is a blocking finding, and your table is the
   evidence.

Do not ask the author to edit the version in the package metadata. The version
comes from that metadata, and the `tag-and-release` workflow promotes
`## [Unreleased]` and sets the number. The pull request's job is to make the
number unarguable, not to write it.

## 3. What needs no bump

Say this plainly when it applies, and do not ask for a CHANGELOG entry that the
change does not need:

- tests, test fixtures, CI configuration, lint configuration
- comments, docstring wording, formatting
- internal refactoring behind an unchanged public surface

A pull request that only adds tests needs no release. Tests that *find* a bug
do, and the fix carries its own entry.

## 4. The SDK stays aligned with the API

An API improvement that never reaches the SDKs is the failure this rule
prevents. The SDKs and the API stay aligned at all times.

- A pull request that adds support for an API capability covers the whole
  capability: every parameter and every accepted value the API documents, not
  the subset the author needed. Partial coverage is a finding, and you name
  exactly what is missing.
- The canonical REST documentation is authoritative for paths, parameters and
  response shapes. When the SDK and the REST documentation disagree, the SDK is
  wrong.
- Every capability needs a first-class method: `stocks` needs `prices`,
  `quotes`, `candles`, `earnings` and `news`; `options` needs `chain`,
  `expirations`, `quotes` and `lookup`; `funds` needs `candles`; `markets`
  needs `status`; `utilities` needs `status`, `headers` and `user`. Naming is
  idiomatic for the language.

## 5. A defect here is probably a defect in the others

These SDKs are six implementations of one contract, and defects travel between
them. The candle chunking defect is the example: the API treats `to=` as
inclusive, the SDKs assumed it was exclusive, and the boundary day came back
twice. It was present in four of the five SDKs.

When a pull request fixes a defect in logic that every SDK implements, say in
`summary` whether the same defect can exist in the sibling SDKs, and name them.
The logic this covers:

- date-range splitting and chunk merging
- pagination and batch fan-out
- CSV assembly, and the header row of a merged body
- response header parsing and rate-limit accounting
- no-data detection and the empty answer placeholder
- money parsing and decimal precision
- date and time normalisation

Report this as `should_fix`, never as `blocking`. The fix in front of you is
still correct. The purpose is that somebody opens the sibling issues.

## 6. Tests

- Unit tests mock every HTTP call. A new behaviour needs a test that fails
  without the change.
- **Every endpoint needs at least one integration test that calls the live
  API.** A mock proves only that the SDK parses the response the author
  expected, and that is the assumption that goes stale when the API changes. A
  pull request that adds or changes an endpoint method with no live test is a
  finding.
- An integration suite that skips when the token is absent reports success
  while it tests nothing. A change that makes a missing token skip rather than
  fail is `blocking`.
- An integration test asserts on the shape of the decoded answer: that the
  fields the SDK models arrived, and that they carry usable values. A test that
  checks only the status code passes against an endpoint that has silently
  started to answer with an empty body.
- Coverage is 100%. A new branch that nothing reaches is a finding, and so is a
  new branch with no test. Every coverage-ignore carries a comment that says
  why.

## 7. Documentation in the same pull request

A new public method, a changed parameter, a changed default and a changed
return type each need, in the same pull request:

- the docstring or the doc comment on the public method
- the resource page under `docs/`
- the README, when the method list or the quick start changes
- the `CHANGELOG.md` entry

## The public surface of this SDK

The surface is every exported identifier, meaning every name with an initial
capital, in `marketdata/` and its subpackages. `internal/` is not surface: the
compiler enforces that, so a change there is not breaking on its own.

Breaking, for this SDK:

- an exported identifier removed or renamed
- a function or method signature changed, including a changed parameter type
  and a changed return type
- a method added to an exported interface, which breaks every outside
  implementation of it
- an exported struct field removed, renamed or retyped
- a field added to an exported struct that callers construct with positional
  literals
- an exported constant whose value changes, when callers compare against it
- a changed error type, or a sentinel error that stops matching `errors.Is`

Not breaking: a new exported function, a new method on a concrete type, a new
struct field when the struct is documented as keyed-literal only.

**Go decides its major version in the import path.** A breaking change here
changes `module github.com/MarketDataApp/sdk-go/vN` in `go.mod` and every
internal import path in the same pull request. A pull request that breaks the
surface without the module path change is `blocking`: it cannot ship as a
major.

A Go tag is also permanent. `proxy.golang.org` reads the tag and
`sum.golang.org` logs its hash, so deleting the tag withdraws nothing, and the
only remedy for a wrong major is a later version that carries a `retract`
directive. Weigh the surface table accordingly.

`marketdata/surface_test.go` derives surface facts from the AST rather than from
a hand-written list. A pull request that changes the surface and leaves that
test passing by widening it, rather than by satisfying it, is a finding.
