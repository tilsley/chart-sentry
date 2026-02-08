# Architecture & Implementation Details

## Diffing Implementation

### Hybrid Approach: dyff + go-difflib

We use a **hybrid diffing strategy** that automatically uses the best tool available:

1. **Primary: dyff CLI** ([homeport/dyff](https://github.com/homeport/dyff))
   - Kubernetes-aware semantic YAML diffing
   - Shows structured field paths (e.g., `spec.replicas`)
   - Resource type annotations (e.g., `apps/v1/Deployment/my-app`)
   - Semantic understanding (e.g., "one document added")
   - Better readability for Kubernetes manifests

2. **Fallback: go-difflib** ([pmezard/go-difflib](https://github.com/pmezard/go-difflib))
   - Traditional line-by-line unified diff
   - No external dependencies required
   - Always available as a fallback

```go
func computeDiff(baseName, headName string, base, head []byte) string {
    // Try dyff first (better for YAML/Kubernetes manifests)
    if diff, ok := tryDyffDiff(baseName, headName, base, head); ok {
        return diff
    }

    // Fallback to line-based diff
    return lineDiff(baseName, headName, base, head)
}
```

### Installation

**dyff (optional but recommended):**
```bash
# macOS
brew install dyff

# Linux
wget -qO- https://github.com/homeport/dyff/releases/latest/download/dyff_linux_amd64.tar.gz | tar xz
sudo mv dyff /usr/local/bin/

# Or use Go
go install github.com/homeport/dyff/cmd/dyff@latest
```

**Benefits:**
- ✅ Works out-of-the-box with line-based diff
- ✅ Better diffs if dyff is installed
- ✅ No hard dependency - graceful degradation
- ✅ Kubernetes-aware semantic diffing with dyff
- ✅ Easy to test both modes

**Trade-offs:**
- ⚠️ dyff output format may change between versions
- ⚠️ Subprocess overhead (minimal, ~10-50ms per diff)

## Code Quality & Linting

### Linters Configured

#### 1. golangci-lint
Comprehensive Go linter aggregator with 30+ linters enabled:

**Categories:**
- ✅ Error checking (errcheck, errorlint, goerr113)
- ✅ Code style (revive, stylecheck, gocritic)
- ✅ Security (gosec)
- ✅ Performance (prealloc, gocritic)
- ✅ Best practices (unparam, unconvert, wastedassign)

**Run:**
```bash
make lint-go
```

#### 2. go-arch-lint
Hexagonal architecture enforcement:

**Rules:**
- ✅ Domain cannot import adapters
- ✅ Domain cannot import platform
- ✅ Ports cannot import adapters
- ✅ App cannot import adapters directly
- ✅ Adapters are independent (no cross-adapter imports)

**Run:**
```bash
make lint-arch
```

### All Linters

```bash
# Run all linters (fmt, vet, golangci-lint, go-arch-lint)
make lint
```

## Hexagonal Architecture

### Layers

```
                    ┌─────────────────┐
                    │      cmd/       │
                    │  (Entry Point)  │
                    │ Composition Root│
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             ▼
     ┌────────────────┐           ┌────────────────┐
     │  Adapters IN   │           │  Adapters OUT  │
     │                │           │                │
     │  github_in     │           │  github_out    │
     │  (webhook)     │           │  source_ctrl   │
     │                │           │  helm_cli      │
     └───────┬────────┘           └────────┬───────┘
             │                             │
             │    ┌──────────────────┐     │
             └───▶│      Ports       │◀────┘
                  │  (Interfaces)    │
                  │                  │
                  │  DiffUseCase     │
                  │  ReportingPort   │
                  │  SourceCtrlPort  │
                  │  RendererPort    │
                  └────────┬─────────┘
                           │
                           ▼
                  ┌──────────────────┐
                  │   App Layer      │
                  │   (service.go)   │
                  │                  │
                  │  Use case logic  │
                  │  Orchestration   │
                  └────────┬─────────┘
                           │
                           ▼
                  ┌──────────────────┐
                  │     Domain       │
                  │ (Business Types) │
                  │                  │
                  │  PRContext       │
                  │  DiffResult      │
                  │  EnvironmentConfig│
                  └──────────────────┘
```

**Key Points:**
- Domain is pure business types (no dependencies)
- App layer orchestrates use cases using domain types
- Ports define interfaces at the application boundary
- Adapters implement ports (dependency inversion)
- CMD wires everything together (composition root)

### Dependencies

**Allowed:**
- Domain → (nothing except stdlib)
- Ports → Domain
- App → Domain, Ports
- Adapters → Domain, Ports, Platform, External libs
- Cmd → Everything (composition root)

**Forbidden:**
- Domain ↛ Adapters ❌
- Domain ↛ Platform ❌
- Ports ↛ Adapters ❌
- App ↛ Adapters ❌
- Adapters ↛ Other Adapters ❌

### Enforcement

Architecture rules are enforced via `go-arch-lint` and will fail CI if violated.

## Installation & Setup

### Prerequisites

```bash
# Install golangci-lint
brew install golangci-lint

# Install go-arch-lint
go install github.com/fe3dback/go-arch-lint@latest
```

### Running Quality Checks

```bash
# Format code
make fmt

# Run all linters
make lint

# Generate coverage report
make coverage

# Full quality check
make lint && make test
```

## Future Improvements

### Diffing
- [ ] Add YAML normalization option (sort keys, normalize whitespace)
- [ ] Add resource sorting option (group by kind, sort alphabetically)
- [ ] Add semantic diff for Kubernetes resources (ignore order-independent fields)
- [ ] Add diff statistics (lines changed, resources added/removed/modified)
- [ ] Add diff filtering (ignore specific fields like timestamps, hashes)

### Architecture
- [ ] Add metrics/observability port for telemetry
- [ ] Add caching port for rendered manifests (avoid re-rendering on sync)
- [ ] Consider splitting adapters into separate packages per bounded context
- [ ] Add plugin system for custom diff formatters (JSON, YAML, HTML)
- [ ] Add event sourcing for audit trail of all diffs
- [ ] Add notification port for Slack/email alerts
