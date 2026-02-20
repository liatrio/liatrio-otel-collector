#!/bin/bash

echo "=== Verifying Local Development Setup ==="
echo ""

# Check if Docker services are running
echo "1. Checking Docker services..."
if docker ps | grep -q opensearch; then
    echo "   ✅ OpenSearch is running"
else
    echo "   ❌ OpenSearch is not running"
    echo "      Run: docker-compose up -d"
fi

if docker ps | grep -q grafana; then
    echo "   ✅ Grafana is running"
else
    echo "   ❌ Grafana is not running"
    echo "      Run: docker-compose up -d"
fi

echo ""

# Check if local.env exists
echo "2. Checking environment configuration..."
if [ -f local.env ]; then
    echo "   ✅ local.env file exists"
    
    # Check if required vars are set (without showing values)
    source local.env
    if [ -n "$ADO_ORG" ]; then
        echo "   ✅ ADO_ORG is set"
    else
        echo "   ❌ ADO_ORG is not set in local.env"
    fi
    
    if [ -n "$ADO_PROJECT" ]; then
        echo "   ✅ ADO_PROJECT is set"
    else
        echo "   ❌ ADO_PROJECT is not set in local.env"
    fi
    
    if [ -n "$ADO_PAT" ]; then
        echo "   ✅ ADO_PAT is set"
    else
        echo "   ❌ ADO_PAT is not set in local.env"
    fi
else
    echo "   ❌ local.env file not found"
    echo "      Create it with:"
    echo "      export ADO_ORG=\"your-org\""
    echo "      export ADO_PROJECT=\"your-project\""
    echo "      export ADO_PAT=\"your-personal-access-token\""
fi

echo ""

# Check if collector binary exists
echo "3. Checking collector binary..."
if [ -f ./build/otelcol-custom ]; then
    echo "   ✅ Collector binary exists"
else
    echo "   ❌ Collector binary not found"
    echo "      Run: make build"
fi

echo ""

# Check OpenSearch health
echo "4. Checking OpenSearch health..."
if curl -s http://localhost:9200/_cluster/health > /dev/null 2>&1; then
    echo "   ✅ OpenSearch is accessible"
    
    # Check if index exists
    if curl -s http://localhost:9200/otel-logs > /dev/null 2>&1; then
        echo "   ✅ otel-logs index exists"
        
        # Count documents
        DOC_COUNT=$(curl -s http://localhost:9200/otel-logs/_count | grep -o '"count":[0-9]*' | cut -d':' -f2)
        echo "      Documents in index: $DOC_COUNT"
    else
        echo "   ⚠️  otel-logs index does not exist yet"
        echo "      This is normal - it will be created when the collector sends the first logs"
    fi
else
    echo "   ❌ Cannot connect to OpenSearch"
    echo "      Make sure Docker services are running"
fi

echo ""

# Check Grafana
echo "5. Checking Grafana..."
if curl -s http://localhost:3000/api/health > /dev/null 2>&1; then
    echo "   ✅ Grafana is accessible at http://localhost:3000"
    echo "      Login: admin/admin"
else
    echo "   ❌ Cannot connect to Grafana"
fi

echo ""
echo "=== Next Steps ==="
echo ""
echo "If all checks pass, run the collector:"
echo "  ./run-local.sh"
echo ""
echo "Or manually:"
echo "  source local.env && ./build/otelcol-custom --config=config/config-local-dev.yaml"
echo ""
echo "After the collector runs for ~10 seconds, the otel-logs index will be created."
echo "Then you can access the Grafana dashboard at:"
echo "  http://localhost:3000/d/azure-devops-pipeline-metrics"
echo ""
