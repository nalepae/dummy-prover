package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
)

var logger = slog.New(tint.NewHandler(os.Stderr, &tint.Options{Level: slog.LevelInfo}))

const targetBeaconNodeFlag = "target-beacon-node"

func main() {
	targetBeaconNode := flag.String(targetBeaconNodeFlag, "http://localhost:3500", "Beacon node HTTP endpoint to submit proofs to")
	sourceBeaconNode := flag.String("source-beacon-node", "", fmt.Sprintf("Beacon node HTTP endpoint to source blocks from (defaults to -%s)", targetBeaconNodeFlag))
	proofsPerBlock := flag.Int("proofs-per-block", 1, "Number of proof IDs to submit per block (max 8)")
	proofDelayMs := flag.Int("proof-delay-ms", 1000, "Delay in milliseconds to simulate proof generation time")

	flag.Parse()

	if err := run(*targetBeaconNode, *sourceBeaconNode, *proofsPerBlock, *proofDelayMs); err != nil {
		os.Exit(1)
	}
}

func run(targetBeaconNode string, sourceBeaconNode string, proofsPerBlock int, proofDelayMs int) error {
	// Use beacon-node as source if not specified
	sourceURL := sourceBeaconNode
	if sourceURL == "" {
		sourceURL = targetBeaconNode
	}

	// Create beacon clients
	target := NewBeaconClient(targetBeaconNode)
	source := NewBeaconClient(sourceURL)

	// Create prover
	prover := NewProver(source, target, proofsPerBlock, time.Duration(proofDelayMs)*time.Millisecond)

	logger.Info("Starting dummy prover",
		"source", sourceURL,
		"target", targetBeaconNode,
		"proofsPerBlock", proofsPerBlock,
		"proofDelayMs", proofDelayMs,
	)

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Subscribe to block_gossip events from source
	events, errs := source.subscribeToBlockGossip(ctx)

	// Main event loop
	for {
		select {
		case sig := <-sigChan:
			logger.Info("Shutdown requested", "signal", sig)
			return nil

		case event, ok := <-events:
			if !ok {
				logger.Error("Event stream ended")
				return nil
			}

			if err := prover.handleBlockGossip(ctx, event); err != nil {
				logger.Error("Failed to handle block gossip", "blockRoot", fmt.Sprintf("%#x", event.Block), "slot", event.Slot, "error", err)
			}

		case err, ok := <-errs:
			if ok && err != nil {
				logger.Error("Event stream error", "error", err)
				return err
			}
		}
	}
}
