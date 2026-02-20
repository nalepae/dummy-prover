package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ValidatorClient is an HTTP client for interacting with a validator client.
type ValidatorClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewValidatorClient creates a new validator client.
func NewValidatorClient(baseURL string) *ValidatorClient {
	// Ensure no trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &ValidatorClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

// SignExecutionProof sends an execution proof to the validator client for signing.
func (c *ValidatorClient) SignExecutionProof(ctx context.Context, proof *ExecutionProof) (*SignedExecutionProof, error) {
	url := c.baseURL + "/eth/v2/validator/execution_proofs"

	reqBody := &ExecutionProofRequest{
		Data: proof,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	var signedResp SignedExecutionProofResponse
	if err := json.Unmarshal(respBody, &signedResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if signedResp.Data == nil {
		return nil, fmt.Errorf("response data is nil")
	}

	return signedResp.Data, nil
}
