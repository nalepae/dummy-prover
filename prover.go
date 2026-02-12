package main

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

// Prover handles proof generation and submission.
type Prover struct {
	source         *BeaconClient
	target         *BeaconClient
	proofsPerBlock int
	proofDelay     time.Duration
}

// NewProver creates a new Prover instance.
func NewProver(source *BeaconClient, target *BeaconClient, proofsPerBlock int, proofDelay time.Duration) *Prover {
	return &Prover{
		source:         source,
		target:         target,
		proofsPerBlock: proofsPerBlock,
		proofDelay:     proofDelay,
	}
}

// handleBlockGossip processes a block gossip event by fetching the block and submitting proofs.
func (p *Prover) handleBlockGossip(ctx context.Context, event BlockEventData) error {
	signedBlindedBeaconBlock, err := p.source.GetSignedBlindedBeaconBlock(ctx, fmt.Sprintf("%d", event.Slot))
	if err != nil {
		return fmt.Errorf("get signed blinded beacon block: %w", err)
	}

	if err := p.generateAndSubmitDummyProofs(ctx, signedBlindedBeaconBlock); err != nil {
		return fmt.Errorf("generate and submit dummy proofs: %w", err)
	}

	logger.Info("Submitted dummy proofs", "blockRoot", fmt.Sprintf("%#x", event.Block), "slot", event.Slot)
	return nil
}

// generateAndSubmitDummyProofs generates and submits dummy proofs for a block.
func (p *Prover) generateAndSubmitDummyProofs(ctx context.Context, block *SignedBlindedBeaconBlock) error {
	// Generate all proofs in parallel
	var genGroup errgroup.Group

	proofs := make([]*SignedExecutionProof, p.proofsPerBlock)
	for proofType := range ProofType(p.proofsPerBlock) {
		genGroup.Go(func() error {
			proof, err := generateDummyProof(proofType, block)
			if err != nil {
				return fmt.Errorf("generate dummy proof %d: %w", proofType, err)
			}

			proofs[proofType] = proof
			return nil
		})
	}

	if err := genGroup.Wait(); err != nil {
		return err
	}

	// Simulate proof generation delay (wait once for all proofs)
	if p.proofDelay > 0 {
		select {
		case <-time.After(p.proofDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Submit all proofs in parallel
	var submitGroup errgroup.Group

	for proofType, proof := range proofs {
		submitGroup.Go(func() error {
			if err := p.target.SubmitSignedExecutionProof(ctx, proof); err != nil {
				return fmt.Errorf("submit proof %d: %w", proofType, err)
			}

			return nil
		})
	}

	if err := submitGroup.Wait(); err != nil {
		return err
	}

	return nil
}

// generateDummyProof creates a dummy signed proof with the standard format.
func generateDummyProof(proofType ProofType, signedBlindedBeaconBlock *SignedBlindedBeaconBlock) (*SignedExecutionProof, error) {
	beaconBlock := signedBlindedBeaconBlock.Message
	beaconBlockBody := beaconBlock.Body
	ExecutionPayloadHeader := beaconBlockBody.ExecutionPayloadHeader

	// Dummy proof data format: [0xFF, proofID, blockHash[0], blockHash[1], blockHash[2], blockHash[3]]
	blockHash := ExecutionPayloadHeader.BlockHash

	proofData := []byte{
		0xFF,
		byte(proofType),
		blockHash[0],
		blockHash[1],
		blockHash[2],
		blockHash[3],
	}

	newPayloadRequestHeader := &NewPayloadRequestHeader{
		ExecutionPayloadHeader: ExecutionPayloadHeader,
		VersionedHashes:        kzgCommitmentsToVersionedHashes(beaconBlockBody),
		ParentBeaconBlockRoot:  beaconBlock.ParentRoot, // <-- We cheat here as we should use state.latest_block_header.parent_root
		ExecutionRequests:      beaconBlockBody.ExecutionRequests,
	}

	newPayloadRequestRoot, err := newPayloadRequestHeader.HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("new payload request root: %w", err)
	}

	publicInput := &PublicInput{
		NewPayloadRequestRoot: newPayloadRequestRoot[:],
	}

	executionProof := &ExecutionProof{
		ProofData:   proofData,
		ProofType:   proofType,
		PublicInput: publicInput,
	}

	signedProof := &SignedExecutionProof{
		Message:      executionProof,
		ProverPubkey: make([]byte, 48), // 48 zero bytes (dummy)
		Signature:    make([]byte, 96), // 96 zero bytes (dummy)
	}

	return signedProof, nil
}
