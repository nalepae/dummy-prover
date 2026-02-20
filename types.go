package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const blobCommitmentVersionKZG uint8 = 0x01

type (
	BlindedBlockBeaconAPIResponse struct {
		Data *SignedBlindedBeaconBlock `json:"data"`
	}

	NewPayloadRequestHeader struct {
		ExecutionPayloadHeader *ExecutionPayloadHeader
		VersionedHashes        [][]byte `ssz-max:"4096" ssz-size:"?,32"`
		ParentBeaconBlockRoot  []byte   `ssz-size:"32"`
		ExecutionRequests      *ExecutionRequests
	}

	SignedBlindedBeaconBlock struct {
		Message *BlindedBeaconBlock `json:"message"`
	}

	BlindedBeaconBlock struct {
		Slot       Slot                    `json:"slot"`
		ParentRoot []byte                  `json:"parent_root" ssz-size:"32"`
		Body       *BlindedBeaconBlockBody `json:"body"`
	}

	BlindedBeaconBlockBody struct {
		ExecutionPayloadHeader *ExecutionPayloadHeader `json:"execution_payload_header"`
		BlobKzgCommitments     [][]byte                `json:"blob_kzg_commitments,omitempty" ssz-max:"4096" ssz-size:"?,48"`
		ExecutionRequests      *ExecutionRequests      `json:"execution_requests"`
	}

	ExecutionPayloadHeader struct {
		ParentHash       []byte `json:"parent_hash,omitempty" ssz-size:"32"`
		FeeRecipient     []byte `json:"fee_recipient,omitempty" ssz-size:"20"`
		StateRoot        []byte `json:"state_root,omitempty" ssz-size:"32"`
		ReceiptsRoot     []byte `json:"receipts_root,omitempty" ssz-size:"32"`
		LogsBloom        []byte `json:"logs_bloom,omitempty" ssz-size:"256"`
		PrevRandao       []byte `json:"prev_randao,omitempty" ssz-size:"32"`
		BlockNumber      uint64 `json:"block_number,omitempty"`
		GasLimit         uint64 `json:"gas_limit,omitempty"`
		GasUsed          uint64 `json:"gas_used,omitempty"`
		Timestamp        uint64 `json:"timestamp,omitempty"`
		ExtraData        []byte `json:"extra_data,omitempty" ssz-max:"32"`
		BaseFeePerGas    []byte `json:"base_fee_per_gas,omitempty" ssz-size:"32"`
		BlockHash        []byte `json:"block_hash,omitempty" ssz-size:"32"`
		TransactionsRoot []byte `json:"transactions_root,omitempty" ssz-size:"32"`
		WithdrawalsRoot  []byte `json:"withdrawals_root,omitempty" ssz-size:"32"`
		BlobGasUsed      uint64 `json:"blob_gas_used,omitempty"`
		ExcessBlobGas    uint64 `json:"excess_blob_gas,omitempty"`
	}

	ExecutionRequests struct {
		Deposits       []*Deposit       `json:"deposits,omitempty" ssz-max:"8192"`
		Withdrawals    []*Withdrawal    `json:"withdrawals,omitempty" ssz-max:"16"`
		Consolidations []*Consolidation `json:"consolidations,omitempty" ssz-max:"2"`
	}

	Deposit struct {
		Pubkey                []byte `json:"pubkey,omitempty" ssz-size:"48"`
		WithdrawalCredentials []byte `json:"withdrawal_credentials,omitempty" ssz-size:"32"`
		Amount                uint64 `json:"amount,omitempty"`
		Signature             []byte `json:"signature,omitempty" ssz-size:"96"`
		Index                 uint64 `json:"index,omitempty"`
	}

	Withdrawal struct {
		SourceAddress   []byte `json:"source_address,omitempty" ssz-size:"20"`
		ValidatorPubkey []byte `json:"validator_pubkey,omitempty" ssz-size:"48"`
		Amount          uint64 `json:"amount,omitempty"`
	}

	Consolidation struct {
		SourceAddress []byte `json:"source_address,omitempty" ssz-size:"20"`
		SourcePubkey  []byte `json:"source_pubkey,omitempty" ssz-size:"48"`
		TargetPubkey  []byte `json:"target_pubkey,omitempty" ssz-size:"48"`
	}

	ExecutionProof struct {
		ProofData   []byte       `json:"proof_data"`
		ProofType   ProofType    `json:"proof_type"`
		PublicInput *PublicInput `json:"public_input"`
	}

	SignedExecutionProof struct {
		Message        *ExecutionProof `json:"message"`
		ValidatorIndex uint64          `json:"validator_index"`
		Signature      []byte          `json:"signature"` // 96 bytes
	}

	PublicInput struct {
		NewPayloadRequestRoot []byte `json:"new_payload_request_root,omitempty"`
	}

	BlockEventData struct {
		Slot  Slot `json:"slot"`
		Block Root `json:"block"`
	}

	ProofType uint8
	Slot      uint64
	Root      [32]byte
	Hash      [32]byte
)

