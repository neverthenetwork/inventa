// Package aws provides a discovery plugin that maps AWS network topology
// (VPCs, subnets, transit gateways, peering, VPN, etc.) into cytoscape.js
// elements for visualization in Inventa.
package aws

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/neverthenetwork/inventa/internal/config"
	"github.com/neverthenetwork/inventa/internal/datastore"

	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// Plugin implements discovery.Plugin for AWS topology discovery.
type Plugin struct {
	cfg    *config.Conf
	logger *slog.Logger
	sdkCfg aws.Config
	cancel context.CancelFunc
}

// New creates a new AWS discovery plugin.
func New(cfg *config.Conf, logger *slog.Logger) (*Plugin, error) {
	awscfg := cfg.Sources.AWS
	if len(awscfg.Regions) == 0 {
		return nil, fmt.Errorf("aws plugin: at least one region must be configured")
	}

	// AWS SDK config with optional profile
	var loadOpts []func(*awsconfig.LoadOptions) error
	if awscfg.Profile != "" {
		loadOpts = append(loadOpts, awsconfig.WithSharedConfigProfile(awscfg.Profile))
	}

	// Custom endpoint URL (e.g. Floci, LocalStack)
	if awscfg.EndpointURL != "" {
		//nolint:staticcheck // EndpointResolverWithOptionsFunc is deprecated but still the cleanest way
		// to support multiple AWS services with a single custom endpoint URL.
		resolver := aws.EndpointResolverWithOptionsFunc(
			func(_, region string, _ ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					PartitionID:   "aws",
					URL:           awscfg.EndpointURL,
					SigningRegion: region,
				}, nil
			},
		)
		//nolint:staticcheck // WithEndpointResolverWithOptions is coupled to the resolver above.
		loadOpts = append(loadOpts, awsconfig.WithEndpointResolverWithOptions(resolver))
		// Use anonymous credentials for local emulators
		loadOpts = append(loadOpts, func(o *awsconfig.LoadOptions) error {
			o.Credentials = aws.AnonymousCredentials{}
			return nil
		})
		logger.Info("aws plugin: using custom endpoint", "url", awscfg.EndpointURL)
	}

	sdkCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	// Assume role if configured
	if awscfg.RoleARN != "" {
		// Role assumption would go here — omitted for initial implementation
		logger.Warn("aws plugin: role_arn configured but role assumption not yet implemented")
	}

	return &Plugin{
		cfg:    cfg,
		logger: logger,
		sdkCfg: sdkCfg,
	}, nil
}

// Name returns the source identifier.
func (p *Plugin) Name() string { return "aws" }

// Start begins the AWS topology polling loop. Blocks until context is cancelled.
func (p *Plugin) Start(ctx context.Context, store *datastore.TopologyStore) error {
	ctx, p.cancel = context.WithCancel(ctx)
	defer p.cancel()

	awscfg := p.cfg.Sources.AWS
	interval := time.Duration(awscfg.PollInterval) * time.Second
	if interval <= 0 {
		interval = 300 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately, then on tick
	for {
		if err := p.discoverAndStore(ctx, store); err != nil {
			p.logger.Error("AWS discovery failed", "error", err)
			// Continue polling — transient errors shouldn't kill the plugin
		}

		select {
		case <-ctx.Done():
			p.logger.Info("AWS discovery stopped")
			return nil
		case <-ticker.C:
		}
	}
}

// Stop gracefully cancels the polling loop.
func (p *Plugin) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

// discoverAndStore runs a full discovery cycle and updates the store.
func (p *Plugin) discoverAndStore(ctx context.Context, store *datastore.TopologyStore) error {
	elements, err := p.discoverTopology(ctx)
	if err != nil {
		return err
	}
	store.Set(elements)
	p.logger.Info("AWS topology updated",
		"nodes", len(elements.Nodes),
		"edges", len(elements.Edges),
	)
	return nil
}

// discoverTopology runs AWS discovery across all configured regions.
func (p *Plugin) discoverTopology(ctx context.Context) (cy.Elements, error) {
	regions := p.cfg.Sources.AWS.Regions

	nodes := make([]cy.Node, 0)
	edges := make([]cy.Edge, 0)

	for _, region := range regions {
		regionCfg := p.sdkCfg.Copy()
		regionCfg.Region = region

		elementsCh := make(chan cy.Elements, 1)
		if err := Discover(ctx, regionCfg, p.logger, func(e cy.Elements) {
			elementsCh <- e
		}); err != nil {
			p.logger.Error("failed to discover region", "region", region, "error", err)
			continue
		}
		elements := <-elementsCh
		nodes = append(nodes, elements.Nodes...)
		edges = append(edges, elements.Edges...)
	}

	return cy.Elements{Nodes: nodes, Edges: edges}, nil
}

// getTagValue returns the value of a tag with the given key, or a default.
func getTagValue(tags []ec2types.Tag, key, defaultVal string) string {
	for _, t := range tags {
		if aws.ToString(t.Key) == key {
			return aws.ToString(t.Value)
		}
	}
	return defaultVal
}

// getCIDRString returns a comma-separated list of CIDR blocks from the association set.
func getCIDRString(associations []ec2types.VpcCidrBlockAssociation) string {
	cidrs := make([]string, 0, len(associations))
	for _, assoc := range associations {
		if assoc.CidrBlockState.State == ec2types.VpcCidrBlockStateCodeAssociated {
			cidrs = append(cidrs, aws.ToString(assoc.CidrBlock))
		}
	}
	if len(cidrs) == 0 {
		return ""
	}
	result := cidrs[0]
	for _, c := range cidrs[1:] {
		result += ", " + c
	}
	return result
}
