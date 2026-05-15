package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/neverthenetwork/inventa/internal/config"
)

func TestGetTagValue(t *testing.T) {
	tags := []ec2types.Tag{
		{Key: aws.String("Name"), Value: aws.String("my-vpc")},
		{Key: aws.String("env"), Value: aws.String("prod")},
	}

	result := getTagValue(tags, "Name", "default")
	if result != "my-vpc" {
		t.Errorf("expected 'my-vpc', got %q", result)
	}

	result = getTagValue(tags, "missing", "fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback', got %q", result)
	}

	result = getTagValue(nil, "Name", "default")
	if result != "default" {
		t.Errorf("expected 'default' for nil tags, got %q", result)
	}
}

func TestGetCIDRString(t *testing.T) {
	associations := []ec2types.VpcCidrBlockAssociation{
		{
			CidrBlock:      aws.String("10.0.0.0/16"),
			CidrBlockState: &ec2types.VpcCidrBlockState{State: ec2types.VpcCidrBlockStateCodeAssociated},
		},
		{
			CidrBlock:      aws.String("10.1.0.0/16"),
			CidrBlockState: &ec2types.VpcCidrBlockState{State: ec2types.VpcCidrBlockStateCodeAssociated},
		},
	}

	result := getCIDRString(associations)
	if result != "10.0.0.0/16, 10.1.0.0/16" {
		t.Errorf("expected '10.0.0.0/16, 10.1.0.0/16', got %q", result)
	}

	// Single CIDR
	single := []ec2types.VpcCidrBlockAssociation{
		{
			CidrBlock:      aws.String("172.16.0.0/16"),
			CidrBlockState: &ec2types.VpcCidrBlockState{State: ec2types.VpcCidrBlockStateCodeAssociated},
		},
	}
	if result := getCIDRString(single); result != "172.16.0.0/16" {
		t.Errorf("expected '172.16.0.0/16', got %q", result)
	}

	// Empty
	if result := getCIDRString(nil); result != "" {
		t.Errorf("expected empty for nil, got %q", result)
	}

	// Disassociated CIDR (should be skipped)
	disassociated := []ec2types.VpcCidrBlockAssociation{
		{
			CidrBlock:      aws.String("10.0.0.0/16"),
			CidrBlockState: &ec2types.VpcCidrBlockState{State: ec2types.VpcCidrBlockStateCodeDisassociated},
		},
	}
	if result := getCIDRString(disassociated); result != "" {
		t.Errorf("expected empty for disassociated CIDR, got %q", result)
	}
}

func TestPluginName(t *testing.T) {
	p := &Plugin{}
	if p.Name() != "aws" {
		t.Errorf("expected name 'aws', got %q", p.Name())
	}
}

func TestNew_missingRegions(t *testing.T) {
	cfg := &config.Conf{}
	// cfg.Sources.AWS.Regions is empty

	_, err := New(cfg, nil)
	if err == nil {
		t.Error("expected error for missing regions")
	}
}
