@include default
@include sdk

## 9. The public surface of this SDK

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
