// Package aws discovers AWS network topology and converts it to cytoscape graph elements.
// It queries EC2 (VPCs, subnets, instances, security groups, internet gateways)
// and ELBv2 (load balancers, target groups) via the AWS SDK.
package aws

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

// Setter is a function type for setting the discovered topology elements.
type Setter func(elements cy.Elements)

// Discover queries the AWS API and calls set with the discovered topology.
func Discover(ctx context.Context, cfg aws.Config, logger *slog.Logger, set Setter) error {
	ec2c := ec2.NewFromConfig(cfg)
	elbc := elasticloadbalancingv2.NewFromConfig(cfg)

	logger.Info("discovering AWS topology")

	// Fetch all resources
	vpcs, err := describeVPCs(ctx, ec2c)
	if err != nil {
		return fmt.Errorf("describing VPCs: %w", err)
	}

	subnets, err := describeSubnets(ctx, ec2c)
	if err != nil {
		return fmt.Errorf("describing subnets: %w", err)
	}

	instances, err := describeInstances(ctx, ec2c)
	if err != nil {
		return fmt.Errorf("describing instances: %w", err)
	}

	sgs, err := describeSecurityGroups(ctx, ec2c)
	if err != nil {
		return fmt.Errorf("describing security groups: %w", err)
	}

	igws, err := describeInternetGateways(ctx, ec2c)
	if err != nil {
		return fmt.Errorf("describing internet gateways: %w", err)
	}

	lbs, tgToInstances, lbToTGs, err := describeLoadBalancers(ctx, elbc)
	if err != nil {
		return fmt.Errorf("describing load balancers: %w", err)
	}

	elements := buildTopology(vpcs, subnets, instances, sgs, igws, lbs, tgToInstances, lbToTGs)
	set(elements)

	logger.Info("AWS topology discovered",
		"vpcs", len(vpcs),
		"subnets", len(subnets),
		"instances", len(instances),
		"security_groups", len(sgs),
		"internet_gateways", len(igws),
		"load_balancers", len(lbs),
		"nodes", len(elements.Nodes),
		"edges", len(elements.Edges),
	)

	return nil
}

// ----- EC2 API wrappers -----

func describeVPCs(ctx context.Context, c *ec2.Client) ([]ec2types.Vpc, error) {
	var out []ec2types.Vpc
	paginator := ec2.NewDescribeVpcsPaginator(c, &ec2.DescribeVpcsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, page.Vpcs...)
	}
	return out, nil
}

func describeSubnets(ctx context.Context, c *ec2.Client) ([]ec2types.Subnet, error) {
	var out []ec2types.Subnet
	paginator := ec2.NewDescribeSubnetsPaginator(c, &ec2.DescribeSubnetsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, page.Subnets...)
	}
	return out, nil
}

func describeInstances(ctx context.Context, c *ec2.Client) ([]ec2types.Instance, error) {
	var out []ec2types.Instance
	paginator := ec2.NewDescribeInstancesPaginator(c, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"pending", "running", "stopping", "stopped"},
			},
		},
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, r := range page.Reservations {
			out = append(out, r.Instances...)
		}
	}
	return out, nil
}

func describeSecurityGroups(ctx context.Context, c *ec2.Client) ([]ec2types.SecurityGroup, error) {
	var out []ec2types.SecurityGroup
	paginator := ec2.NewDescribeSecurityGroupsPaginator(c, &ec2.DescribeSecurityGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, page.SecurityGroups...)
	}
	return out, nil
}

func describeInternetGateways(ctx context.Context, c *ec2.Client) ([]ec2types.InternetGateway, error) {
	var out []ec2types.InternetGateway
	paginator := ec2.NewDescribeInternetGatewaysPaginator(c, &ec2.DescribeInternetGatewaysInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, page.InternetGateways...)
	}
	return out, nil
}

