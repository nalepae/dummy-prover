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

## Usage

```bash
dummy-prover [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-target-beacon-node` | `http://localhost:3500` | Beacon node HTTP endpoint to submit proofs to |
| `-source-beacon-node` | (same as target) | Beacon node HTTP endpoint to source blocks from |
| `-proofs-per-block` | `1` | Number of proof IDs to submit per block (max 8) |
| `-proof-delay-ms` | `1000` | Delay in milliseconds to simulate proof generation time |

### Example

```bash
dummy-prover -source-beacon-node http://0.0.0.0:34177 -target-beacon-node http://0.0.0.0:34177 -proofs-per-block 4 -proof-delay-ms 500

Feb  6 18:07:35.071 INF Starting dummy prover source=http://0.0.0.0:34177 target=http://0.0.0.0:34177 proofsPerBlock=1 proofDelayMs=1000
Feb  6 18:07:35.077 INF Connected to SSE stream event=block url="http://0.0.0.0:34177/eth/v1/events?topics=block"
Feb  6 18:07:37.177 INF Submitted dummy proofs blockRoot=0xdb10ce488d3e52c9385641b7324e896050eaf932e75724dc9bd85a4e4b1f6e7e slot=1256
Feb  6 18:07:41.196 INF Submitted dummy proofs blockRoot=0x79e3b83e30f50c91940699a3b41d9b39001cc38449119ea25ef5bfde71de1bd7 slot=1257
Feb  6 18:07:45.182 INF Submitted dummy proofs blockRoot=0x139b83f569d7036c74d2e64f8aacf32eeaaa1f2dd17a079bee987f8a959e931b slot=1258
Feb  6 18:07:49.184 INF Submitted dummy proofs blockRoot=0xba052cccea64caa3e90b87407e42f3a2b2a6da4a5db462dd50f3cf61d569617c slot=1259
Feb  6 18:07:53.181 INF Submitted dummy proofs blockRoot=0x327a0afa70e7b83913069ac93d6da9aa39fab23028ed0d45775afa6edff760f3 slot=1260
Feb  6 18:07:57.176 INF Submitted dummy proofs blockRoot=0x69bb7622e1d4ee593a2ce9c7b8afe1fa7e8efd15a2fc1afc02e3471c3d7d516b slot=1261
Feb  6 18:08:01.172 INF Submitted dummy proofs blockRoot=0x24e2e63419587eef9174e4d07bcb3ccf01992b5e7b3723767ca2328fa1e086a8 slot=1262
Feb  6 18:08:05.186 INF Submitted dummy proofs blockRoot=0xcef0cc3455e0334528fe17744b683c5ffc790b87cb2eaf07c2ddd30fffa761d8 slot=1263
Feb  6 18:08:09.159 INF Submitted dummy proofs blockRoot=0x82b3f3778fffaaed63c0d295d2241a5430e5812387fec79e245e259e8ed9fc17 slot=1264

Feb  6 18:08:10.023 INF Shutdown requested signal=interrupt
```

## Proof Format

Each dummy [execution proof](https://github.com/ethereum/consensus-specs/blob/master/specs/_features/eip8025/beacon-chain.md#new-executionproof) contains:

- **proof_data**: `[0xFF, proof_type, block_hash[0:4]]`
- **proof_type**: Sequential ID from 0 to `proofs-per-block - 1`
- **public_input**: SSZ hash tree root of the `NewPayloadRequestHeader`

