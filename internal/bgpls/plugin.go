package bgpls

import (
	"context"
	"log/slog"

	"github.com/neverthenetwork/inventa/internal/config"
	"github.com/neverthenetwork/inventa/internal/datastore"
	"github.com/neverthenetwork/inventa/internal/logging"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"
)

// Plugin implements discovery.Plugin for BGP-LS topology data.
type Plugin struct {
	cfg    *config.Conf
	logger *slog.Logger
	srv    *server.BgpServer
	cancel context.CancelFunc
}

// New creates a new BGP-LS plugin.
func New(cfg *config.Conf, logger *slog.Logger) *Plugin {
	return &Plugin{
		cfg:    cfg,
		logger: logger,
	}
}

// Name returns the source identifier.
func (p *Plugin) Name() string { return "bgpls" }

// Start begins BGP-LS discovery. It starts the BGP server, adds the peer,
// watches for link-state updates, and streams topology into the store.
// Blocks until the context is cancelled.
func (p *Plugin) Start(ctx context.Context, store *datastore.TopologyStore) error {
	ctx, p.cancel = context.WithCancel(ctx)
	defer p.cancel()

	bgpLogger := logging.NewSlogAdapter(p.logger)
	p.srv = server.NewBgpServer(server.LoggerOption(bgpLogger))
	go p.srv.Serve()

	if err := p.srv.StartBgp(ctx, &api.StartBgpRequest{
		Global: &api.Global{
			Asn:        uint32(p.cfg.LocalASN),
			RouterId:   p.cfg.LocalRouterID,
			ListenPort: -1, // don't listen on tcp:179
		},
	}); err != nil {
		return err
	}

	p.logger.Info("starting BGP-LS peer",
		"peer", p.cfg.PeerIPv4Address,
		"asn", p.cfg.PeerASN,
	)

	if err := p.srv.AddPeer(ctx, &api.AddPeerRequest{
		Peer: MakePeerConfiguration(p.cfg.PeerIPv4Address, p.cfg.PeerASN),
	}); err != nil {
		return err
	}

	count := 0
	if err := p.srv.WatchEvent(ctx, &api.WatchEventRequest{
		Peer: &api.WatchEventRequest_Peer{},
		Table: &api.WatchEventRequest_Table{
			Filters: []*api.WatchEventRequest_Table_Filter{
				{
					Type: api.WatchEventRequest_Table_Filter_BEST,
				},
			},
		}}, func(r *api.WatchEventResponse) {
		ProcessBGPUpdates(r, count, p.srv, store, p.cfg, p.logger)
		count++
	}); err != nil {
		return err
	}

	p.logger.Info("BGP-LS discovery stopped")
	return nil
}

// Stop gracefully shuts down the BGP server and cancels the watch.
func (p *Plugin) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}
