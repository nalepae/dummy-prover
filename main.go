package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
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
	validatorClientURL := flag.String("validator-client", "http://localhost:7500", "Validator client HTTP endpoint for signing proofs")
	proofsPerBlock := flag.Int("proofs-per-block", 2, "Number of proof IDs to submit per block (max 8)")
	proofDelayMs := flag.Int("proof-delay-ms", 1000, "Delay in milliseconds to simulate proof generation time")
	proofDelayJitterMs := flag.Int("proof-delay-jitter-ms", 0, "Random jitter in milliseconds added to proof delay (±)")
	metricsAddr := flag.String("metrics-addr", ":8080", "Address for the metrics/health HTTP server")

	flag.Parse()

	// Start health/metrics HTTP server
	go startHealthServer(*metricsAddr)

	cfg := Config{
		TargetBeaconNode:   *targetBeaconNode,
		SourceBeaconNode:   *sourceBeaconNode,
		ValidatorClientURL: *validatorClientURL,
		ProofsPerBlock:     *proofsPerBlock,
		ProofDelayMs:       *proofDelayMs,
		ProofDelayJitterMs: *proofDelayJitterMs,
	}

	if err := run(cfg); err != nil {
		os.Exit(1)
	}
}

func startHealthServer(addr string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	logger.Info("Starting health server", "addr", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Health server error", "error", err)
	}
}

// Config holds the configuration for the dummy prover.
type Config struct {
	TargetBeaconNode   string
	SourceBeaconNode   string
	ValidatorClientURL string
	ProofsPerBlock     int
	ProofDelayMs       int
	ProofDelayJitterMs int
}

func run(cfg Config) error {
	// Use beacon-node as source if not specified
	sourceURL := cfg.SourceBeaconNode
	if sourceURL == "" {
		sourceURL = cfg.TargetBeaconNode
	}

	// Create beacon clients
	target := NewBeaconClient(cfg.TargetBeaconNode)
	source := NewBeaconClient(sourceURL)

	// Create validator client for signing
	validatorClient := NewValidatorClient(cfg.ValidatorClientURL)

	// Create prover
	prover := NewProver(source, target, validatorClient, cfg.ProofsPerBlock, time.Duration(cfg.ProofDelayMs)*time.Millisecond, time.Duration(cfg.ProofDelayJitterMs)*time.Millisecond)

	logger.Info("Starting dummy prover",
		"source", sourceURL,
		"target", cfg.TargetBeaconNode,
		"validatorClient", cfg.ValidatorClientURL,
		"proofsPerBlock", cfg.ProofsPerBlock,
		"proofDelayMs", cfg.ProofDelayMs,
		"proofDelayJitterMs", cfg.ProofDelayJitterMs,
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
