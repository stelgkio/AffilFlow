# Hyperledger Fabric in Docker (local dev)

AffilFlow does **not** embed a full Fabric topology in this repo. The maintained, Docker-based way to run Fabric locally is the official **[fabric-samples](https://github.com/hyperledger/fabric-samples) `test-network`**: it brings up orderer, peers, and CAs using Docker Compose.

## Prerequisites

- **Docker** and **Docker Compose** (or Docker Desktop)
- Enough disk/RAM for Fabric images (~several GB on first pull)

Install Fabric **samples, binaries, and Docker images** once (pick a stable release, e.g. 2.5 LTS):

- Follow: [Install the Samples, Binaries and Docker Images](https://hyperledger-fabric.readthedocs.io/en/latest/install.html)

Or use the helper script from fabric-samples:

```bash
curl -sSLO https://raw.githubusercontent.com/hyperledger/fabric/main/scripts/install-fabric.sh
chmod +x install-fabric.sh
./install-fabric.sh docker samples
```

## Clone fabric-samples

```bash
git clone https://github.com/hyperledger/fabric-samples.git
export FABRIC_SAMPLES_DIR="$(pwd)/fabric-samples"   # optional; used by our script
```

## Start the network (Docker)

From the repo root:

```bash
./scripts/fabric-test-network.sh up
```

Or manually:

```bash
cd "$FABRIC_SAMPLES_DIR/test-network"
./network.sh up createChannel -ca
```

This starts the usual dev topology (orderer + peers + CAs) as **containers**. Create a channel and (optionally) deploy sample chaincode using the [test-network README](https://github.com/hyperledger/fabric-samples/tree/main/test-network).

## Stop and clean

```bash
./scripts/fabric-test-network.sh down
```

Or: `cd .../test-network && ./network.sh down`

## Point AffilFlow at the network

With the API running **on your host** (not inside Docker), peers are typically reachable on **localhost**:

| Service | Default host port (test-network) |
|---------|----------------------------------|
| Orderer | `7050` |
| Org1 peer | `7051` |

Set in `backend/.env` (when the Go Fabric client is wired to `FABRIC_ENABLED=true`):

- `FABRIC_ENABLED=true`
- `FABRIC_NETWORK_CONFIG` — path to a **connection profile** (YAML/JSON). After `network.sh up`, fabric-samples often provides org connection files under `test-network/organizations/peerOrganizations/` (exact filename varies by release; use the file for **Org1**).
- `FABRIC_CHANNEL` — channel name you created (default from `network.sh` is often `mychannel`).
- `FABRIC_CHAINCODE` — deployed chaincode name.

Until the real SDK implementation is enabled, the API uses a **no-op** blockchain service when `FABRIC_ENABLED=false`.

## AffilFlow API in Docker + Fabric on Docker

If the API runs in **Compose** and Fabric runs in **test-network’s** Compose project, both must share a Docker network **or** you use host-published ports:

- **Simplest:** publish peer/orderer ports to the host (test-network default) and set the connection profile to use `host.docker.internal` (Mac/Windows) or the host gateway IP (Linux) instead of `localhost` from **inside** the API container.
- **Alternative:** connect the API container to the Fabric network:  
  `docker network ls` → find the test-network network (often `fabric_test` or `test-network_default`) →  
  `docker network connect <that_network> <affilflow_api_container>`  
  and use peer hostnames from the connection profile as seen on that network.

## See also

- [docs/09-blockchain-hyperledger-fabric.md](../../docs/09-blockchain-hyperledger-fabric.md)
- [docs/10-infrastructure-docker.md](../../docs/10-infrastructure-docker.md)
