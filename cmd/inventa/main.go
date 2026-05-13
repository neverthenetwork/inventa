package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
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
)

//go:embed static/*
var staticFiles embed.FS

func loadJSON(fileName string) error {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("reading JSON file: %w", err)
	}
	if err := json.Unmarshal(content, &datastore.Elements); err != nil {
		return fmt.Errorf("unmarshaling JSON: %w", err)
	}
	return nil
}

func main() {
	var configPath string
	var runMode string

	// Logging first
	logging.SetUpLogger()

	// Flags
	flag.StringVar(&configPath, "c", "/etc/inventa.yaml", "specify location of config file, default is /etc/inventa.yaml")
	flag.StringVar(&runMode, "r", "bgp", "specify run mode, use 'local' to load from local JSON file")
	flag.Parse()

	config.InitConfig(configPath)

	// Set embedded static files for the web package
	web.StaticFS = staticFiles

	s := server.NewBgpServer(server.LoggerOption(&logging.MyLogger{Logger: logging.Log}))

	if runMode != "local" {
		logging.Log.Info("Starting BGP")
		go s.Serve()

		if err := s.StartBgp(context.Background(), &api.StartBgpRequest{
			Global: &api.Global{
				Asn:        uint32(config.Configs.LocalASN),
				RouterId:   config.Configs.LocalRouterID,
				ListenPort: -1, // gobgp won't listen on tcp:179
			},
		}); err != nil {
			logging.Log.Fatal(err)
		}
	} else {
		if err := loadJSON(config.Configs.LocalJSONFile); err != nil {
			logging.Log.Fatal(err)
		}
		logging.Log.Info(fmt.Sprintf("Read static file: %d Nodes loaded", len(datastore.Elements.Nodes)))
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
			bgpls.ProcessBGPUpdates(r, count, s)
			count++
		}); err != nil {
			logging.Log.Fatal(err)
		}

		if err := s.AddPeer(context.Background(), &api.AddPeerRequest{
			Peer: bgpls.MakePeerConfiguration(config.Configs.PeerIPv4Address, config.Configs.PeerASN),
		}); err != nil {
			logging.Log.Fatal(err)
		}
	}

	// Serve static files (CSS, JS) from the embedded filesystem.
	// Strip the "static/" prefix so URLs like /resources/style.css resolve.
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("failed to create static sub-filesystem: %v", err)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	http.Handle("/resources/", http.StripPrefix("/resources", fileServer))
	http.HandleFunc("/", web.IndexHandler)
	http.HandleFunc("/vr", web.VRIndexHandler)
	http.HandleFunc("/3d", web.ThreeDIndexHandler)
	http.HandleFunc("/elementdata.json", web.JsHandler)

	logging.Log.Info(fmt.Sprintf("Starting web server on port %d", config.Configs.HTTPListenPort))

	// Set up signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", config.Configs.HTTPListenPort)
		if config.Configs.HTTPSEnable {
			serverErr <- http.ListenAndServeTLS(addr, config.Configs.HTTPSCertFile, config.Configs.HTTPSKeyFile, nil)
		} else {
			serverErr <- http.ListenAndServe(addr, nil)
		}
	}()

	// Wait for signal or server error
	select {
	case <-ctx.Done():
		logging.Log.Info("Shutting down...")
	case err := <-serverErr:
		if err != nil {
			logging.Log.Fatal(err)
		}
	}
}
