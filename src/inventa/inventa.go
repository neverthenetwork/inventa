package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/neverthenetwork/inventa/src/inventa/datastore"
	"github.com/neverthenetwork/inventa/src/inventa/input/bgpls"
	"github.com/neverthenetwork/inventa/src/inventa/logging"
	"github.com/neverthenetwork/inventa/src/inventa/utils"
	"github.com/neverthenetwork/inventa/src/inventa/web"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"
)

func loadJSON(fileName string) error {
	content, _ := os.ReadFile(fileName)
	if err := json.Unmarshal(content, &datastore.Elements); err != nil {
		return err
	}
	return nil
}

func main() {

	var configPath string
	var runMode string

	// Logging first
	logging.SetUpLogger()

	// flags declaration using flag package
	flag.StringVar(&configPath, "c", "/etc/inventa.yaml", "Specify location of config file, default is /etc/inventa.yaml")
	flag.StringVar(&runMode, "r", "bgp", "Specify run mode, use 'local' to load from local file")

	flag.Parse()

	utils.InitConfig(configPath)

	s := server.NewBgpServer(server.LoggerOption(&logging.MyLogger{Logger: logging.Log}))

	if runMode != "local" {
		logging.Log.Info("Starting BGP")
		go s.Serve()

		if err := s.StartBgp(context.Background(), &api.StartBgpRequest{
			Global: &api.Global{
				Asn:        uint32(utils.Configs.LocalASN),
				RouterId:   utils.Configs.LocalRouterID,
				ListenPort: -1, // gobgp won't listen on tcp:179
			},
		}); err != nil {
			logging.Log.Fatal(err)
		}
	} else {
		if err := loadJSON(utils.Configs.LocalJSONFile); err != nil {
			logging.Log.Fatal(err)
		} else {
			logging.Log.Info(fmt.Sprintf("Read static file: %d Nodes loaded\n", len(datastore.Elements.Nodes)))
		}
	}
	count := 0
	if runMode != "local" {
		// the change of the peer state and path
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
			Peer: bgpls.MakePeerConfiguration(utils.Configs.PeerIPv4Address, utils.Configs.PeerASN),
		}); err != nil {
			logging.Log.Fatal(err)
		}
	}

	fileServer := http.FileServer(http.Dir("../../static"))
	http.Handle("/resources/", http.StripPrefix("/resources", fileServer))
	http.HandleFunc("/", web.IndexHandler)
	http.HandleFunc("/vr", web.VRIndexHandler)
	http.HandleFunc("/3d", web.ThreeDIndexHandler)
	http.HandleFunc("/elementdata.json", web.JsHandler)
	logging.Log.Info(fmt.Sprintf("Starting web server on port %d", utils.Configs.HTTPListenPort))
	if utils.Configs.HTTPSEnable {
		if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", utils.Configs.HTTPListenPort), "../../cert/certificate.pem", "../../cert/key.pem", nil); err != nil {
			logging.Log.Fatal(err)
		}
	} else {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", utils.Configs.HTTPListenPort), nil); err != nil {
			logging.Log.Fatal(err)
		}
	}

	select {}
}
