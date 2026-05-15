package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

func TestBuildTopology_Empty(t *testing.T) {
	elements := buildTopology(nil, nil, nil, nil, nil, nil, nil, nil)
	if len(elements.Nodes) != 0 || len(elements.Edges) != 0 {
		t.Errorf("expected empty topology, got %d nodes, %d edges",
			len(elements.Nodes), len(elements.Edges))
	}
}

func TestBuildTopology_SingleVpcWithSubnet(t *testing.T) {
	vpcs := []ec2types.Vpc{
		{VpcId: aws.String("vpc-abc123"), CidrBlock: aws.String("10.0.0.0/16"), IsDefault: aws.Bool(false)},
	}
	subnets := []ec2types.Subnet{
		{SubnetId: aws.String("subnet-abc"), VpcId: aws.String("vpc-abc123"), CidrBlock: aws.String("10.0.1.0/24"), AvailabilityZone: aws.String("us-east-1a"), MapPublicIpOnLaunch: aws.Bool(true)},
	}
	elements := buildTopology(vpcs, subnets, nil, nil, nil, nil, nil, nil)
	if len(elements.Nodes) != 2 { t.Fatalf("expected 2 nodes, got %d", len(elements.Nodes)) }
	if len(elements.Edges) != 1 { t.Fatalf("expected 1 edge, got %d", len(elements.Edges)) }

	var vpcNode, subnetNode *cy.Node
	for i := range elements.Nodes {
		switch elements.Nodes[i].Data.ID {
		case "vpc-abc123": vpcNode = &elements.Nodes[i]
		case "subnet-abc": subnetNode = &elements.Nodes[i]
		}
	}
	if vpcNode == nil { t.Fatal("missing VPC node") }
	if vpcNode.Data.Attributes["group"] != "vpc" { t.Errorf("VPC group = %q, want vpc", vpcNode.Data.Attributes["group"]) }
	if subnetNode == nil { t.Fatal("missing subnet node") }
	if subnetNode.Data.Attributes["group"] != "subnet" { t.Errorf("subnet group = %q, want subnet", subnetNode.Data.Attributes["group"]) }

	edge := elements.Edges[0]
	if edge.Data.Source != "subnet-abc" || edge.Data.Target != "vpc-abc123" {
		t.Errorf("edge = %s→%s, want subnet-abc→vpc-abc123", edge.Data.Source, edge.Data.Target)
	}
	if edge.Data.Attributes["type"] != "parent" { t.Errorf("edge type = %q, want parent", edge.Data.Attributes["type"]) }
}

func TestBuildTopology_InstanceWithSG(t *testing.T) {
	vpcs := []ec2types.Vpc{{VpcId: aws.String("vpc-1"), CidrBlock: aws.String("10.0.0.0/16")}}
	subnets := []ec2types.Subnet{{SubnetId: aws.String("subnet-1"), VpcId: aws.String("vpc-1"), CidrBlock: aws.String("10.0.1.0/24"), AvailabilityZone: aws.String("us-east-1a")}}
	instances := []ec2types.Instance{{
		InstanceId: aws.String("i-001"), InstanceType: ec2types.InstanceTypeT3Micro, SubnetId: aws.String("subnet-1"),
		PrivateIpAddress: aws.String("10.0.1.10"), State: &ec2types.InstanceState{Name: ec2types.InstanceStateNameRunning},
		SecurityGroups: []ec2types.GroupIdentifier{{GroupId: aws.String("sg-web")}},
		Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("web-1")}},
	}}
	sgs := []ec2types.SecurityGroup{{GroupId: aws.String("sg-web"), GroupName: aws.String("web-sg"), VpcId: aws.String("vpc-1"), Description: aws.String("Web traffic")}}

	elements := buildTopology(vpcs, subnets, instances, sgs, nil, nil, nil, nil)
	if len(elements.Nodes) != 4 { t.Fatalf("expected 4 nodes, got %d", len(elements.Nodes)) }
	if len(elements.Edges) != 4 { t.Fatalf("expected 4 edges, got %d", len(elements.Edges)) }

	var instNode, sgNode *cy.Node
	for i := range elements.Nodes {
		switch elements.Nodes[i].Data.ID {
		case "i-001": instNode = &elements.Nodes[i]
		case "sg-web": sgNode = &elements.Nodes[i]
		}
	}
	if instNode == nil { t.Fatal("missing instance node i-001") }
	if instNode.Data.Attributes["group"] != "instance" { t.Errorf("instance group = %q, want instance", instNode.Data.Attributes["group"]) }
	if sgNode == nil { t.Fatal("missing SG node sg-web") }

	found := false
	for _, e := range elements.Edges {
		if e.Data.Source == "i-001" && e.Data.Target == "sg-web" && e.Data.Attributes["type"] == "member" {
			found = true; break
		}
	}
	if !found { t.Error("missing instance→SG member edge") }
}

