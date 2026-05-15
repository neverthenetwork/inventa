package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/neverthenetwork/inventa/internal/aws"
	"github.com/neverthenetwork/inventa/internal/bgpls"
	"github.com/neverthenetwork/inventa/internal/config"
	"github.com/neverthenetwork/inventa/internal/datastore"
	"github.com/neverthenetwork/inventa/internal/discovery"
	"github.com/neverthenetwork/inventa/internal/localjson"
	"github.com/neverthenetwork/inventa/internal/logging"
	"github.com/neverthenetwork/inventa/internal/web"
)

//go:embed all:web-dist
var webDist embed.FS

func main() {
	var configPath string

	// Logging first
	logger := logging.NewLogger()

	// Flags
	flag.StringVar(&configPath, "c", "/etc/inventa.yaml", "specify location of config file, default is /etc/inventa.yaml")
	flag.Parse()

	// Load config
	cfg, err := config.LoadConfig(configPath, logger)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Create shared dependencies
	store := datastore.NewTopologyStore()

	// Build plugin list based on config
	plugins := buildPlugins(cfg, logger)
	if len(plugins) == 0 {
		logger.Error("no topology sources enabled — enable at least one source in config")
		os.Exit(1)
	}

	// Start all plugins
	ctx, cancelAll := context.WithCancel(context.Background())
	defer cancelAll()

	for _, p := range plugins {
		go func(p discovery.Plugin) {
			logger.Info("starting source", "name", p.Name())
			if err := p.Start(ctx, store); err != nil {
				logger.Error("source failed", "name", p.Name(), "error", err)
				cancelAll() // stop all plugins if one fails
			}
		}(p)
	}

	// Set up web server with dependencies
	webSrv := &web.Server{
		StaticFS: webDist,
		Store:    store,
		Cfg:      cfg,
		Logger:   logger,
	}

	// Serve static files from embedded Vite build output at /
	// http.FileServer serves index.html for / and all assets with correct MIME types
	distFS, err := fs.Sub(webDist, "web-dist")
	if err != nil {
		logger.Error("failed to create dist sub-filesystem", "error", err)
		os.Exit(1)
	}
	fileServer := http.FileServer(http.FS(distFS))
	http.Handle("/", fileServer)
	http.HandleFunc("/vr", webSrv.VRIndexHandler)
	http.HandleFunc("/3d", webSrv.ThreeDIndexHandler)
	http.HandleFunc("/elementdata.json", webSrv.JsHandler)

	logger.Info("starting web server", "port", cfg.HTTPListenPort)

	// Graceful shutdown
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", cfg.HTTPListenPort)
		if cfg.HTTPSEnable {
			serverErr <- http.ListenAndServeTLS(addr, cfg.HTTPSCertFile, cfg.HTTPSKeyFile, nil)
		} else {
			serverErr <- http.ListenAndServe(addr, nil)
		}
	}()

	select {
	case <-sigCtx.Done():
		logger.Info("shutting down")
		cancelAll()
		for _, p := range plugins {
			if err := p.Stop(); err != nil {
				logger.Error("error stopping source", "name", p.Name(), "error", err)
			}
		}
	case err := <-serverErr:
		if err != nil {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}
}

// buildPlugins returns the list of enabled discovery plugins based on config.
// Falls back to legacy flat fields when no sources section is present.
func buildPlugins(cfg *config.Conf, logger *slog.Logger) []discovery.Plugin {
	var plugins []discovery.Plugin

	bgplsEnabled := cfg.Sources.BGPLS.Enabled || (cfg.Sources.BGPLS.PeerIPv4Address == "" &&
		cfg.PeerIPv4Address != "")
	jsonEnabled := cfg.Sources.LocalJSON.Enabled || (cfg.Sources.LocalJSON.File == "" &&
		cfg.LocalJSONFile != "")
	awsEnabled := cfg.Sources.AWS.Enabled

	if bgplsEnabled {
		plugins = append(plugins, bgpls.New(cfg, logger))
	}
	if jsonEnabled {
		plugins = append(plugins, localjson.New(cfg, logger))
	}
	if awsEnabled {
		awsPlugin, err := aws.New(cfg, logger)
		if err != nil {
			logger.Error("failed to create AWS plugin", "error", err)
		} else {
			plugins = append(plugins, awsPlugin)
		}
	}

	return plugins
}
