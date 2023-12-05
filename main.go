package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/dns"
	"github.com/Tarow/dockdns/internal/provider"
	"github.com/docker/docker/client"
	"github.com/ilyakaznacheev/cleanenv"
)

var configPath string

func main() {
	flag.StringVar(&configPath, "config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	var appCfg config.AppConfig
	err := cleanenv.ReadConfig(configPath, &appCfg)
	if err != nil {
		slog.Error("Failed to read "+configPath, "error", err)
		os.Exit(1)
	}
	slog.Debug("Successfully read config", "config", appCfg)

	dnsProvider, err := provider.Get(appCfg.Provider)
	if err != nil {
		slog.Error("Failed to create DNS provider", "error", err)
		os.Exit(1)
	}

	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		slog.Error("Could not create docker client", "error", err)
	}

	handler := dns.NewHandler(dnsProvider, appCfg.DNS, appCfg.Domains, dockerCli)
	err = handler.Run()
	if err != nil {
		slog.Debug("DNS Handler exited with error", "error", err)
		os.Exit(1)
	}
}
