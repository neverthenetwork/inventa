//go:build integration
// +build integration

package aws

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	cy "gonum.org/v1/gonum/graph/formats/cytoscapejs"
)

const (
	flociEndpoint     = "http://localhost:4566"
	flociStartTimeout = 30 * time.Second
	testVpcCIDR       = "10.100.0.0/16"
)

// TestIntegration_FlociTopologyDiscovery starts Floci, creates a known AWS
// topology, runs the discovery engine, and validates the resulting graph.
func TestIntegration_FlociTopologyDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skipf("cannot find repo root: %v", err)
	}
	composeFile := repoRoot + "/test/integration/aws/docker-compose.yml"
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		t.Skipf("docker compose file not found: %s", composeFile)
	}

	// Start Floci
	t.Log("starting Floci...")
	startFloci(t, composeFile)
	t.Cleanup(func() { stopFloci(t, composeFile) })

	// Wait for Floci to be ready
	t.Log("waiting for Floci to be ready...")
	waitForFloci(t)

	// Create AWS client config pointing at Floci
	cfg, err := newFlociConfig(t.Context())
	if err != nil {
		t.Fatalf("failed to create AWS config: %v", err)
	}

	// Create topology resources
	t.Log("creating test topology...")
	resourceIDs := createTestTopology(t, cfg)

	// Run discovery
	t.Log("running topology discovery...")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	elementsCh := make(chan cy.Elements, 1)
	err = Discover(t.Context(), cfg, logger, func(elements cy.Elements) {
		elementsCh <- elements
	})
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	elements := <-elementsCh

	// Validate the discovered topology
	t.Logf("discovered %d nodes, %d edges", len(elements.Nodes), len(elements.Edges))
	validateDiscoveredTopology(t, elements, resourceIDs)
}

// newFlociConfig creates an AWS SDK config targeting the local Floci instance.
func newFlociConfig(ctx context.Context) (aws.Config, error) {
	//nolint:staticcheck // Deprecated resolver is still the cleanest way for multi-service endpoint override.
	resolver := aws.EndpointResolverWithOptionsFunc(
		func(_, region string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           flociEndpoint,
				SigningRegion: "us-east-1",
			}, nil
		},
	)

	return config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		//nolint:staticcheck // Coupled to resolver above.
		config.WithEndpointResolverWithOptions(resolver),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
}

