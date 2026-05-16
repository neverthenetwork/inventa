#!/bin/sh
# ── Inventa Demo Topology Seed ──────────────────────────────────
# Creates a rich AWS topology in Floci for the inventa demo.
# Runs as a one-shot container after Floci is healthy.
#
# Resources created:
#   VPC (10.100.0.0/16) with DNS hostnames
#   ├── Subnet public-a (10.100.1.0/24, us-east-1a)
#   │   └── EC2 inventa-web-a (t3.micro)
#   ├── Subnet public-b (10.100.2.0/24, us-east-1b)
#   │   └── EC2 inventa-web-b (t3.micro)
#   ├── SG demo-web-sg (HTTP:80 ingress)
#   ├── IGW demo-igw (attached)
#   └── ALB demo-alb (internet-facing) → TG → web-a, web-b

set -eu

AWS="aws --endpoint-url http://floci:4566 --region us-east-1"

echo "=== Cleaning up Floci default resources ==="
echo "[cleanup] Removing default VPC..."

# Detach and delete default IGW
$AWS ec2 detach-internet-gateway --internet-gateway-id igw-default --vpc-id vpc-default 2>/dev/null || true
$AWS ec2 delete-internet-gateway --internet-gateway-id igw-default 2>/dev/null || true

# Delete default subnets
for sid in subnet-default-a subnet-default-b subnet-default-c; do
  $AWS ec2 delete-subnet --subnet-id $sid 2>/dev/null || true
done

# Delete default SG
$AWS ec2 delete-security-group --group-id sg-default 2>/dev/null || true

# Delete default VPC (last, no cascade in Floci)
$AWS ec2 delete-vpc --vpc-id vpc-default 2>/dev/null || true

echo "  Default resources removed"
echo ""

echo "=== Seeding demo topology ==="

# ── VPC ──
echo "[1/9] Creating VPC..."
VPC_ID=$($AWS ec2 create-vpc \
  --cidr-block 10.100.0.0/16 \
  --tag-specifications "ResourceType=vpc,Tags=[{Key=Name,Value=demo-vpc}]" \
  --query "Vpc.VpcId" --output text)
echo "  VPC: $VPC_ID"

$AWS ec2 modify-vpc-attribute --vpc-id "$VPC_ID" --enable-dns-hostnames
$AWS ec2 modify-vpc-attribute --vpc-id "$VPC_ID" --enable-dns-support

# ── Subnets ──
echo "[2/9] Creating subnets..."
SUBNET_A=$($AWS ec2 create-subnet \
  --vpc-id "$VPC_ID" --cidr-block 10.100.1.0/24 --availability-zone us-east-1a \
  --tag-specifications "ResourceType=subnet,Tags=[{Key=Name,Value=public-a}]" \
  --query "Subnet.SubnetId" --output text)
echo "  Subnet A: $SUBNET_A"

SUBNET_B=$($AWS ec2 create-subnet \
  --vpc-id "$VPC_ID" --cidr-block 10.100.2.0/24 --availability-zone us-east-1b \
  --tag-specifications "ResourceType=subnet,Tags=[{Key=Name,Value=public-b}]" \
  --query "Subnet.SubnetId" --output text)
echo "  Subnet B: $SUBNET_B"

# ── Internet Gateway ──
echo "[3/9] Creating Internet Gateway..."
IGW_ID=$($AWS ec2 create-internet-gateway \
  --tag-specifications "ResourceType=internet-gateway,Tags=[{Key=Name,Value=demo-igw}]" \
  --query "InternetGateway.InternetGatewayId" --output text)
echo "  IGW: $IGW_ID"

$AWS ec2 attach-internet-gateway --internet-gateway-id "$IGW_ID" --vpc-id "$VPC_ID"
echo "  IGW attached to VPC"

# ── Security Group ──
echo "[4/9] Creating Security Group..."
SG_ID=$($AWS ec2 create-security-group \
  --group-name demo-web-sg --description "Demo web traffic (HTTP:80)" \
  --vpc-id "$VPC_ID" \
  --query "GroupId" --output text)
echo "  SG: $SG_ID"

$AWS ec2 authorize-security-group-ingress \
  --group-id "$SG_ID" --protocol tcp --port 80 --cidr 0.0.0.0/0
echo "  SG ingress: HTTP:80 from 0.0.0.0/0"

# ── EC2 Instances ──
echo "[5/9] Launching EC2 instance web-a..."
INST_A=$($AWS ec2 run-instances \
  --image-id ami-amazonlinux2023 --instance-type t3.micro \
  --subnet-id "$SUBNET_A" --security-group-ids "$SG_ID" \
  --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=inventa-web-a}]" \
  --query "Instances[0].InstanceId" --output text)
echo "  Instance A: $INST_A"

echo "[6/9] Launching EC2 instance web-b..."
INST_B=$($AWS ec2 run-instances \
  --image-id ami-amazonlinux2023 --instance-type t3.micro \
  --subnet-id "$SUBNET_B" --security-group-ids "$SG_ID" \
  --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=inventa-web-b}]" \
  --query "Instances[0].InstanceId" --output text)
echo "  Instance B: $INST_B"

# ── Application Load Balancer ──
echo "[7/9] Creating ALB..."
ALB_ARN=$($AWS elbv2 create-load-balancer \
  --name demo-alb --subnets "$SUBNET_A" "$SUBNET_B" \
  --security-groups "$SG_ID" --scheme internet-facing --type application \
  --query "LoadBalancers[0].LoadBalancerArn" --output text)
echo "  ALB ARN: $ALB_ARN"

# ── Target Group ──
echo "[8/9] Creating Target Group + registering targets..."
TG_ARN=$($AWS elbv2 create-target-group \
  --name demo-tg --protocol HTTP --port 80 \
  --vpc-id "$VPC_ID" --target-type instance \
  --query "TargetGroups[0].TargetGroupArn" --output text)
echo "  TG ARN: $TG_ARN"

# Floci RegisterTargets may emit XML warnings — ignore them
$AWS elbv2 register-targets \
  --target-group-arn "$TG_ARN" \
  --targets "Id=$INST_A,Port=80" "Id=$INST_B,Port=80" 2>/dev/null || true
echo "  Targets registered: $INST_A, $INST_B"

# ── Listener ──
echo "[9/9] Creating ALB listener (HTTP:80 → TG)..."
$AWS elbv2 create-listener \
  --load-balancer-arn "$ALB_ARN" --protocol HTTP --port 80 \
  --default-actions "Type=forward,TargetGroupArn=$TG_ARN" > /dev/null
echo "  Listener created"

echo ""
echo "=== Demo topology seeded successfully ==="
echo "VPC:       $VPC_ID"
echo "Subnets:   $SUBNET_A, $SUBNET_B"
echo "IGW:       $IGW_ID"
echo "SG:        $SG_ID"
echo "Instances: $INST_A, $INST_B"
echo "ALB:       demo-alb"
