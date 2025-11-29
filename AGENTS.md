# Agent Guidelines for homeddns

## Build & Test Commands (Use mise tasks)
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

## Code Style & Conventions
- **Go Version**: 1.25.4 (managed via mise.toml)
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

## Reference Documentation
If you are unsure about build tools or configurations, check the `repomix/` folder:
- **goreleaser.xml**: Complete documentation for GoReleaser (multi-platform release builds)
- **ko.xml**: Complete documentation for Ko (containerless Kubernetes image builder)