// createTestTopology creates a known AWS topology and returns resource IDs.
//
// Topology:
//   VPC (10.100.0.0/16)
//   ├── Subnet public-a (10.100.1.0/24, us-east-1a)
//   │   └── EC2 inventa-web-a
//   ├── Subnet public-b (10.100.2.0/24, us-east-1b)
//   │   └── EC2 inventa-web-b
//   ├── Security Group inventa-test-sg
//   ├── Internet Gateway (attached)
//   └── ALB inventa-test-alb → target group → web-a, web-b
func createTestTopology(t *testing.T, cfg aws.Config) map[string]string {
	t.Helper()
	ctx := t.Context()
	ec2c := ec2.NewFromConfig(cfg)
	elbc := elasticloadbalancingv2.NewFromConfig(cfg)

	ids := make(map[string]string)

	// 1. Create VPC with Name tag
	vpc, err := ec2c.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String(testVpcCIDR),
		TagSpecifications: []ec2types.TagSpecification{
			{ResourceType: ec2types.ResourceTypeVpc, Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("inventa-test-vpc")}}},
		},
	})
	if err != nil { t.Fatalf("CreateVpc: %v", err) }
	ids["vpc"] = aws.ToString(vpc.Vpc.VpcId)
	t.Logf("  VPC: %s", ids["vpc"])

	// Enable DNS hostnames
	_, _ = ec2c.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId: vpc.Vpc.VpcId, EnableDnsHostnames: &ec2types.AttributeBooleanValue{Value: aws.Bool(true)},
	})

	// 2. Create subnets
	subnetA, err := ec2c.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		VpcId: vpc.Vpc.VpcId, CidrBlock: aws.String("10.100.1.0/24"), AvailabilityZone: aws.String("us-east-1a"),
		TagSpecifications: []ec2types.TagSpecification{
			{ResourceType: ec2types.ResourceTypeSubnet, Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("public-a")}}},
		},
	})
	if err != nil { t.Fatalf("CreateSubnet(a): %v", err) }
	ids["subnet-a"] = aws.ToString(subnetA.Subnet.SubnetId)

	subnetB, err := ec2c.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		VpcId: vpc.Vpc.VpcId, CidrBlock: aws.String("10.100.2.0/24"), AvailabilityZone: aws.String("us-east-1b"),
		TagSpecifications: []ec2types.TagSpecification{
			{ResourceType: ec2types.ResourceTypeSubnet, Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("public-b")}}},
		},
	})
	if err != nil { t.Fatalf("CreateSubnet(b): %v", err) }
	ids["subnet-b"] = aws.ToString(subnetB.Subnet.SubnetId)
	t.Logf("  Subnets: %s, %s", ids["subnet-a"], ids["subnet-b"])

	// 3. Create Internet Gateway and attach
	igw, err := ec2c.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
		TagSpecifications: []ec2types.TagSpecification{
			{ResourceType: ec2types.ResourceTypeInternetGateway, Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("inventa-test-igw")}}},
		},
	})
	if err != nil { t.Fatalf("CreateInternetGateway: %v", err) }
	ids["igw"] = aws.ToString(igw.InternetGateway.InternetGatewayId)
	_, _ = ec2c.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: igw.InternetGateway.InternetGatewayId, VpcId: vpc.Vpc.VpcId,
	})
	t.Logf("  IGW: %s (attached)", ids["igw"])

	// 4. Create security group + ingress rule
	sg, err := ec2c.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName: aws.String("inventa-test-sg"), Description: aws.String("inventa integration test"),
		VpcId: vpc.Vpc.VpcId,
	})
	if err != nil { t.Fatalf("CreateSecurityGroup: %v", err) }
	ids["sg"] = aws.ToString(sg.GroupId)
	_, _ = ec2c.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: sg.GroupId,
		IpPermissions: []ec2types.IpPermission{
			{IpProtocol: aws.String("tcp"), FromPort: aws.Int32(80), ToPort: aws.Int32(80), IpRanges: []ec2types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}}},
		},
	})
	t.Logf("  SG: %s", ids["sg"])

	// 5. Run instances
	instA, err := ec2c.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId: aws.String("ami-amazonlinux2023"), InstanceType: ec2types.InstanceTypeT3Micro,
		SubnetId: subnetA.Subnet.SubnetId, MinCount: aws.Int32(1), MaxCount: aws.Int32(1),
		SecurityGroupIds: []string{ids["sg"]},
		TagSpecifications: []ec2types.TagSpecification{
			{ResourceType: ec2types.ResourceTypeInstance, Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("inventa-web-a")}}},
		},
	})
	if err != nil { t.Fatalf("RunInstances(a): %v", err) }
	ids["instance-a"] = aws.ToString(instA.Instances[0].InstanceId)

	instB, err := ec2c.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId: aws.String("ami-amazonlinux2023"), InstanceType: ec2types.InstanceTypeT3Micro,
		SubnetId: subnetB.Subnet.SubnetId, MinCount: aws.Int32(1), MaxCount: aws.Int32(1),
		SecurityGroupIds: []string{ids["sg"]},
		TagSpecifications: []ec2types.TagSpecification{
			{ResourceType: ec2types.ResourceTypeInstance, Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("inventa-web-b")}}},
		},
	})
	if err != nil { t.Fatalf("RunInstances(b): %v", err) }
	ids["instance-b"] = aws.ToString(instB.Instances[0].InstanceId)
	t.Logf("  Instances: %s, %s", ids["instance-a"], ids["instance-b"])

	// 6. Create ALB
	alb, err := elbc.CreateLoadBalancer(ctx, &elasticloadbalancingv2.CreateLoadBalancerInput{
		Name: aws.String("inventa-test-alb"), Subnets: []string{ids["subnet-a"], ids["subnet-b"]},
		SecurityGroups: []string{ids["sg"]}, Scheme: elbtypes.LoadBalancerSchemeEnumInternetFacing,
		Type: elbtypes.LoadBalancerTypeEnumApplication,
	})
	if err != nil { t.Fatalf("CreateLoadBalancer: %v", err) }
	ids["alb"] = fmt.Sprintf("alb_%s", aws.ToString(alb.LoadBalancers[0].LoadBalancerName))
	ids["alb-arn"] = aws.ToString(alb.LoadBalancers[0].LoadBalancerArn)
	t.Logf("  ALB: %s", ids["alb"])

	// 7. Create target group + register targets
	tg, err := elbc.CreateTargetGroup(ctx, &elasticloadbalancingv2.CreateTargetGroupInput{
		Name: aws.String("inventa-test-tg"), Protocol: elbtypes.ProtocolEnumHttp, Port: aws.Int32(80),
		VpcId: vpc.Vpc.VpcId, TargetType: elbtypes.TargetTypeEnumInstance,
	})
	if err != nil { t.Fatalf("CreateTargetGroup: %v", err) }
	tgARN := aws.ToString(tg.TargetGroups[0].TargetGroupArn)
	ids["tg"] = tgARN

	_, _ = elbc.RegisterTargets(ctx, &elasticloadbalancingv2.RegisterTargetsInput{
		TargetGroupArn: aws.String(tgARN),
		Targets: []elbtypes.TargetDescription{
			{Id: aws.String(ids["instance-a"]), Port: aws.Int32(80)},
			{Id: aws.String(ids["instance-b"]), Port: aws.Int32(80)},
		},
	})

	// 8. Create listener
	_, _ = elbc.CreateListener(ctx, &elasticloadbalancingv2.CreateListenerInput{
		LoadBalancerArn: alb.LoadBalancers[0].LoadBalancerArn, Protocol: elbtypes.ProtocolEnumHttp,
		Port: aws.Int32(80),
		DefaultActions: []elbtypes.Action{
			{Type: elbtypes.ActionTypeEnumForward, TargetGroupArn: aws.String(tgARN)},
		},
	})
	t.Log("  Target group + listener created")

	return ids
}

