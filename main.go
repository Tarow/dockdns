package main

import (
	"context"
	"embed"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Tarow/dockdns/internal/api"
	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/dns"
	"github.com/Tarow/dockdns/internal/provider"
	"github.com/Tarow/dockdns/internal/schedule"
	"github.com/docker/docker/client"
	"github.com/ilyakaznacheev/cleanenv"
)

var configPath string
var dryRun bool

//go:embed static/*
var staticAssets embed.FS

func main() {
	flag.StringVar(&configPath, "config", "config.yaml", "Path to the configuration file")
	flag.BoolVar(&dryRun, "dry-run", false, "Enable dry run mode (no changes will be made)")
	flag.Parse()

	var appCfg config.AppConfig
	err := cleanenv.ReadConfig(configPath, &appCfg)
	if err != nil {
		slog.Error("Failed to read config", "path", configPath, "error", err)
		os.Exit(1)
	}
	slog.SetDefault(getLogger(appCfg.Log))
	slog.Debug("Successfully read config", "config", appCfg)

	if !(len(appCfg.Zones) > 0) {
		slog.Error("no zone configuration found, exiting")
		os.Exit(1)
	}

	if dryRun {
		slog.Info("Dry run enabled, changes won't be applied")
	}
	providers := map[string]dns.Provider{}
	for _, zone := range appCfg.Zones {
		dnsProvider, err := provider.Get(zone, dryRun)
		if err != nil {
			slog.Error("Failed to create DNS provider", "zone", zone.Name, "error", err)
			os.Exit(1)
		}
		providers[zone.Name] = dnsProvider
	}

	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		dockerCli = nil
		slog.Warn("Could not create docker client, ignoring dynamic configuration", "error", err)
	} else {
		dockerCli.NegotiateAPIVersion(context.Background())
	}

	dnsHandler := dns.NewHandler(providers, appCfg.DNS, appCfg.Domains, dockerCli)
	//run function
	run := func() {
		if err := dnsHandler.Run(); err != nil {
			slog.Error("DNS update failed with error", "error", err)
		}
	}

	// If interval is less than 0, we will only run once, otherwise we will run in continuous mode
	if appCfg.Interval < 0 {
		slog.Info("Negative interval specified, running DNS update just once")
		run()
		slog.Info("Finished DNS update")
		return
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	scheduler := schedule.NewScheduler(run)
	dockerEventTrigger := schedule.NewDockerEventTrigger(dockerCli)
	intervalTrigger := schedule.NewIntervalTrigger(time.Duration(appCfg.Interval) * time.Second)
	scheduler.Register(dockerEventTrigger)
	scheduler.Register(intervalTrigger)

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("Starting DNS updater")

		// Start scheduler
		scheduler.Start(ctx,
			time.Duration(appCfg.DebounceTime)*time.Second,
			time.Duration(appCfg.MaxDebounceTime)*time.Second,
			true)

		slog.Info("Received termination signal. Exiting DNS updater...")
	}()

	var server *http.Server
	if appCfg.WebUI {
		slog.Info("WebUI enabled, starting server ...")
		// Run the API server
		apiHandler := api.NewHandler(&dnsHandler)
		mux := http.NewServeMux()
		mux.Handle("/static/", http.FileServer(http.FS(staticAssets)))
		mux.HandleFunc("/", apiHandler.GetIndex)
		server = &http.Server{
			Addr:    ":8080",
			Handler: mux,
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				slog.Error("HTTP server error", "err", err)
			} else {
				slog.Info("Received shutdown signal, shutting down API server ...")
			}
		}()
	}

	// wait for kill signal
	<-signalCh
	slog.Info("Received shutdown signal, shutting down goroutines")

	// stop goroutines
	cancel()
	if appCfg.WebUI {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()
		server.Shutdown(shutdownCtx)
	}
	wg.Wait()
	slog.Info("Stopped all goroutines, bye")
}

func getLogger(cfg config.LogConfig) *slog.Logger {
	var logLevel = parseLogLevel(cfg.Level)
	var handler slog.Handler

	switch cfg.Format {
	case config.LogFormatJson:
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	case config.LogFormatSimple:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	default:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	}

	return slog.New(handler)
}

func parseLogLevel(logLevel string) slog.Level {
	switch strings.ToLower(logLevel) {
	case "debug":
		return slog.LevelDebug

	case "info":
		return slog.LevelInfo

	case "warn":
		return slog.LevelWarn

	case "error":
		return slog.LevelError

	default:
		return slog.LevelInfo
	}
}
