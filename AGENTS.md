# Agent Guidelines for homeddns

## Build & Test Commands (Use mise tasks)
- **CI Pipeline**: `mise run ci` - Run full CI (tidy, test, build, ko, validate)
- **Build Binary**: `mise run build`
- **Build Linux Binary**: `mise run build:linux` or `mise run build:linux-arm`
- **Build Container**: `mise run build:ko` (Ko builds without Dockerfile)
- **Build Multi-platform**: `mise run build:ko-multiplatform`
- **Run Server**: `mise run run`
- **Test All**: `mise run test`
- **Test Single Package**: `go test ./internal/provider`
- **Format Code**: `mise run fmt`
- **Clean Build Artifacts**: `mise run clean`
- **List All Tasks**: `mise tasks`

## Build Optimizations
All builds use aggressive optimization flags for minimal binary size:
- `-trimpath`: Remove file paths from binary
- `-ldflags='-s -w'`: Strip debug info and symbols
- `-extldflags=-static`: Fully static binary
- `-tags netgo`: Pure Go networking (no CGO)
- Result: ~14MB optimized binary

## Mise Best Practices
- **Prefer mise tasks over shell scripts**: All build, test, and release operations should be defined as mise tasks in `mise.toml`
- **Use mise task dependencies**: Leverage `depends = [...]` to ensure tasks run in correct order (e.g., `fmt` and `tidy` before `build`)
- **Keep GitHub workflows minimal**: Workflows should call ONE mise task - let mise dependencies handle the rest. Business logic belongs in `mise.toml`
- **Single task execution**: CI workflows should run `mise run ci` or `mise run release`, not multiple separate tasks
- **Never call mise from mise**: NEVER use `mise run` inside task scripts - use `depends = [...]` instead to create proper DAG
- **Centralize tool versions**: All tools (go, goreleaser, ko, cosign, svu, yq) are defined in `mise.toml` `[tools]` section
- **Environment consistency**: mise ensures same tool versions locally and in CI
- **Task naming conventions**: Use `task:subtask` format (e.g., `build:linux`, `release:validate`)
- **Orchestration via dependencies**: Use task dependencies in mise.toml instead of workflow steps to orchestrate complex builds

## Code Style & Conventions
- **Go Version**: 1.25.4 (managed via mise.toml)
- **Tools**: All tools (go, goreleaser, ko, cosign, svu, yq) managed via mise.toml
- **Environment Variables**: Use `.env.local` for local development (copy from `.env.local.example`)
- **Imports**: Group stdlib, external, and local packages with blank lines between groups
- **Error Handling**: Always wrap errors with context: `fmt.Errorf("action failed: %w", err)`
- **Naming**: Use camelCase for unexported, PascalCase for exported; use full words (avoid abbreviations)
- **Types**: Define config structs with clear field names; use pointers for optional fields
- **Contexts**: Pass `context.Context` as first parameter; set reasonable timeouts (e.g., 30s for API calls)
- **Logging**: Use `log.Printf()` for important events; include relevant context (hostname, IP, error details)
- **Provider Pattern**: Each provider handles its own credential loading (env vars → config file fallback)
- **Separation of Concerns**: `cmd/` for CLI/config, `internal/` for business logic, providers self-contained

## Architecture Notes
- **Provider Interface**: All DNS providers implement `Provider` interface with `Name()`, `GetRecord()`, `UpdateRecord()`, `Close()`
- **Credential Loading**: Providers load their own credentials (e.g., `LoadNetcupConfig()` checks env vars then `~/.homeddns/netcup_credentials`)
- **Factory Pattern**: Providers register via `RegisterFactory()` in `init()` functions
- **No Tests Present**: Add tests when modifying critical logic (especially providers and handlers)

## Release Workflow
The project uses a **Ko-first** strategy with clean separation of concerns:
- **Ko**: Handles container images (multi-arch, SBOM, signed) → GHCR
- **GoReleaser**: Handles binaries, archives, checksums, signatures → GitHub Releases

### Release Commands
- **Full Release**: `mise run release` - Complete release workflow (version, tag, push)
  - Set `RELEASE_BUMP=patch|minor|major|next` environment variable
- **Validate Config**: `mise run release:validate` - Check goreleaser config
- **Dry Run**: `mise run release:dry-run` - Test full release locally
- **Snapshot**: `mise run release:snapshot` - Build without publishing
- **GoReleaser**: `mise run release:goreleaser` - Release binaries
- **Ko Images**: `mise run release:ko` or `mise run release:ko-with-version`
- **Sign Images**: `mise run release:ko-sign` - Sign with cosign
- **Verify Images**: `mise run release:ko-verify` - Verify signatures
- **Check Release**: `mise run release:check` - Verify published artifacts

### Execution Order (in CI)
1. Run tests (`mise run test`)
2. Validate config (`mise run release:validate`)
3. Run GoReleaser (`mise run release:goreleaser`) → Binaries + GitHub Release
4. Run Ko (`mise run release:ko-with-version`) → Container images + SBOM
5. Sign images (`mise run release:ko-sign`) → Cosign signatures

### What Gets Released
- **Binaries**: Linux, macOS, Windows, FreeBSD (amd64, arm64, arm/v7)
- **Archives**: `.tar.gz` and `.zip` with README, LICENSE, example config
- **Checksums**: SHA256 checksums with cosign signatures
- **SBOMs**: SPDX format for binaries and containers
- **Container Images**: Multi-arch (linux/amd64, linux/arm64, linux/arm/v7) on GHCR
- **Tags**: `latest`, `{version}` for containers

## Reference Documentation
If you are unsure about build tools or configurations, check the `repomix/` folder:
- **goreleaser.xml**: Complete documentation for GoReleaser (multi-platform release builds)
- **ko.xml**: Complete documentation for Ko (containerless Kubernetes image builder)
