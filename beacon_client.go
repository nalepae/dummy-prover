package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 12 * time.Second

	blockEvent = "block"
)

// BeaconClient is an HTTP client for interacting with a beacon node.
type BeaconClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewBeaconClient creates a new beacon node client.
func NewBeaconClient(baseURL string) *BeaconClient {
	// Ensure no trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &BeaconClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// subscribeToBlockGossip subscribes to block SSE events.
// Returns a channel that receives BlockEvent and an error channel.
func (c *BeaconClient) subscribeToBlockGossip(ctx context.Context) (<-chan BlockEventData, <-chan error) {
	events := make(chan BlockEventData)
	errs := make(chan error, 1)

	go func() {
		var (
			eventType string
			data      string
		)

		defer close(events)
		defer close(errs)

		url := c.baseURL + "/eth/v1/events?topics=" + blockEvent

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			errs <- fmt.Errorf("failed to create request: %w", err)
			return
		}
		req.Header.Set("Accept", "text/event-stream")

		// Use a client without timeout for SSE
		sseClient := &http.Client{}
		resp, err := sseClient.Do(req)
		if err != nil {
			errs <- fmt.Errorf("failed to connect to SSE stream: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errs <- fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
			return
		}

		logger.Info("Connected to SSE stream", "event", blockEvent, "url", url)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if line != "" {
				switch {
				case strings.HasPrefix(line, "event:"):
					eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				case strings.HasPrefix(line, "data:"):
					data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				}
				continue
			}

			// End of event, process if we have data
			if eventType != blockEvent || data == "" {
				eventType = ""
				data = ""
				continue
			}

			var eventData BlockEventData
			if err := json.Unmarshal([]byte(data), &eventData); err != nil {
				logger.Warn("Failed to parse event", "event", blockEvent, "error", err, "data", data)
				eventType = ""
				data = ""
				continue
			}

			select {
			case events <- eventData:
			case <-ctx.Done():
				return
			}

			eventType = ""
			data = ""
		}

		if err := scanner.Err(); err != nil {
			if ctx.Err() == nil {
				errs <- fmt.Errorf("SSE stream error: %w", err)
			}
		}
	}()

	return events, errs
}

// GetSignedBlindedBeaconBlock fetches a signed blinded block by ID (root or slot).
func (c *BeaconClient) GetSignedBlindedBeaconBlock(ctx context.Context, blockID string) (*SignedBlindedBeaconBlock, error) {
	url := c.baseURL + "/eth/v1/beacon/blinded_blocks/" + blockID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get blinded block: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("block not found: %s", blockID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse the JSON response to extract block_root, slot, and block_hash
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}

	response := new(BlindedBlockBeaconAPIResponse)

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return response.Data, nil
}

// SubmitSignedExecutionProof submits a signed execution proof to the beacon node pool.
func (c *BeaconClient) SubmitSignedExecutionProof(ctx context.Context, proof *SignedExecutionProof) error {
	url := c.baseURL + "/eth/v1/prover/execution_proofs"

	body, err := json.Marshal(proof)
	if err != nil {
		return fmt.Errorf("failed to marshal proof: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to submit proof: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
