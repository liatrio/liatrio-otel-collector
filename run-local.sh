#!/bin/bash

# Load environment variables from local.env
if [ -f local.env ]; then
    source local.env
else
    echo "Error: local.env file not found"
    echo "Please create local.env with:"
    echo "  export ADO_ORG=\"your-org\""
    echo "  export ADO_PROJECT=\"your-project\""
    echo "  export ADO_PAT=\"your-personal-access-token\""
    exit 1
fi

# Run the collector
./build/otelcol-custom --config=config/config-local-dev.yaml
