#!/bin/bash
set -euo pipefail

echo "=== Cartography: AWS discovery → Neo4j ==="

# Wait for Neo4j bolt port using Python (no netcat in slim image)
echo "Waiting for Neo4j bolt..."
python3 -c "
import socket, time
for i in range(30):
    try:
        s = socket.create_connection(('neo4j', 7687), timeout=2)
        s.close()
        print(f'Neo4j ready (bolt://neo4j:7687).')
        break
    except (ConnectionRefusedError, OSError):
        print(f'  attempt {i+1}/30...')
        time.sleep(2)
else:
    raise SystemExit('Neo4j did not become ready')
"

# Point AWS SDK at Floci (compose service name)
export AWS_ENDPOINT_URL=http://floci:4566
export AWS_DEFAULT_REGION=us-east-1
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1

echo "AWS endpoint:  $AWS_ENDPOINT_URL"
echo "Neo4j URI:     bolt://neo4j:7687"

# Run Cartography — use requested syncs to skip unsupported modules (IAM, etc.)
echo ""
echo "--- Running Cartography sync ---"
cartography \
  --neo4j-uri bolt://neo4j:7687 \
  --selected-modules aws \
  --aws-requested-syncs 'ec2:vpc,ec2:subnet,ec2:instance,ec2:security_group,ec2:internet_gateway,ec2:load_balancer_v2,ec2:keypair,elastic_ip_addresses'

echo ""
echo "=== Cartography sync complete ==="
