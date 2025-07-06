package ethereum

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/matheus/eth-validator-api/internal/config"
	"github.com/matheus/eth-validator-api/pkg/errors"
)

type Client interface {
	GetBlockBySlot(ctx context.Context, slot uint64) (*BeaconBlock, error)
	GetSyncCommittee(ctx context.Context, slot uint64) ([]string, error)
	GetCurrentSlot(ctx context.Context) (uint64, error)
	GetBlockRewards(ctx context.Context, slot uint64) (*BlockRewards, error)
	GetProposerDuties(ctx context.Context, epoch uint64) ([]ProposerDuty, error)
}

type client struct {
	httpClient     *http.Client
	rpcEndpoint    string
	requestCounter uint64
	config         *config.RequestConfig
}

func NewClient(cfg *config.Config) (Client, error) {
	return &client{
		httpClient: &http.Client{
			Timeout: cfg.Request.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		rpcEndpoint: cfg.Ethereum.RPCEndpoint,
		config:      &cfg.Request,
	}, nil
}

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      uint64      `json:"id"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
	ID      uint64          `json:"id"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type BeaconBlock struct {
	Version string          `json:"version"`
	Data    BeaconBlockData `json:"data"`
}

type BeaconBlockData struct {
	Message   BlockMessage `json:"message"`
	Signature string       `json:"signature"`
}

type BlockMessage struct {
	Slot          string    `json:"slot"`
	ProposerIndex string    `json:"proposer_index"`
	ParentRoot    string    `json:"parent_root"`
	StateRoot     string    `json:"state_root"`
	Body          BlockBody `json:"body"`
}

type BlockBody struct {
	ExecutionPayload *ExecutionPayload `json:"execution_payload,omitempty"`
	SyncAggregate    *SyncAggregate    `json:"sync_aggregate,omitempty"`
}

type ExecutionPayload struct {
	FeeRecipient  string   `json:"fee_recipient"`
	BlockHash     string   `json:"block_hash"`
	Transactions  []string `json:"transactions"`
	BaseFeePerGas string   `json:"base_fee_per_gas"`
	GasUsed       string   `json:"gas_used"`
	BlockNumber   string   `json:"block_number"`
}

type SyncAggregate struct {
	SyncCommitteeBits      string `json:"sync_committee_bits"`
	SyncCommitteeSignature string `json:"sync_committee_signature"`
}

type BlockRewards struct {
	ProposerIndex     string `json:"proposer_index"`
	Total             string `json:"total"`
	Attestations      string `json:"attestations"`
	SyncAggregate     string `json:"sync_aggregate"`
	ProposerSlashings string `json:"proposer_slashings"`
	AttesterSlashings string `json:"attester_slashings"`
}

type SyncCommitteeResponse struct {
	Data SyncCommitteeData `json:"data"`
}

type SyncCommitteeData struct {
	Validators []string `json:"validators"`
}

type ProposerDuty struct {
	Pubkey         string `json:"pubkey"`
	ValidatorIndex string `json:"validator_index"`
	Slot           string `json:"slot"`
}

type ProposerDutiesResponse struct {
	Data []ProposerDuty `json:"data"`
}

type GenesisResponse struct {
	Data GenesisData `json:"data"`
}

type GenesisData struct {
	GenesisTime string `json:"genesis_time"`
}

type HeaderResponse struct {
	Data HeaderData `json:"data"`
}

type HeaderData struct {
	Header HeaderInfo `json:"header"`
}

type HeaderInfo struct {
	Message HeaderMessage `json:"message"`
}

type HeaderMessage struct {
	Slot string `json:"slot"`
}

func (c *client) doRequest(ctx context.Context, method string, params interface{}, result interface{}) error {
	id := atomic.AddUint64(&c.requestCounter, 1)

	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.rpcEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return errors.RPCError{
			Code:    rpcResp.Error.Code,
			Message: rpcResp.Error.Message,
			Data:    rpcResp.Error.Data,
		}
	}

	if result != nil && len(rpcResp.Result) > 0 {
		if err := json.Unmarshal(rpcResp.Result, result); err != nil {
			return fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return nil
}

func (c *client) doBeaconRequest(ctx context.Context, endpoint string, result interface{}) error {
	url := fmt.Sprintf("%s/eth/v1/beacon/%s", c.rpcEndpoint, endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errors.ErrSlotNotFound
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *client) GetBlockBySlot(ctx context.Context, slot uint64) (*BeaconBlock, error) {
	var block BeaconBlock
	endpoint := fmt.Sprintf("blocks/%d", slot)

	if err := c.doBeaconRequest(ctx, endpoint, &block); err != nil {
		return nil, err
	}

	return &block, nil
}

func (c *client) GetSyncCommittee(ctx context.Context, slot uint64) ([]string, error) {
	epoch := slot / 32
	syncCommitteePeriod := epoch / 256

	stateID := fmt.Sprintf("%d", syncCommitteePeriod*256*32)
	endpoint := fmt.Sprintf("states/%s/sync_committees", stateID)

	var resp SyncCommitteeResponse
	if err := c.doBeaconRequest(ctx, endpoint, &resp); err != nil {
		return nil, err
	}

	return resp.Data.Validators, nil
}

func (c *client) GetCurrentSlot(ctx context.Context) (uint64, error) {
	var genesis GenesisResponse
	if err := c.doBeaconRequest(ctx, "genesis", &genesis); err != nil {
		return 0, err
	}

	genesisTime, err := parseUint64(genesis.Data.GenesisTime)
	if err != nil {
		return 0, fmt.Errorf("failed to parse genesis time: %w", err)
	}

	currentTime := uint64(time.Now().Unix())
	if currentTime < genesisTime {
		return 0, fmt.Errorf("current time is before genesis")
	}

	return (currentTime - genesisTime) / 12, nil
}

func (c *client) GetBlockRewards(ctx context.Context, slot uint64) (*BlockRewards, error) {
	endpoint := fmt.Sprintf("rewards/blocks/%d", slot)

	type rewardsResponse struct {
		Data BlockRewards `json:"data"`
	}

	var resp rewardsResponse
	if err := c.doBeaconRequest(ctx, endpoint, &resp); err != nil {
		return nil, err
	}

	return &resp.Data, nil
}
func (c *client) GetProposerDuties(ctx context.Context, epoch uint64) ([]ProposerDuty, error) {
	endpoint := fmt.Sprintf("duties/proposer/%d", epoch)

	var resp ProposerDutiesResponse
	if err := c.doBeaconRequest(ctx, endpoint, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func parseUint64(s string) (uint64, error) {
	var n uint64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