func describeLoadBalancers(ctx context.Context, c *elasticloadbalancingv2.Client) (
	lbs []elbtypes.LoadBalancer,
	tgToInstances map[string][]string,
	lbToTGs map[string][]string,
	err error,
) {
	tgToInstances = make(map[string][]string)
	lbToTGs = make(map[string][]string)

	// Describe load balancers
	lbPaginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(c, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	for lbPaginator.HasMorePages() {
		page, err := lbPaginator.NextPage(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		lbs = append(lbs, page.LoadBalancers...)
	}

	// For each LB, discover TGs via listeners (Floci doesn't populate LoadBalancerArns on TGs)
	seenTGs := make(map[string]bool)
	for _, lb := range lbs {
		lbARN := aws.ToString(lb.LoadBalancerArn)
		listeners, err := c.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{
			LoadBalancerArn: lb.LoadBalancerArn,
		})
		if err != nil {
			continue
		}
		for _, l := range listeners.Listeners {
			for _, action := range l.DefaultActions {
				tgARN := aws.ToString(action.TargetGroupArn)
				if tgARN == "" {
					continue
				}
				lbToTGs[lbARN] = append(lbToTGs[lbARN], tgARN)
				if seenTGs[tgARN] {
					continue
				}
				seenTGs[tgARN] = true
				// Fetch targets for this TG
				targets, err := c.DescribeTargetHealth(ctx, &elasticloadbalancingv2.DescribeTargetHealthInput{
					TargetGroupArn: aws.String(tgARN),
				})
				if err != nil {
					continue
				}
				for _, t := range targets.TargetHealthDescriptions {
					id := aws.ToString(t.Target.Id)
					tgToInstances[tgARN] = append(tgToInstances[tgARN], id)
				}
			}
		}
	}

	return lbs, tgToInstances, lbToTGs, nil
}

// ----- Topology builder -----

func buildTopology(
	vpcs []ec2types.Vpc,
	subnets []ec2types.Subnet,
	instances []ec2types.Instance,
	sgs []ec2types.SecurityGroup,
	igws []ec2types.InternetGateway,
	lbs []elbtypes.LoadBalancer,
	tgToInstances map[string][]string,
	lbToTGs map[string][]string,
) cy.Elements {
	elements := cy.Elements{
		Nodes: make([]cy.Node, 0),
		Edges: make([]cy.Edge, 0),
	}

	vpcIndex := make(map[string]int, len(vpcs))
	for i, v := range vpcs {
		vpcIndex[aws.ToString(v.VpcId)] = i
	}

	subnetVPC := make(map[string]string)
	instanceSubnet := make(map[string]string)
	instanceSGs := make(map[string][]string)
	subnetAZ := make(map[string]string)
	sgIDToName := make(map[string]string)

	for _, s := range subnets {
		id := aws.ToString(s.SubnetId)
		vpcID := aws.ToString(s.VpcId)
		subnetVPC[id] = vpcID
		subnetAZ[id] = aws.ToString(s.AvailabilityZone)
	}

	for _, inst := range instances {
		id := aws.ToString(inst.InstanceId)
		if inst.SubnetId != nil {
			instanceSubnet[id] = aws.ToString(inst.SubnetId)
		}
		for _, sg := range inst.SecurityGroups {
			instanceSGs[id] = append(instanceSGs[id], aws.ToString(sg.GroupId))
		}
	}

	for _, sg := range sgs {
		sgIDToName[aws.ToString(sg.GroupId)] = aws.ToString(sg.GroupName)
	}

	// VPC nodes
	for _, v := range vpcs {
		id := aws.ToString(v.VpcId)
		cidr := aws.ToString(v.CidrBlock)
		name := tagValue(v.Tags, "Name", id)
		clusterIdx := vpcIndex[id]
		elements.Nodes = append(elements.Nodes, cy.Node{
			Data: cy.NodeData{ID: id, Attributes: map[string]any{
				"label":     fmt.Sprintf("%s (%s)", name, cidr),
				"group":     "vpc",
				"cluster":   clusterIdx,
				"cidr":      cidr,
				"isDefault": aws.ToBool(v.IsDefault),
			}},
			Selectable: true,
		})
	}

	// Subnet nodes + edges
	for _, s := range subnets {
		id := aws.ToString(s.SubnetId)
		vpcID := aws.ToString(s.VpcId)
		cidr := aws.ToString(s.CidrBlock)
		az := aws.ToString(s.AvailabilityZone)
		name := tagValue(s.Tags, "Name", id)
		clusterIdx := vpcIndex[vpcID]
		elements.Nodes = append(elements.Nodes, cy.Node{
			Data: cy.NodeData{ID: id, Attributes: map[string]any{
				"label":   fmt.Sprintf("%s (%s, %s)", name, cidr, az),
				"group":   "subnet",
				"cluster": clusterIdx,
				"cidr":    cidr,
				"vpcId":   vpcID,
				"az":      az,
				"public":  aws.ToBool(s.MapPublicIpOnLaunch),
			}},
			Selectable: true,
		})
		elements.Edges = append(elements.Edges, cy.Edge{
			Data: cy.EdgeData{
				ID:         fmt.Sprintf("%s_parent_%s", id, vpcID),
				Source:     id,
				Target:     vpcID,
				Attributes: map[string]any{"type": "parent"},
			},
			Selectable: true,
		})
	}

	// Instance nodes + edges
	for _, inst := range instances {
		id := aws.ToString(inst.InstanceId)
		instType := string(inst.InstanceType)
		name := tagValue(inst.Tags, "Name", id)
		privIP := safeString(inst.PrivateIpAddress)
		pubIP := safeString(inst.PublicIpAddress)
		subnetID := instanceSubnet[id]
		clusterIdx := vpcIndex[subnetVPC[subnetID]]
		elements.Nodes = append(elements.Nodes, cy.Node{
			Data: cy.NodeData{ID: id, Attributes: map[string]any{
				"label":        fmt.Sprintf("%s (%s)", name, privIP),
				"group":        "instance",
				"cluster":      clusterIdx,
				"instanceType": instType,
				"privateIp":    privIP,
				"publicIp":     pubIP,
				"subnetId":     subnetID,
				"vpcId":        subnetVPC[subnetID],
				"state":        string(inst.State.Name),
			}},
			Selectable: true,
		})
		if subnetID != "" {
			elements.Edges = append(elements.Edges, cy.Edge{
				Data: cy.EdgeData{
					ID:         fmt.Sprintf("%s_parent_%s", id, subnetID),
					Source:     id,
					Target:     subnetID,
					Attributes: map[string]any{"type": "parent"},
				},
				Selectable: true,
			})
		}
		for _, sgID := range instanceSGs[id] {
			sgName := sgIDToName[sgID]
			if sgName == "" {
				sgName = sgID
			}
			elements.Edges = append(elements.Edges, cy.Edge{
				Data: cy.EdgeData{
					ID:     fmt.Sprintf("%s_member_%s", id, sgID),
					Source: id,
					Target: sgID,
					Attributes: map[string]any{
						"type":      "member",
						"groupName": sgName,
					},
				},
				Selectable: true,
			})
		}
	}

	// Security Group nodes + edges
	for _, sg := range sgs {
		id := aws.ToString(sg.GroupId)
		name := aws.ToString(sg.GroupName)
		vpcID := aws.ToString(sg.VpcId)
		clusterIdx := vpcIndex[vpcID]
		elements.Nodes = append(elements.Nodes, cy.Node{
			Data: cy.NodeData{ID: id, Attributes: map[string]any{
				"label":       fmt.Sprintf("SG: %s", name),
				"group":       "security_group",
				"cluster":     clusterIdx,
				"groupName":   name,
				"description": aws.ToString(sg.Description),
				"vpcId":       vpcID,
			}},
			Selectable: true,
		})
		elements.Edges = append(elements.Edges, cy.Edge{
			Data: cy.EdgeData{
				ID:         fmt.Sprintf("%s_parent_%s", id, vpcID),
				Source:     id,
				Target:     vpcID,
				Attributes: map[string]any{"type": "parent"},
			},
			Selectable: true,
		})
	}

	// Internet Gateway nodes + edges
	for _, igw := range igws {
		id := aws.ToString(igw.InternetGatewayId)
		name := tagValue(igw.Tags, "Name", id)
		vpcID := ""
		for _, att := range igw.Attachments {
			// Floci uses "available" state; real AWS uses "attached"
			if att.State == ec2types.AttachmentStatusAttached || att.State == "available" {
				vpcID = aws.ToString(att.VpcId)
				break
			}
		}
		clusterIdx := vpcIndex[vpcID]
		elements.Nodes = append(elements.Nodes, cy.Node{
			Data: cy.NodeData{ID: id, Attributes: map[string]any{
				"label":   fmt.Sprintf("IGW: %s", shortID(name, id)),
				"group":   "igw",
				"cluster": clusterIdx,
				"vpcId":   vpcID,
			}},
			Selectable: true,
		})
		if vpcID != "" {
			elements.Edges = append(elements.Edges, cy.Edge{
				Data: cy.EdgeData{
					ID:         fmt.Sprintf("%s_attach_%s", id, vpcID),
					Source:     id,
					Target:     vpcID,
					Attributes: map[string]any{"type": "attached"},
				},
				Selectable: true,
			})
		}
	}

	// Load Balancer nodes + edges
	lbARNToID := make(map[string]string)
	for _, lb := range lbs {
		arn := aws.ToString(lb.LoadBalancerArn)
		name := aws.ToString(lb.LoadBalancerName)
		scheme := string(lb.Scheme)
		dns := aws.ToString(lb.DNSName)
		vpcID := ""
		clusterIdx := -1
		if lb.VpcId != nil {
			vpcID = aws.ToString(lb.VpcId)
			if _, ok := vpcIndex[vpcID]; !ok {
				// Floci may return a fake VPC ID; derive from subnets instead
				vpcID = ""
			}
		}
		// Fallback: derive VPC from the ALB's first subnet
		if vpcID == "" {
			for _, az := range lb.AvailabilityZones {
				sid := aws.ToString(az.SubnetId)
				if sid == "" {
					sid = aws.ToString(az.ZoneName)
				}
				if v, ok := subnetVPC[sid]; ok {
					vpcID = v
					break
				}
			}
		}
		if vpcID != "" {
			clusterIdx = vpcIndex[vpcID]
		}
		nodeID := fmt.Sprintf("alb_%s", name)
		lbARNToID[arn] = nodeID
		elements.Nodes = append(elements.Nodes, cy.Node{
			Data: cy.NodeData{ID: nodeID, Attributes: map[string]any{
				"label":   fmt.Sprintf("ALB: %s (%s)", name, scheme),
				"group":   "elb",
				"cluster": clusterIdx,
				"dns":     dns,
				"type":    string(lb.Type),
				"scheme":  scheme,
				"vpcId":   vpcID,
			}},
			Selectable: true,
		})
		// ALB → Subnet edges
		for _, az := range lb.AvailabilityZones {
			// Floci puts subnet ID in ZoneName; real AWS uses SubnetId
			sid := aws.ToString(az.SubnetId)
			if sid == "" {
				sid = aws.ToString(az.ZoneName)
			}
			if sid == "" {
				continue
			}
			elements.Edges = append(elements.Edges, cy.Edge{
				Data: cy.EdgeData{
					ID:         fmt.Sprintf("%s_subnet_%s", nodeID, sid),
					Source:     nodeID,
					Target:     sid,
					Attributes: map[string]any{"type": "lb-subnet"},
				},
				Selectable: true,
			})
		}
		if vpcID != "" {
			elements.Edges = append(elements.Edges, cy.Edge{
				Data: cy.EdgeData{
					ID:         fmt.Sprintf("%s_parent_%s", nodeID, vpcID),
					Source:     nodeID,
					Target:     vpcID,
					Attributes: map[string]any{"type": "parent"},
				},
				Selectable: true,
			})
		}
	}

	// ALB → Instance edges (via target groups)
	for lbARN, tgARNs := range lbToTGs {
		lbNodeID := lbARNToID[lbARN]
		if lbNodeID == "" {
			continue
		}
		for _, tgARN := range tgARNs {
			for _, instID := range tgToInstances[tgARN] {
				elements.Edges = append(elements.Edges, cy.Edge{
					Data: cy.EdgeData{
						ID:         fmt.Sprintf("%s_target_%s", lbNodeID, instID),
						Source:     lbNodeID,
						Target:     instID,
						Attributes: map[string]any{"type": "target"},
					},
					Selectable: true,
				})
			}
		}
	}

	// ── Internet egress (synthetic node) ──
	// Only create the internet node if at least one egress edge exists.
	hasInternet := false
	for _, igw := range igws {
		for _, att := range igw.Attachments {
			if att.State == ec2types.AttachmentStatusAttached || att.State == "available" {
				hasInternet = true
				break
			}
		}
		if hasInternet {
			break
		}
	}
	if !hasInternet {
		for _, lb := range lbs {
			if lb.Scheme == elbtypes.LoadBalancerSchemeEnumInternetFacing {
				hasInternet = true
				break
			}
		}
	}

	if hasInternet {
		elements.Nodes = append(elements.Nodes, cy.Node{
			Data: cy.NodeData{ID: "internet", Attributes: map[string]any{
				"label": "Internet",
				"group": "internet",
			}},
			Selectable: true,
		})

		// IGW → internet
		for _, igw := range igws {
			id := aws.ToString(igw.InternetGatewayId)
			attached := false
			for _, att := range igw.Attachments {
				if att.State == ec2types.AttachmentStatusAttached || att.State == "available" {
					attached = true
					break
				}
			}
			if attached {
				elements.Edges = append(elements.Edges, cy.Edge{
					Data: cy.EdgeData{
						ID:         fmt.Sprintf("%s_egress_internet", id),
						Source:     id,
						Target:     "internet",
						Attributes: map[string]any{"type": "egress"},
					},
					Selectable: true,
				})
			}
		}

		// internet-facing ALB → internet
		for _, lb := range lbs {
			if lb.Scheme != elbtypes.LoadBalancerSchemeEnumInternetFacing {
				continue
			}
			nodeID := fmt.Sprintf("alb_%s", aws.ToString(lb.LoadBalancerName))
			elements.Edges = append(elements.Edges, cy.Edge{
				Data: cy.EdgeData{
					ID:         fmt.Sprintf("%s_egress_internet", nodeID),
					Source:     nodeID,
					Target:     "internet",
					Attributes: map[string]any{"type": "egress"},
				},
				Selectable: true,
			})
		}
	}

	// Sort for deterministic output
	sort.Slice(elements.Nodes, func(i, j int) bool {
		return elements.Nodes[i].Data.ID < elements.Nodes[j].Data.ID
	})
	sort.Slice(elements.Edges, func(i, j int) bool {
		return elements.Edges[i].Data.ID < elements.Edges[j].Data.ID
	})

	return elements
}

// ----- Helpers -----

func tagValue(tags []ec2types.Tag, key, fallback string) string {
	for _, t := range tags {
		if aws.ToString(t.Key) == key {
			return aws.ToString(t.Value)
		}
	}
	return fallback
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func shortID(name, id string) string {
	if name == id {
		if len(id) > 12 {
			return id[:8]
		}
		return id
	}
	return name
}