// HashTreeRoot computes a placeholder block root.
// TODO: Implement proper SSZ hash tree root computation.
func (b *BlindedBeaconBlock) HashTreeRoot() [32]byte {
	// For now, use a hash of the block hash as a placeholder
	return sha256.Sum256(b.Body.ExecutionPayloadHeader.BlockHash)
}

func kzgCommitmentsToVersionedHashes(blindedBody *BlindedBeaconBlockBody) [][]byte {
	commitments := blindedBody.BlobKzgCommitments

	versionedHashes := make([][]byte, 0, len(commitments))
	for _, commitment := range commitments {
		versionedHash := kzgCommitmentsToVersionedHash(commitment)
		versionedHashes = append(versionedHashes, versionedHash[:])
	}

	return versionedHashes
}

func kzgCommitmentsToVersionedHash(commitment []byte) common.Hash {
	versionedHash := sha256.Sum256(commitment)
	versionedHash[0] = blobCommitmentVersionKZG
	return versionedHash
}

// UnmarshalJSON parses a quoted decimal string into a Slot.
func (s *Slot) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("unmarshal slot string: %w", err)
	}

	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return fmt.Errorf("parse slot: %w", err)
	}

	*s = Slot(val)
	return nil
}

// UnmarshalJSON parses a hex string with 0x prefix into a Root.
func (r *Root) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("unmarshal root string: %w", err)
	}

	str = strings.TrimPrefix(str, "0x")
	decoded, err := hex.DecodeString(str)
	if err != nil {
		return fmt.Errorf("decode root hex: %w", err)
	}

	if len(decoded) != 32 {
		return fmt.Errorf("invalid root length: got %d, want 32", len(decoded))
	}

	copy(r[:], decoded)
	return nil
}

// Helper to decode hex string to bytes
func decodeHexBytes(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	return hex.DecodeString(s)
}

// Helper to parse quoted uint64
func parseQuotedUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

// UnmarshalJSON parses beacon API JSON format into BlindedBeaconBlock.
func (b *BlindedBeaconBlock) UnmarshalJSON(data []byte) error {
	type jsonBlindedBeaconBlock struct {
		Slot       string                  `json:"slot"`
		ParentRoot string                  `json:"parent_root"`
		Body       *BlindedBeaconBlockBody `json:"body"`
	}

	var jb jsonBlindedBeaconBlock
	if err := json.Unmarshal(data, &jb); err != nil {
		return err
	}

	slot, err := parseQuotedUint64(jb.Slot)
	if err != nil {
		return fmt.Errorf("parse slot: %w", err)
	}
	b.Slot = Slot(slot)

	parentRoot, err := decodeHexBytes(jb.ParentRoot)
	if err != nil {
		return fmt.Errorf("decode parent_root: %w", err)
	}
	b.ParentRoot = parentRoot
	b.Body = jb.Body

	return nil
}

