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
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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

// discoverTopology fetches AWS network resources and converts to cytoscape elements.
func (p *Plugin) discoverTopology(ctx context.Context) (cy.Elements, error) {
	regions := p.cfg.Sources.AWS.Regions

	nodes := make([]cy.Node, 0)
	edges := make([]cy.Edge, 0)

	for _, region := range regions {
		client := ec2.NewFromConfig(p.sdkCfg, func(o *ec2.Options) {
			o.Region = region
		})

		regionNodes, regionEdges, err := p.discoverRegion(ctx, client, region)
		if err != nil {
			p.logger.Error("failed to discover region", "region", region, "error", err)
			continue
		}
		nodes = append(nodes, regionNodes...)
		edges = append(edges, regionEdges...)
	}

	return cy.Elements{Nodes: nodes, Edges: edges}, nil
}

// discoverRegion discovers all network resources in a single AWS region.
func (p *Plugin) discoverRegion(ctx context.Context, client *ec2.Client, region string) ([]cy.Node, []cy.Edge, error) {
	nodes := make([]cy.Node, 0)
	edges := make([]cy.Edge, 0)

	// VPCs
	vpcs, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, nil, fmt.Errorf("describing VPCs in %s: %w", region, err)
	}

	vpcIDs := make(map[string]bool)
	for _, vpc := range vpcs.Vpcs {
		id := aws.ToString(vpc.VpcId)
		name := getTagValue(vpc.Tags, "Name", id)
		cidr := getCIDRString(vpc.CidrBlockAssociationSet)

		vpcIDs[id] = true
		nodes = append(nodes, cy.Node{
			Data: cy.NodeData{
				ID: id,
				Attributes: map[string]interface{}{
					"label":   name,
					"type":    "vpc",
					"cidr":    cidr,
					"region":  region,
					"group":   region,
					"cluster": len(nodes),
				},
			},
			Selectable: true,
		})
	}

	// Subnets
	subnets, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{})
	if err != nil {
		return nil, nil, fmt.Errorf("describing subnets in %s: %w", region, err)
	}
	for _, sn := range subnets.Subnets {
		id := aws.ToString(sn.SubnetId)
		name := getTagValue(sn.Tags, "Name", id)
		vpcID := aws.ToString(sn.VpcId)

		nodes = append(nodes, cy.Node{
			Data: cy.NodeData{
				ID: id,
				Attributes: map[string]interface{}{
					"label":  name,
					"type":   "subnet",
					"cidr":   aws.ToString(sn.CidrBlock),
					"az":     aws.ToString(sn.AvailabilityZone),
					"group":  vpcID,
					"vpc_id": vpcID,
				},
			},
			Selectable: true,
		})

		// Subnet → VPC edge
		edges = append(edges, cy.Edge{
			Data: cy.EdgeData{
				ID:     fmt.Sprintf("subnet-%s-%s", id, vpcID),
				Source: id,
				Target: vpcID,
				Attributes: map[string]interface{}{
					"type": "subnet_of",
				},
			},
			Selectable: true,
		})
	}

	// Transit Gateways
	tgws, err := client.DescribeTransitGateways(ctx, &ec2.DescribeTransitGatewaysInput{})
	if err != nil {
		return nil, nil, fmt.Errorf("describing transit gateways in %s: %w", region, err)
	}
	tgwIDs := make(map[string]bool)
	for _, tgw := range tgws.TransitGateways {
		id := aws.ToString(tgw.TransitGatewayId)
		name := getTagValue(tgw.Tags, "Name", id)

		tgwIDs[id] = true
		nodes = append(nodes, cy.Node{
			Data: cy.NodeData{
				ID: id,
				Attributes: map[string]interface{}{
					"label":  name,
					"type":   "transit_gateway",
					"region": region,
				},
			},
			Selectable: true,
		})
	}

	// Transit Gateway Attachments
	attachments, err := client.DescribeTransitGatewayAttachments(ctx, &ec2.DescribeTransitGatewayAttachmentsInput{})
	if err != nil {
		return nil, nil, fmt.Errorf("describing TGW attachments in %s: %w", region, err)
	}
	for _, att := range attachments.TransitGatewayAttachments {
		if att.State != ec2types.TransitGatewayAttachmentStateAvailable {
			continue
		}
		attID := aws.ToString(att.TransitGatewayAttachmentId)
		tgwID := aws.ToString(att.TransitGatewayId)
		resID := aws.ToString(att.ResourceId)
		resType := string(att.ResourceType)

		edges = append(edges, cy.Edge{
			Data: cy.EdgeData{
				ID:     attID,
				Source: resID,
				Target: tgwID,
				Attributes: map[string]interface{}{
					"type":          "tgw_attachment",
					"resource_type": resType,
				},
			},
			Selectable: true,
		})
	}

	// VPC Peering
	peerings, err := client.DescribeVpcPeeringConnections(ctx, &ec2.DescribeVpcPeeringConnectionsInput{})
	if err != nil {
		return nil, nil, fmt.Errorf("describing VPC peering in %s: %w", region, err)
	}
	for _, pcx := range peerings.VpcPeeringConnections {
		if pcx.Status.Code != ec2types.VpcPeeringConnectionStateReasonCodeActive {
			continue
		}
		id := aws.ToString(pcx.VpcPeeringConnectionId)
		accepter := aws.ToString(pcx.AccepterVpcInfo.VpcId)
		requester := aws.ToString(pcx.RequesterVpcInfo.VpcId)

		edges = append(edges, cy.Edge{
			Data: cy.EdgeData{
				ID:     id,
				Source: requester,
				Target: accepter,
				Attributes: map[string]interface{}{
					"type": "vpc_peering",
				},
			},
			Selectable: true,
		})
	}

	// Internet Gateways
	igws, err := client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{})
	if err != nil {
		return nil, nil, fmt.Errorf("describing internet gateways in %s: %w", region, err)
	}
	for _, igw := range igws.InternetGateways {
		id := aws.ToString(igw.InternetGatewayId)
		name := getTagValue(igw.Tags, "Name", id)

		nodes = append(nodes, cy.Node{
			Data: cy.NodeData{
				ID: id,
				Attributes: map[string]interface{}{
					"label": name,
					"type":  "internet_gateway",
				},
			},
			Selectable: true,
		})

		for _, att := range igw.Attachments {
			vpcID := aws.ToString(att.VpcId)
			if vpcIDs[vpcID] {
				edges = append(edges, cy.Edge{
					Data: cy.EdgeData{
						ID:     fmt.Sprintf("igw-%s-%s", id, vpcID),
						Source: id,
						Target: vpcID,
						Attributes: map[string]interface{}{
							"type": "igw_attachment",
						},
					},
					Selectable: true,
				})
			}
		}
	}

	// NAT Gateways
	natGateways, err := client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
		Filter: []ec2types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("describing NAT gateways in %s: %w", region, err)
	}
	for _, nat := range natGateways.NatGateways {
		id := aws.ToString(nat.NatGatewayId)
		name := getTagValue(nat.Tags, "Name", id)
		subnetID := aws.ToString(nat.SubnetId)
		vpcID := aws.ToString(nat.VpcId)

		nodes = append(nodes, cy.Node{
			Data: cy.NodeData{
				ID: id,
				Attributes: map[string]interface{}{
					"label":     name,
					"type":      "nat_gateway",
					"public_ip": aws.ToString(nat.NatGatewayAddresses[0].PublicIp),
				},
			},
			Selectable: true,
		})

		// NAT GW → Subnet
		edges = append(edges, cy.Edge{
			Data: cy.EdgeData{
				ID:     fmt.Sprintf("nat-%s-%s", id, subnetID),
				Source: id,
				Target: subnetID,
				Attributes: map[string]interface{}{
					"type": "nat_in_subnet",
				},
			},
			Selectable: true,
		})

		// NAT GW → VPC (parent)
		if vpcIDs[vpcID] {
			edges = append(edges, cy.Edge{
				Data: cy.EdgeData{
					ID:     fmt.Sprintf("nat-%s-%s", id, vpcID),
					Source: id,
					Target: vpcID,
					Attributes: map[string]interface{}{
						"type": "nat_in_vpc",
					},
				},
				Selectable: true,
			})
		}
	}

	// VPN Gateways
	vpnGateways, err := client.DescribeVpnGateways(ctx, &ec2.DescribeVpnGatewaysInput{})
	if err != nil {
		return nil, nil, fmt.Errorf("describing VPN gateways in %s: %w", region, err)
	}
	for _, vgw := range vpnGateways.VpnGateways {
		if vgw.State != ec2types.VpnStateAvailable {
			continue
		}
		id := aws.ToString(vgw.VpnGatewayId)
		name := getTagValue(vgw.Tags, "Name", id)

		nodes = append(nodes, cy.Node{
			Data: cy.NodeData{
				ID: id,
				Attributes: map[string]interface{}{
					"label": name,
					"type":  "vpn_gateway",
					"asn":   aws.ToInt64(vgw.AmazonSideAsn),
				},
			},
			Selectable: true,
		})

		for _, att := range vgw.VpcAttachments {
			vpcID := aws.ToString(att.VpcId)
			if vpcIDs[vpcID] && att.State == ec2types.AttachmentStatusAttached {
				edges = append(edges, cy.Edge{
					Data: cy.EdgeData{
						ID:     fmt.Sprintf("vgw-%s-%s", id, vpcID),
						Source: id,
						Target: vpcID,
						Attributes: map[string]interface{}{
							"type": "vpn_gateway_attachment",
						},
					},
					Selectable: true,
				})
			}
		}
	}

	return nodes, edges, nil
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
