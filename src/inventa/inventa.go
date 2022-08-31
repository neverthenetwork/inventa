package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/shelson/inventa/src/inventa/datastore"
	"github.com/shelson/inventa/src/inventa/input/bgpls"
	"github.com/shelson/inventa/src/inventa/utils"
	"github.com/shelson/inventa/src/inventa/web"

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

	utils.InitConfig()

	utils.SetUpLogger()

	s := server.NewBgpServer(server.LoggerOption(&utils.MyLogger{Logger: utils.Log}))

	if utils.Configs.RunTimeMode != "local" {
		utils.Log.Info("Starting BGP")
		go s.Serve()

		if err := s.StartBgp(context.Background(), &api.StartBgpRequest{
			Global: &api.Global{
				Asn:        uint32(utils.Configs.LocalASN),
				RouterId:   utils.Configs.LocalRouterID,
				ListenPort: -1, // gobgp won't listen on tcp:179
			},
		}); err != nil {
			utils.Log.Fatal(err)
		}
	} else {
		if err := loadJSON(utils.Configs.LocalJSONFile); err != nil {
			utils.Log.Fatal(err)
		} else {
			utils.Log.Info(fmt.Sprintf("Read static file: %d Nodes loaded\n", len(datastore.Elements.Nodes)))
		}
	}
	count := 0
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
		utils.Log.Fatal(err)
	}

	if err := s.AddPeer(context.Background(), &api.AddPeerRequest{
		Peer: bgpls.MakePeerConfiguration(utils.Configs.PeerIPv4Address, utils.Configs.PeerASN),
	}); err != nil {
		utils.Log.Fatal(err)
	}

	fileServer := http.FileServer(http.Dir("../../static"))
	http.Handle("/resources/", http.StripPrefix("/resources", fileServer))
	http.HandleFunc("/", web.IndexHandler)
	http.HandleFunc("/elementdata.json", web.JsHandler)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", utils.Configs.HTTPListenPort), nil); err != nil {
		utils.Log.Fatal(err)
	}

	select {}
}
