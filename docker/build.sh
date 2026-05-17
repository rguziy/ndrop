#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")"

cmd=${1:-help}

case "$cmd" in
  build)
    docker compose -f docker-compose.yml build --no-cache ndropd
    ;;
  up)
    docker compose -f docker-compose.yml down --remove-orphans
    docker compose -f docker-compose.yml build --no-cache ndropd
    docker compose -f docker-compose.yml up -d ndropd
    ;;
  help|--help|-h)
    cat <<'EOF'
Usage: ./docker/build.sh [COMMAND]

Commands:
  build   build the ndropd image only
  up      rebuild and start the ndropd service
  help    show this help message
EOF
    ;;
  *)
    echo "Unknown command: $cmd"
    echo "Run ./docker/build.sh help for usage."
    exit 1
    ;;
esac