// validateDiscoveredTopology asserts the discovered graph matches the expected topology.
func validateDiscoveredTopology(t *testing.T, elements cy.Elements, ids map[string]string) {
	t.Helper()

	nodeMap := make(map[string]cy.Node, len(elements.Nodes))
	for _, n := range elements.Nodes {
		nodeMap[n.Data.ID] = n
	}

	edgeSet := make(map[string]bool)
	for _, e := range elements.Edges {
		edgeSet[fmt.Sprintf("%s→%s(%s)", e.Data.Source, e.Data.Target, e.Data.Attributes["type"])] = true
	}

	vpcID := ids["vpc"]

	// VPC exists
	if _, ok := nodeMap[vpcID]; !ok {
		t.Errorf("missing VPC node: %s", vpcID)
	} else if nodeMap[vpcID].Data.Attributes["group"] != "vpc" {
		t.Errorf("VPC group = %q, want vpc", nodeMap[vpcID].Data.Attributes["group"])
	}

	// Subnets exist and have parent edges to VPC
	for _, key := range []string{"subnet-a", "subnet-b"} {
		sid := ids[key]
		if _, ok := nodeMap[sid]; !ok {
			t.Errorf("missing subnet node: %s (%s)", key, sid); continue
		}
		if nodeMap[sid].Data.Attributes["group"] != "subnet" {
			t.Errorf("%s group = %q, want subnet", key, nodeMap[sid].Data.Attributes["group"])
		}
		if !edgeSet[fmt.Sprintf("%s→%s(parent)", sid, vpcID)] {
			t.Errorf("missing parent edge: %s→%s", sid, vpcID)
		}
	}

	// Instances exist and have member edges to SG
	for _, key := range []string{"instance-a", "instance-b"} {
		iid := ids[key]
		if _, ok := nodeMap[iid]; !ok {
			t.Errorf("missing instance node: %s (%s)", key, iid); continue
		}
		if nodeMap[iid].Data.Attributes["group"] != "instance" {
			t.Errorf("%s group = %q, want instance", key, nodeMap[iid].Data.Attributes["group"])
		}
		if !edgeSet[fmt.Sprintf("%s→%s(member)", iid, ids["sg"])] {
			t.Errorf("missing instance→SG member edge: %s→%s", iid, ids["sg"])
		}
	}

	// SG exists
	sgID := ids["sg"]
	if _, ok := nodeMap[sgID]; !ok {
		t.Errorf("missing SG node: %s", sgID)
	} else {
		if nodeMap[sgID].Data.Attributes["group"] != "security_group" {
			t.Errorf("SG group = %q, want security_group", nodeMap[sgID].Data.Attributes["group"])
		}
		if !edgeSet[fmt.Sprintf("%s→%s(parent)", sgID, vpcID)] {
			t.Errorf("missing SG→VPC parent edge: %s→%s", sgID, vpcID)
		}
	}

	// IGW exists and attached to VPC
	igwID := ids["igw"]
	if _, ok := nodeMap[igwID]; !ok {
		t.Errorf("missing IGW node: %s", igwID)
	} else {
		if nodeMap[igwID].Data.Attributes["group"] != "igw" {
			t.Errorf("IGW group = %q, want igw", nodeMap[igwID].Data.Attributes["group"])
		}
		if !edgeSet[fmt.Sprintf("%s→%s(attached)", igwID, vpcID)] {
			t.Errorf("missing IGW→VPC attached edge: %s→%s", igwID, vpcID)
		}
	}

	// ALB exists
	albID := ids["alb"]
	if _, ok := nodeMap[albID]; !ok {
		t.Errorf("missing ALB node: %s", albID)
	} else {
		if nodeMap[albID].Data.Attributes["group"] != "elb" {
			t.Errorf("ALB group = %q, want elb", nodeMap[albID].Data.Attributes["group"])
		}
		if !edgeSet[fmt.Sprintf("%s→%s(parent)", albID, vpcID)] {
			t.Errorf("missing ALB→VPC parent edge: %s→%s", albID, vpcID)
		}
		// ALB → subnet edges
		for _, key := range []string{"subnet-a", "subnet-b"} {
			if !edgeSet[fmt.Sprintf("%s→%s(lb-subnet)", albID, ids[key])] {
				t.Errorf("missing ALB→%s lb-subnet edge: %s→%s", key, albID, ids[key])
			}
		}
		// ALB → instance target edges
		for _, key := range []string{"instance-a", "instance-b"} {
			if !edgeSet[fmt.Sprintf("%s→%s(target)", albID, ids[key])] {
				t.Errorf("missing ALB→%s target edge: %s→%s", key, albID, ids[key])
			}
		}
	}

	t.Logf("topology validation passed (%d nodes, %d edges total, including default resources)",
		len(elements.Nodes), len(elements.Edges))
}

func startFloci(t *testing.T, composeFile string) {
	t.Helper()
	cmd := exec.Command("sudo", "docker", "compose", "-f", composeFile, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to start Floci: %v", err)
	}
}

func stopFloci(t *testing.T, composeFile string) {
	t.Helper()
	cmd := exec.Command("sudo", "docker", "compose", "-f", composeFile, "down", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func waitForFloci(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(flociStartTimeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", "localhost:4566", 1*time.Second)
		if err == nil {
			conn.Close()
			time.Sleep(500 * time.Millisecond)
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("Floci did not become reachable within timeout")
}

var _ = strings.Join

// findRepoRoot walks up from the test file's directory to find the Go module root (where go.mod lives).
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found in any parent directory of %s", dir)
		}
		dir = parent
	}
}