// UnmarshalJSON parses beacon API JSON format into ExecutionPayloadHeader.
func (e *ExecutionPayloadHeader) UnmarshalJSON(data []byte) error {
	type jsonExecutionPayloadHeader struct {
		ParentHash       string `json:"parent_hash"`
		FeeRecipient     string `json:"fee_recipient"`
		StateRoot        string `json:"state_root"`
		ReceiptsRoot     string `json:"receipts_root"`
		LogsBloom        string `json:"logs_bloom"`
		PrevRandao       string `json:"prev_randao"`
		BlockNumber      string `json:"block_number"`
		GasLimit         string `json:"gas_limit"`
		GasUsed          string `json:"gas_used"`
		Timestamp        string `json:"timestamp"`
		ExtraData        string `json:"extra_data"`
		BaseFeePerGas    string `json:"base_fee_per_gas"`
		BlockHash        string `json:"block_hash"`
		TransactionsRoot string `json:"transactions_root"`
		WithdrawalsRoot  string `json:"withdrawals_root"`
		BlobGasUsed      string `json:"blob_gas_used"`
		ExcessBlobGas    string `json:"excess_blob_gas"`
	}

	var je jsonExecutionPayloadHeader
	if err := json.Unmarshal(data, &je); err != nil {
		return err
	}

	var err error

	if e.ParentHash, err = decodeHexBytes(je.ParentHash); err != nil {
		return fmt.Errorf("decode parent_hash: %w", err)
	}
	if e.FeeRecipient, err = decodeHexBytes(je.FeeRecipient); err != nil {
		return fmt.Errorf("decode fee_recipient: %w", err)
	}
	if e.StateRoot, err = decodeHexBytes(je.StateRoot); err != nil {
		return fmt.Errorf("decode state_root: %w", err)
	}
	if e.ReceiptsRoot, err = decodeHexBytes(je.ReceiptsRoot); err != nil {
		return fmt.Errorf("decode receipts_root: %w", err)
	}
	if e.LogsBloom, err = decodeHexBytes(je.LogsBloom); err != nil {
		return fmt.Errorf("decode logs_bloom: %w", err)
	}
	if e.PrevRandao, err = decodeHexBytes(je.PrevRandao); err != nil {
		return fmt.Errorf("decode prev_randao: %w", err)
	}
	if e.BlockNumber, err = parseQuotedUint64(je.BlockNumber); err != nil {
		return fmt.Errorf("parse block_number: %w", err)
	}
	if e.GasLimit, err = parseQuotedUint64(je.GasLimit); err != nil {
		return fmt.Errorf("parse gas_limit: %w", err)
	}
	if e.GasUsed, err = parseQuotedUint64(je.GasUsed); err != nil {
		return fmt.Errorf("parse gas_used: %w", err)
	}
	if e.Timestamp, err = parseQuotedUint64(je.Timestamp); err != nil {
		return fmt.Errorf("parse timestamp: %w", err)
	}
	if e.ExtraData, err = decodeHexBytes(je.ExtraData); err != nil {
		return fmt.Errorf("decode extra_data: %w", err)
	}
	// BaseFeePerGas is a uint256 encoded as decimal string, we need to convert to 32-byte big-endian
	baseFee, err := parseQuotedUint64(je.BaseFeePerGas)
	if err != nil {
		return fmt.Errorf("parse base_fee_per_gas: %w", err)
	}
	e.BaseFeePerGas = make([]byte, 32)
	// Store as little-endian (SSZ uses little-endian for uint256)
	for i := 0; i < 8; i++ {
		e.BaseFeePerGas[i] = byte(baseFee >> (8 * i))
	}
	if e.BlockHash, err = decodeHexBytes(je.BlockHash); err != nil {
		return fmt.Errorf("decode block_hash: %w", err)
	}
	if e.TransactionsRoot, err = decodeHexBytes(je.TransactionsRoot); err != nil {
		return fmt.Errorf("decode transactions_root: %w", err)
	}
	if e.WithdrawalsRoot, err = decodeHexBytes(je.WithdrawalsRoot); err != nil {
		return fmt.Errorf("decode withdrawals_root: %w", err)
	}
	if e.BlobGasUsed, err = parseQuotedUint64(je.BlobGasUsed); err != nil {
		return fmt.Errorf("parse blob_gas_used: %w", err)
	}
	if e.ExcessBlobGas, err = parseQuotedUint64(je.ExcessBlobGas); err != nil {
		return fmt.Errorf("parse excess_blob_gas: %w", err)
	}

	return nil
}

// UnmarshalJSON parses beacon API JSON format into BlindedBeaconBlockBody.
func (b *BlindedBeaconBlockBody) UnmarshalJSON(data []byte) error {
	type jsonBlindedBeaconBlockBody struct {
		ExecutionPayloadHeader *ExecutionPayloadHeader `json:"execution_payload_header"`
		BlobKzgCommitments     []string                `json:"blob_kzg_commitments"`
		ExecutionRequests      *ExecutionRequests      `json:"execution_requests"`
	}

	var jb jsonBlindedBeaconBlockBody
	if err := json.Unmarshal(data, &jb); err != nil {
		return err
	}

	b.ExecutionPayloadHeader = jb.ExecutionPayloadHeader
	b.ExecutionRequests = jb.ExecutionRequests

	b.BlobKzgCommitments = make([][]byte, len(jb.BlobKzgCommitments))
	for i, c := range jb.BlobKzgCommitments {
		decoded, err := decodeHexBytes(c)
		if err != nil {
			return fmt.Errorf("decode blob_kzg_commitments[%d]: %w", i, err)
		}
		b.BlobKzgCommitments[i] = decoded
	}

	return nil
}