func TestBuildTopology_InternetGateway(t *testing.T) {
	vpcs := []ec2types.Vpc{{VpcId: aws.String("vpc-1"), CidrBlock: aws.String("10.0.0.0/16")}}
	igws := []ec2types.InternetGateway{{
		InternetGatewayId: aws.String("igw-abc"),
		Attachments: []ec2types.InternetGatewayAttachment{{State: ec2types.AttachmentStatusAttached, VpcId: aws.String("vpc-1")}},
	}}
	elements := buildTopology(vpcs, nil, nil, nil, igws, nil, nil, nil)
	if len(elements.Nodes) != 2 { t.Fatalf("expected 2 nodes, got %d", len(elements.Nodes)) }
	if len(elements.Edges) != 1 { t.Fatalf("expected 1 edge, got %d", len(elements.Edges)) }

	var igwNode *cy.Node
	for i := range elements.Nodes {
		if elements.Nodes[i].Data.ID == "igw-abc" { igwNode = &elements.Nodes[i]; break }
	}
	if igwNode == nil { t.Fatal("missing IGW node") }
	if igwNode.Data.Attributes["group"] != "igw" { t.Errorf("IGW group = %q, want igw", igwNode.Data.Attributes["group"]) }
	edge := elements.Edges[0]
	if edge.Data.Source != "igw-abc" || edge.Data.Target != "vpc-1" { t.Errorf("edge = %s→%s, want igw-abc→vpc-1", edge.Data.Source, edge.Data.Target) }
}

func TestBuildTopology_LoadBalancerWithTargets(t *testing.T) {
	vpcs := []ec2types.Vpc{{VpcId: aws.String("vpc-1"), CidrBlock: aws.String("10.0.0.0/16")}}
	subnets := []ec2types.Subnet{
		{SubnetId: aws.String("subnet-a"), VpcId: aws.String("vpc-1"), CidrBlock: aws.String("10.0.1.0/24"), AvailabilityZone: aws.String("us-east-1a")},
		{SubnetId: aws.String("subnet-b"), VpcId: aws.String("vpc-1"), CidrBlock: aws.String("10.0.2.0/24"), AvailabilityZone: aws.String("us-east-1b")},
	}
	instances := []ec2types.Instance{
		{InstanceId: aws.String("i-001"), SubnetId: aws.String("subnet-a"), PrivateIpAddress: aws.String("10.0.1.10"), State: &ec2types.InstanceState{Name: ec2types.InstanceStateNameRunning}},
		{InstanceId: aws.String("i-002"), SubnetId: aws.String("subnet-b"), PrivateIpAddress: aws.String("10.0.2.10"), State: &ec2types.InstanceState{Name: ec2types.InstanceStateNameRunning}},
	}
	lbs := []elbtypes.LoadBalancer{{
		LoadBalancerArn: aws.String("arn:aws:elbv2:us-east-1:000000000000:loadbalancer/app/my-alb/abc123"),
		LoadBalancerName: aws.String("my-alb"), Type: elbtypes.LoadBalancerTypeEnumApplication,
		Scheme: elbtypes.LoadBalancerSchemeEnumInternetFacing, VpcId: aws.String("vpc-1"),
		DNSName: aws.String("my-alb.elb.amazonaws.com"),
		AvailabilityZones: []elbtypes.AvailabilityZone{{SubnetId: aws.String("subnet-a")}, {SubnetId: aws.String("subnet-b")}},
	}}
	tgToInstances := map[string][]string{"arn:tg/my-tg": {"i-001", "i-002"}}
	lbToTGs := map[string][]string{"arn:aws:elbv2:us-east-1:000000000000:loadbalancer/app/my-alb/abc123": {"arn:tg/my-tg"}}

	elements := buildTopology(vpcs, subnets, instances, nil, nil, lbs, tgToInstances, lbToTGs)
	if len(elements.Nodes) != 6 { t.Fatalf("expected 6 nodes, got %d", len(elements.Nodes)) }

	var albNode *cy.Node
	for i := range elements.Nodes {
		if elements.Nodes[i].Data.ID == "alb_my-alb" { albNode = &elements.Nodes[i]; break }
	}
	if albNode == nil { t.Fatal("missing ALB node") }

	instanceEdges, subnetEdges := 0, 0
	for _, e := range elements.Edges {
		if e.Data.Source == "alb_my-alb" {
			switch e.Data.Attributes["type"] {
			case "target": instanceEdges++
			case "lb-subnet": subnetEdges++
			}
		}
	}
	if instanceEdges != 2 { t.Errorf("expected 2 target edges, got %d", instanceEdges) }
	if subnetEdges != 2 { t.Errorf("expected 2 lb-subnet edges, got %d", subnetEdges) }
}

func TestBuildTopology_TagValues(t *testing.T) {
	tags := []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("my-vpc")}, {Key: aws.String("Env"), Value: aws.String("prod")}}
	if v := tagValue(tags, "Name", "fallback"); v != "my-vpc" { t.Errorf("tagValue(Name) = %q, want my-vpc", v) }
	if v := tagValue(tags, "Missing", "fallback"); v != "fallback" { t.Errorf("tagValue(Missing) = %q, want fallback", v) }
}
