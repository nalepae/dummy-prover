```
    ____                                  ____
   / __ \__  ______ ___  ____ ___  __  __/ __ \_________  _   _____  _____
  / / / / / / / __ `__ \/ __ `__ \/ / / / /_/ / ___/ __ \| | / / _ \/ ___/
 / /_/ / /_/ / / / / / / / / / / / /_/ / ____/ /  / /_/ /| |/ /  __/ /
/_____/\__,_/_/ /_/ /_/_/ /_/ /_/\__, /_/   /_/   \____/ |___/\___/_/
                                /____/
```

# Dummy Prover

A dummy execution prover for Ethereum beacon nodes. It subscribes to block gossip events from a source beacon node, generates dummy proofs for each block, and submits them to a target beacon node.

## Overview

The dummy prover:

1. Connects to a source beacon node's SSE stream for `block` events
2. For each new block, fetches the signed blinded beacon block
3. Generates configurable number of dummy proofs in parallel
4. Submits all proofs to the target beacon node's `/eth/v1/prover/execution_proofs` endpoint

## Installation

```bash
go install github.com/nalepae/dummy-prover@latest
```

Or build from source:

```bash
go build -o dummy-prover .
```

Or build with Docker:

```bash
docker build -t dummy-prover:latest .
```

## Usage

```bash
dummy-prover [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-target-beacon-node` | `http://localhost:3500` | Beacon node HTTP endpoint to submit proofs to |
| `-source-beacon-node` | (same as target) | Beacon node HTTP endpoint to source blocks from |
| `-validator-client` | `http://localhost:7500` | Validator client HTTP endpoint for signing proofs |
| `-proofs-per-block` | `2` | Number of proof IDs to submit per block (max 8) |
| `-proof-delay-ms` | `1000` | Delay in milliseconds to simulate proof generation time |
| `-metrics-addr` | `:8080` | Address for the metrics/health HTTP server |

### Example

```bash
dummy-prover -target-beacon-node http://cl-2-prysm-geth:3500 -validator-client http://vc-2-geth-prysm:5056

Feb 20 14:26:01.493 INF Starting dummy prover source=http://cl-2-prysm-geth:3500 target=http://cl-2-prysm-geth:3500 validatorClient=http://vc-2-geth-prysm:5056 proofsPerBlock=2 proofDelayMs=1000
Feb 20 14:26:01.493 INF Starting health server addr=:8080
Feb 20 14:26:01.494 INF Connected to SSE stream event=block url="http://cl-2-prysm-geth:3500/eth/v1/events?topics=block"
Feb 20 14:26:28.234 INF Submitted dummy proofs blockRoot=0x94d9283bec01c07cc9778dab6e9f40bb93174f5f7807f415ee9cb4cd16e4dd59 slot=1 count=2
Feb 20 14:26:32.192 INF Submitted dummy proofs blockRoot=0x10aa190a0988e972881ed8577c9c25908b476ec77cd98c1b7b0c3c46a03b73ea slot=2 count=2
Feb 20 14:26:36.177 INF Submitted dummy proofs blockRoot=0x3457d2d4437d9a755ef862166739f7c6e502ffeb7dedbf17eec0af9ca58e4ca9 slot=3 count=2
Feb 20 14:26:40.193 INF Submitted dummy proofs blockRoot=0xe5e04f6bb879e8280fa0ea095c844566489d24ca201a3b32bf19f4dfb33b98a1 slot=4 count=2
Feb 20 14:26:44.180 INF Submitted dummy proofs blockRoot=0x23d993b195fe9e635a4023a2906d29fab28f921aa61326d6add3e3a134594dbe slot=5 count=2
Feb 20 14:26:48.182 INF Submitted dummy proofs blockRoot=0xcf5db56d0dba64b5af8aa589f561717e93aa203525d86606a4faaf8a9472b0bd slot=6 count=2
Feb 20 14:26:52.232 INF Submitted dummy proofs blockRoot=0x79a322295b123ccc7c8f80962f16f678a8949372d0523f616bde7b93acadfbd2 slot=7 count=2
Feb 20 14:26:56.230 INF Submitted dummy proofs blockRoot=0x3c4505727352be8772f63ed505d8e587b464e5fdddc46955cc4986af4ec5e413 slot=8 count=2
Feb 20 14:27:00.183 INF Submitted dummy proofs blockRoot=0xd1900400c1f307c5c0cdc357ca1a6e08b184d56f23a0d13ddbe99a4de49faa39 slot=9 count=2
Feb 20 14:27:04.181 INF Submitted dummy proofs blockRoot=0x8719143ab9ea4103db756504330974de2c894b4b5a9027919dada8010052fa8c slot=10 count=2

Feb  6 18:08:10.023 INF Shutdown requested signal=interrupt
```

## Proof Format

Each dummy [execution proof](https://github.com/ethereum/consensus-specs/blob/master/specs/_features/eip8025/beacon-chain.md#new-executionproof) contains:

- **proof_data**: `[0xFF, proof_type, block_hash[0:4]]`
- **proof_type**: Sequential ID from 0 to `proofs-per-block - 1`
- **public_input**: SSZ hash tree root of the `NewPayloadRequestHeader`

