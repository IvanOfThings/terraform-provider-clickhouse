# ClickHouse Setup Guide

This guide covers setting up ClickHouse for use with the Terraform Provider, from simple standalone instances to clustered deployments.

## Table of Contents

- [Quick Start (Standalone)](#quick-start-standalone)
- [Cluster Setup](#cluster-setup)
- [Testing Prerequisites](#testing-prerequisites)
- [Provider Configuration](#provider-configuration)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)

## Quick Start (Standalone)

For development and testing, a standalone ClickHouse instance is the simplest option.

### Docker (Recommended)

```bash
# Start ClickHouse server
docker run -d \
  --name clickhouse-server \
  -p 9000:9000 \
  -p 8123:8123 \
  clickhouse/clickhouse-server:latest

# Verify it's running
docker ps | grep clickhouse

# Test connection
docker exec -it clickhouse-server clickhouse-client --query "SELECT 1"
```

### Provider Configuration

```hcl
provider "clickhouse" {
  host     = "127.0.0.1"
  port     = 9000              # Native protocol port (NOT 8123)
  username = "default"
  password = ""
}
```

### Environment Variables (Recommended)

```bash
export TF_CLICKHOUSE_HOST="127.0.0.1"
export TF_CLICKHOUSE_PORT=9000
export TF_CLICKHOUSE_USERNAME="default"
export TF_CLICKHOUSE_PASSWORD=""
```

```hcl
# Provider will use environment variables automatically
provider "clickhouse" {}
```

## Cluster Setup

For production use and testing cluster features, use the provided docker-compose configuration.

### Architecture

The cluster setup includes:
- **ZooKeeper**: Cluster coordination
- **ClickHouse Node 1 (clickhouse-01)**: Shard 1, Replica 1
- **ClickHouse Node 2 (clickhouse-02)**: Shard 2, Replica 1

### Start Cluster

```bash
# Navigate to repository root
cd /path/to/terraform-provider-clickhouse

# Start all services
docker-compose up -d

# Check status
docker-compose ps
```

Expected output:
```
NAME                 STATUS          PORTS
clickhouse-01        Up (healthy)    0.0.0.0:8123->8123/tcp, 0.0.0.0:9000->9000/tcp, 0.0.0.0:9009->9009/tcp
clickhouse-02        Up (healthy)    0.0.0.0:8124->8123/tcp, 0.0.0.0:9001->9000/tcp, 0.0.0.0:9010->9009/tcp
clickhouse-zookeeper Up (healthy)    0.0.0.0:2181->2181/tcp, 2888/tcp, 3888/tcp
```

### Cluster Configuration Files

The cluster uses configuration files in `docker/`:

```
docker/
├── clickhouse-01/
│   └── config.d/
│       └── cluster.xml    # Node 1 cluster configuration
└── clickhouse-02/
    └── config.d/
        └── cluster.xml    # Node 2 cluster configuration
```

**Note:** These configuration files define the cluster topology. The cluster name should match your provider configuration.

### Provider Configuration (Cluster)

```hcl
provider "clickhouse" {
  host            = "127.0.0.1"
  port            = 9000
  username        = "default"
  password        = ""
  default_cluster = "cluster"    # Must match cluster name in config files
}

resource "clickhouse_db" "my_db" {
  name    = "my_database"
  comment = "Clustered database"
  cluster = "cluster"            # Creates on all cluster nodes
}
```

### Using Altinity Operator Macros

For Kubernetes deployments with Altinity ClickHouse Operator:

```hcl
provider "clickhouse" {
  host            = "127.0.0.1"
  port            = 9000
  username        = "default"
  password        = ""
  default_cluster = "'{cluster}'"    # Macro will be resolved by operator
}

resource "clickhouse_db" "my_db" {
  name    = "my_database"
  cluster = "'{cluster}'"
}

resource "clickhouse_table" "my_table" {
  database      = clickhouse_db.my_db.name
  name          = "replicated_table"
  cluster       = "'{cluster}'"
  engine        = "ReplicatedMergeTree"
  engine_params = [
    "'/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}'",
    "'{replica}'"
  ]
  order_by = ["id"]

  columns {
    name = "id"
    type = "UInt32"
  }
  columns {
    name = "data"
    type = "String"
  }
}
```

Supported macros:
- `{cluster}` - Cluster name
- `{installation}` - Installation name
- `{replica}` - Replica identifier
- `{shard}` - Shard identifier

## Testing Prerequisites

### For Acceptance Tests

Acceptance tests require a ClickHouse instance running on `localhost:9000`.

**Option 1: Use standalone Docker container**
```bash
docker run -d -p 9000:9000 -p 8123:8123 clickhouse/clickhouse-server
```

**Option 2: Use docker-compose cluster**
```bash
docker-compose up -d
# Tests connect to clickhouse-01 on port 9000
```

### Environment Variables for Tests

```bash
export TF_ACC=1                                  # Required: Enable acceptance tests
export TF_CLICKHOUSE_HOST="127.0.0.1"
export TF_CLICKHOUSE_PORT=9000
export TF_CLICKHOUSE_USERNAME="default"
export TF_CLICKHOUSE_PASSWORD=""

# Run tests
make testacc
```

### For Cluster Feature Tests

Some tests specifically require cluster configuration:

```bash
# Ensure cluster is running
docker-compose up -d

# Wait for health checks
docker-compose ps

# Set cluster name if needed (default is "cluster")
export TF_CLICKHOUSE_DEFAULT_CLUSTER="cluster"

# Run cluster-specific tests
TF_ACC=1 go test -v ./pkg/resources/db/... -run "Cluster"
```

## Provider Configuration

### Configuration Options

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `host` | string | Yes | - | ClickHouse server hostname or IP |
| `port` | int | Yes | - | Native protocol port (usually 9000, 9440 for Cloud) |
| `username` | string | Yes | - | ClickHouse username |
| `password` | string | Yes | - | ClickHouse password (empty string for default user) |
| `default_cluster` | string | No | - | Default cluster name for all resources |

### Example Configurations

**Standalone (Development):**
```hcl
provider "clickhouse" {
  host     = "127.0.0.1"
  port     = 9000
  username = "default"
  password = ""
}
```

**Cluster (Production):**
```hcl
provider "clickhouse" {
  host            = "clickhouse-lb.example.com"
  port            = 9000
  username        = "terraform"
  password        = var.clickhouse_password    # Use variable for sensitive data
  default_cluster = "production_cluster"
}
```

**ClickHouse Cloud:**
```hcl
provider "clickhouse" {
  host     = "abc123.clickhouse.cloud"
  port     = 9440                              # Secure native protocol
  username = "default"
  password = var.clickhouse_cloud_password
}
```

**Kubernetes with Altinity Operator:**
```hcl
provider "clickhouse" {
  host            = "chi-my-cluster-0-0.my-namespace.svc.cluster.local"
  port            = 9000
  username        = "default"
  password        = var.clickhouse_password
  default_cluster = "'{cluster}'"
}
```

### Security Best Practices

**Never commit credentials:**
```hcl
# BAD - credentials in code
provider "clickhouse" {
  password = "my-secret-password"
}

# GOOD - use environment variables
provider "clickhouse" {
  # Reads from TF_CLICKHOUSE_PASSWORD automatically
}

# GOOD - use Terraform variables
provider "clickhouse" {
  password = var.clickhouse_password
}
```

**Use environment variables:**
```bash
# Add to ~/.bashrc or ~/.zshrc
export TF_CLICKHOUSE_HOST="127.0.0.1"
export TF_CLICKHOUSE_PORT=9000
export TF_CLICKHOUSE_USERNAME="default"
export TF_CLICKHOUSE_PASSWORD=""
```

## Verification

### Verify Standalone Setup

```bash
# Test ClickHouse connectivity
docker exec -it clickhouse-server clickhouse-client

# Run simple query
SELECT version();

# Check if default user can connect
clickhouse-client --host 127.0.0.1 --port 9000 --user default --password '' --query "SELECT 1"
```

### Verify Cluster Setup

```bash
# Check cluster status on node 1
docker exec -it clickhouse-01 clickhouse-client --query "SELECT * FROM system.clusters"

# Check ZooKeeper connectivity
docker exec -it clickhouse-01 clickhouse-client --query "SELECT * FROM system.zookeeper WHERE path='/'"

# Verify nodes can see each other
docker exec -it clickhouse-01 clickhouse-client --query "SELECT * FROM system.clusters WHERE cluster='cluster'"
```

Expected output:
```
cluster  clickhouse-01  1  1  1  127.0.0.1  9000  0      0
cluster  clickhouse-02  2  1  1  127.0.0.1  9001  0      0
```

### Verify Provider Connection

Create a test configuration:

```hcl
# test.tf
provider "clickhouse" {
  host     = "127.0.0.1"
  port     = 9000
  username = "default"
  password = ""
}

data "clickhouse_dbs" "all" {}

output "databases" {
  value = data.clickhouse_dbs.all.names
}
```

Run:
```bash
terraform init
terraform plan
terraform apply

# Should output list of databases including 'system', 'default', etc.
```

## Troubleshooting

### Connection Issues

#### "Connection refused" on port 9000
**Problem:** ClickHouse not running or not listening on port 9000.

**Solution:**
```bash
# Check if container is running
docker ps | grep clickhouse

# Check port mapping
docker port clickhouse-server

# Check ClickHouse logs
docker logs clickhouse-server

# Restart container
docker restart clickhouse-server
```

#### "Connection refused" on port 8123
**Problem:** You're trying to use HTTP port with the provider.

**Solution:**
```hcl
# WRONG - HTTP port
provider "clickhouse" {
  port = 8123    # ❌
}

# CORRECT - Native protocol port
provider "clickhouse" {
  port = 9000    # ✅
}
```

#### "Authentication failed"
**Problem:** Wrong username or password.

**Solution:**
```bash
# Default user has empty password
TF_CLICKHOUSE_USERNAME=default
TF_CLICKHOUSE_PASSWORD=""

# Test connection
clickhouse-client --host 127.0.0.1 --port 9000 --user default --password ''
```

### Cluster Issues

#### Nodes cannot see each other
**Problem:** Network configuration or cluster XML mismatch.

**Solution:**
```bash
# Verify network connectivity
docker-compose exec clickhouse-01 ping clickhouse-02

# Check cluster configuration
docker-compose exec clickhouse-01 cat /etc/clickhouse-server/config.d/cluster.xml

# Restart cluster
docker-compose down
docker-compose up -d
```

#### ZooKeeper connection failed
**Problem:** ZooKeeper not running or not healthy.

**Solution:**
```bash
# Check ZooKeeper health
docker-compose ps zookeeper

# Should show "Up (healthy)"
# If not healthy, check logs
docker-compose logs zookeeper

# Restart ZooKeeper
docker-compose restart zookeeper

# Wait for health check
sleep 10
docker-compose ps
```

#### "Cluster not found"
**Problem:** Cluster name mismatch between provider and ClickHouse config.

**Solution:**
```bash
# Check cluster name in ClickHouse
docker exec -it clickhouse-01 clickhouse-client --query "SELECT cluster FROM system.clusters"

# Update provider configuration to match
provider "clickhouse" {
  default_cluster = "cluster"    # Must match output above
}
```

### Docker Compose Issues

#### Containers won't start
```bash
# Check for port conflicts
lsof -i :9000
lsof -i :9001
lsof -i :2181

# Stop conflicting services
docker-compose down

# Clean up volumes if needed
docker-compose down -v

# Start fresh
docker-compose up -d
```

#### Health checks failing
```bash
# Wait longer for initialization
docker-compose up -d
sleep 30
docker-compose ps

# Check specific service logs
docker-compose logs clickhouse-01
docker-compose logs zookeeper
```

### Testing Issues

#### Acceptance tests fail to connect
```bash
# Verify ClickHouse is accessible
nc -zv 127.0.0.1 9000

# Verify environment variables
echo $TF_ACC                    # Should be "1"
echo $TF_CLICKHOUSE_HOST        # Should be "127.0.0.1"
echo $TF_CLICKHOUSE_PORT        # Should be "9000"

# Test direct connection
clickhouse-client --host $TF_CLICKHOUSE_HOST --port $TF_CLICKHOUSE_PORT --query "SELECT 1"
```

#### Tests create resources but don't clean up
```bash
# Manually clean up test resources
docker exec -it clickhouse-server clickhouse-client

DROP ROLE IF EXISTS test_role_1;
DROP ROLE IF EXISTS test_role_2;
DROP DATABASE IF EXISTS role_role_db_1;
DROP DATABASE IF EXISTS role_role_db_2;
```

### Getting Help

If issues persist:

1. Check ClickHouse logs: `docker logs clickhouse-server`
2. Verify ClickHouse version: `SELECT version()` (provider tested with latest)
3. Review provider documentation: `docs/index.md`
4. Check CLAUDE.md for architecture details
5. File an issue with:
   - Docker version
   - ClickHouse version
   - Provider version
   - Error messages
   - Configuration (sanitize credentials!)
