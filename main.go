package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/markussiebert/homeddns/cmd"
	"github.com/markussiebert/homeddns/internal/logger"
	"github.com/markussiebert/homeddns/internal/provider"
)

// CLI holds the command-line interface structure.
type CLI struct {
	Server struct {
		Port int `help:"Port to listen on." default:"8080"`
	} `cmd:"" help:"Run as a web server."`

	Update struct {
		Hostname string `arg:"" help:"Hostname to update (e.g., sub.domain.com)."`
		Type     string `help:"Record type (A or AAAA)." default:"A" enum:"A,AAAA"`
	} `cmd:"" help:"Update a DNS record with the current public IP."`

	HashPassword struct{} `cmd:"" help:"Generate bcrypt hash from stdin password."`

	Version struct{} `cmd:"" help:"Print the current version."`

	ListProviders bool `help:"List available DNS providers."`
}

var (
	buildVersion = "dev"
)

const fallbackVersion = "0.0.0-dev"

func versionString() string {
	if trimmed := strings.TrimSpace(buildVersion); trimmed != "" {
		return trimmed
	}
	return fallbackVersion
}

func ensureServerDefaultForContainer() {
	if len(os.Args) > 1 {
		return
	}
	if isRunningInContainer() {
		os.Args = append(os.Args, "server")
	}
}

func isRunningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	text := string(data)
	return strings.Contains(text, "docker") || strings.Contains(text, "kubepods") || strings.Contains(text, "containerd") || strings.Contains(text, "podman")
}

func main() {
	// Initialize logger at INFO level by default
	// Will be re-initialized after loading Home Assistant config
	logger.SetLevel(logger.LevelInfo)

	ensureServerDefaultForContainer()

	var cli CLI
	ctx := kong.Parse(&cli)

	if cli.ListProviders {
		fmt.Println("Available providers:")
		for _, name := range provider.List() {
			fmt.Println("-", name)
		}
		return
	}

	if ctx.Command() == "version" {
		fmt.Println(versionString())
		return
	}

	if ctx.Command() == "hash-password" {
		err := cmd.RunHashPassword()
		ctx.FatalIfErrorf(err)
		return
	}

	config, err := cmd.LoadConfig()
	if err != nil {
		ctx.FatalIfErrorf(fmt.Errorf("failed to load configuration: %w", err))
	}

	// Re-initialize logger with LOG_LEVEL from environment (set by Home Assistant config)
	logger.SetLevelFromString(os.Getenv("LOG_LEVEL"))
	logger.Debug("Logger re-initialized with level: %s", logger.GetLevel())
	logger.Debug("Configuration loaded: provider=%s, ttl=%d", config.Provider, config.DefaultTTL)

	switch ctx.Command() {
	case "server":
		err = cmd.RunServer(cli.Server.Port, config)
	case "update <hostname>":
		err = cmd.RunUpdate(cli.Update.Hostname, cli.Update.Type, config)
	default:
		err = fmt.Errorf("unknown command: %s", ctx.Command())
	}

	ctx.FatalIfErrorf(err)
}
