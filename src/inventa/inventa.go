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

	"github.com/sirupsen/logrus"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/log"
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

	utils.InitLogger()

	s := server.NewBgpServer(server.LoggerOption(&myLogger{logger: &utils.Log}))

	if utils.Configs.RunTimeMode != "local" {
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
			fmt.Printf("Read static file: %d Nodes loaded\n", len(datastore.Elements.Nodes))
		}
	}

	fileServer := http.FileServer(http.Dir("../../static"))
	http.Handle("/resources/", http.StripPrefix("/resources", fileServer))
	http.HandleFunc("/", web.IndexHandler)
	http.HandleFunc("/elementdata.json", web.JsHandler)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", utils.Configs.HTTPListenPort), nil); err != nil {
		utils.Log.Fatal(err)
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

	select {}
}

// implement github.com/osrg/gobgp/v3/pkg/log/Logger interface
type myLogger struct {
	logger *logrus.Logger
}

func (l *myLogger) Panic(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Panic(msg)
}

func (l *myLogger) Fatal(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Fatal(msg)
}

func (l *myLogger) Error(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Error(msg)
}

func (l *myLogger) Warn(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Warn(msg)
}

func (l *myLogger) Info(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Info(msg)
}

func (l *myLogger) Debug(msg string, fields log.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Debug(msg)
}

func (l *myLogger) SetLevel(level log.LogLevel) {
	l.logger.SetLevel(logrus.Level(level))
}

func (l *myLogger) GetLevel() log.LogLevel {
	return log.LogLevel(l.logger.GetLevel())
}
