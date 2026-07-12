set -euo pipefail

PROJECT_ID=$(
  curl -fsS -X POST http://localhost:8082/api/v1/projects \
    -H "Content-Type: application/json" \
    -d '{
      "name": "Demo Project"
    }' |
  jq -r '.id'
)

echo "PROJECT_ID=${PROJECT_ID}"

SERVICE_ID=$(
  curl -fsS -X POST \
    "http://localhost:8082/api/v1/projects/${PROJECT_ID}/services" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "demo-backend",
      "upstream_url": "http://demo-backend:8081"
    }' |
  jq -r '.id'
)

echo "SERVICE_ID=${SERVICE_ID}"

ROUTE_ID=$(
  curl -fsS -X POST \
    "http://localhost:8082/api/v1/services/${SERVICE_ID}/routes" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "demo-api-v1",
      "path_prefix": "/api/v1",
      "strip_prefix": true,
      "timeout_ms": 3000,
      "enabled": true
    }' |
  jq -r '.id'
)

echo "ROUTE_ID=${ROUTE_ID}"