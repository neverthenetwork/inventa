package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/neverthenetwork/inventa/internal/bgpls"
	"github.com/neverthenetwork/inventa/internal/config"
	"github.com/neverthenetwork/inventa/internal/datastore"
	"github.com/neverthenetwork/inventa/internal/logging"
	"github.com/neverthenetwork/inventa/internal/web"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"
	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

//go:embed static/*
var staticFiles embed.FS

func loadJSON(fileName string, store *datastore.TopologyStore) error {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("reading JSON file: %w", err)
	}
	var elements cy.Elements
	if err := json.Unmarshal(content, &elements); err != nil {
		return fmt.Errorf("unmarshaling JSON: %w", err)
	}
	store.Set(elements)
	return nil
}

func main() {
	var configPath string
	var runMode string

	// Logging first
	logger := logging.NewLogger()

	// Flags
	flag.StringVar(&configPath, "c", "/etc/inventa.yaml", "specify location of config file, default is /etc/inventa.yaml")
	flag.StringVar(&runMode, "r", "bgp", "specify run mode, use 'local' to load from local JSON file")
	flag.Parse()

	// Load config
	cfg, err := config.LoadConfig(configPath, logger)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Create shared dependencies
	store := datastore.NewTopologyStore()

	// Set up web server with dependencies
	webSrv := &web.Server{
		StaticFS: staticFiles,
		Store:    store,
		Cfg:      cfg,
		Logger:   logger,
	}

	// BGP server adapter
	bgpLogger := logging.NewSlogAdapter(logger)
	s := server.NewBgpServer(server.LoggerOption(bgpLogger))

	if runMode != "local" {
		logger.Info("starting BGP")
		go s.Serve()

		if err := s.StartBgp(context.Background(), &api.StartBgpRequest{
			Global: &api.Global{
				Asn:        uint32(cfg.LocalASN),
				RouterId:   cfg.LocalRouterID,
				ListenPort: -1, // gobgp won't listen on tcp:179
			},
		}); err != nil {
			logger.Error("failed to start BGP", "error", err)
			os.Exit(1)
		}
	} else {
		if err := loadJSON(cfg.LocalJSONFile, store); err != nil {
			logger.Error("failed to load JSON", "error", err)
			os.Exit(1)
		}
		logger.Info("loaded static topology", "nodes", store.NodeCount())
	}

	count := 0
	if runMode != "local" {
		if err := s.WatchEvent(context.Background(), &api.WatchEventRequest{
			Peer: &api.WatchEventRequest_Peer{},
			Table: &api.WatchEventRequest_Table{
				Filters: []*api.WatchEventRequest_Table_Filter{
					{
						Type: api.WatchEventRequest_Table_Filter_BEST,
					},
				},
			}}, func(r *api.WatchEventResponse) {
			bgpls.ProcessBGPUpdates(r, count, s, store, cfg, logger)
			count++
		}); err != nil {
			logger.Error("failed to watch BGP events", "error", err)
			os.Exit(1)
		}

		if err := s.AddPeer(context.Background(), &api.AddPeerRequest{
			Peer: bgpls.MakePeerConfiguration(cfg.PeerIPv4Address, cfg.PeerASN),
		}); err != nil {
			logger.Error("failed to add BGP peer", "error", err)
			os.Exit(1)
		}
	}

	// Serve static files from embedded filesystem
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logger.Error("failed to create static sub-filesystem", "error", err)
		os.Exit(1)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	http.Handle("/resources/", http.StripPrefix("/resources", fileServer))
	http.HandleFunc("/", webSrv.IndexHandler)
	http.HandleFunc("/vr", webSrv.VRIndexHandler)
	http.HandleFunc("/3d", webSrv.ThreeDIndexHandler)
	http.HandleFunc("/elementdata.json", webSrv.JsHandler)

	logger.Info("starting web server", "port", cfg.HTTPListenPort)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
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
	case <-ctx.Done():
		logger.Info("shutting down")
	case err := <-serverErr:
		if err != nil {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}
}
