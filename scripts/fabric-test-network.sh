#!/usr/bin/env bash
# Start/stop Hyperledger Fabric test-network (Docker) from fabric-samples.
# Docs: infra/fabric/README.md
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SAMPLES="${FABRIC_SAMPLES_DIR:-$ROOT/../fabric-samples}"

usage() {
  echo "Usage: FABRIC_SAMPLES_DIR=/path/to/fabric-samples $0 up|down"
  echo ""
  echo "  up    Run: ./network.sh up createChannel -ca"
  echo "  down  Run: ./network.sh down"
  echo ""
  echo "Default FABRIC_SAMPLES_DIR: $ROOT/../fabric-samples (sibling of AffilFlow repo)"
  exit 1
}

if [[ $# -lt 1 ]]; then
  usage
fi

TN="$SAMPLES/test-network/network.sh"
if [[ ! -f "$TN" ]]; then
  echo "fabric-samples test-network not found at: $TN"
  echo "Clone: git clone https://github.com/hyperledger/fabric-samples.git \"$SAMPLES\""
  echo "Or set FABRIC_SAMPLES_DIR to your existing clone."
  exit 1
fi

cd "$SAMPLES/test-network"
case "${1:-}" in
  up)
    ./network.sh up createChannel -ca
    echo ""
    echo "Fabric test network is up (Docker). Next: see infra/fabric/README.md for AffilFlow env vars."
    ;;
  down)
    ./network.sh down
    ;;
  *)
    usage
    ;;
esac
