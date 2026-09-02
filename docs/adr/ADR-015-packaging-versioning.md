# ADR-015: Packaging and Versioning

## Status
Accepted

## Context

The SDK must follow standard Go module conventions for versioning and distribution (req 15). The version string is needed at runtime for the `User-Agent` header (see ADR-002) and must be automatically detected from module metadata rather than hardcoded.

## Decision

### Semantic Versioning

The SDK uses Semantic Versioning (MAJOR.MINOR.PATCH):

- **MAJOR**: Breaking API changes (new major version path in `go.mod`)
- **MINOR**: New features, backward-compatible
- **PATCH**: Bug fixes, backward-compatible

Versions are managed via Git tags following Go module conventions:

```
v2.0.0
v2.1.0
v2.1.1
```

### Automatic Version Detection

The SDK detects its version at runtime from Go module metadata using `runtime/debug.BuildInfo`. No hardcoded version constant:

```go
package marketdata

import "runtime/debug"

// Version returns the SDK version from Go module metadata.
// Falls back to "unknown" if build info is unavailable.
func Version() string {
    info, ok := debug.ReadBuildInfo()
    if !ok {
        return "unknown"
    }
    // When used as a dependency, the version comes from the module info
    for _, dep := range info.Deps {
        if dep.Path == "github.com/marketdataapp/sdk-go/v2" {
            return dep.Version
        }
    }
    // When running from the module itself (e.g., tests, examples)
    if info.Main.Path == "github.com/marketdataapp/sdk-go/v2" {
        return info.Main.Version
    }
    return "unknown"
}
```

This version is used in the `User-Agent` header: `marketdata-sdk-go/v2.1.0`

### Go Module Path

The module uses the `/v2` suffix per Go major version conventions:

```
module github.com/marketdataapp/sdk-go/v2
```

### Distribution

The SDK is distributed via Go modules (pkg.go.dev):

- Source on GitHub with tagged releases
- Discoverable at `pkg.go.dev/github.com/marketdataapp/sdk-go/v2`
- Users install with: `go get github.com/marketdataapp/sdk-go/v2`

### Changelog

A `CHANGELOG.md` in the repository root follows the [Keep a Changelog](https://keepachangelog.com/) format:

```markdown
# Changelog

## [Unreleased]

## [2.1.0] - 2026-03-15
### Added
- Options chain filtering by strike range

### Fixed
- Rate limit header parsing for edge cases

## [2.0.0] - 2026-02-01
### Added
- Initial v2 release
```

### License

MIT license. The `LICENSE` file is included in the repository root and distributed with the module.

## Consequences

### Positive
- Version auto-detected — no hardcoded strings to maintain
- Standard Go module conventions — familiar to Go developers
- SemVer communicates compatibility expectations clearly
- Changelog provides human-readable release history

### Negative
- `runtime/debug.ReadBuildInfo()` returns `"(devel)"` for unreleased builds
- Version detection adds minor complexity vs. a hardcoded constant
- `/v2` module path is required by Go conventions for major versions > 1

### Mitigations
- Fallback to `"unknown"` when build info is unavailable
- CI validates that tagged releases produce correct version strings
- Documentation explains the `/v2` import path requirement

## References

- Requirements: Section 15 (Packaging Requirements)
- [Go Module Version Numbering](https://go.dev/doc/modules/version-numbers)
- [Keep a Changelog](https://keepachangelog.com/)
- Related: ADR-002 (User-Agent header uses version)
