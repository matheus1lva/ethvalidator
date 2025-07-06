package domain

import (
	"encoding/json"
	"math/big"
)

type BlockReward struct {
	Status string   `json:"status"`
	Reward *big.Int `json:"-"`
}

func (b BlockReward) MarshalJSON() ([]byte, error) {
	type Alias BlockReward
	return json.Marshal(&struct {
		*Alias
		Reward string `json:"reward"`
	}{
		Alias:  (*Alias)(&b),
		Reward: b.Reward.String(),
	})
}

type SyncCommitteeDuties struct {
	Validators []string `json:"validators"`
}

type Block struct {
	Slot             uint64            `json:"slot"`
	ProposerIndex    uint64            `json:"proposer_index"`
	ParentRoot       string            `json:"parent_root"`
	StateRoot        string            `json:"state_root"`
	Body             BlockBody         `json:"body"`
	ExecutionPayload *ExecutionPayload `json:"execution_payload,omitempty"`
}

type BlockBody struct {
	RandaoReveal      string            `json:"randao_reveal"`
	Eth1Data          Eth1Data          `json:"eth1_data"`
	Graffiti          string            `json:"graffiti"`
	ProposerSlashings []interface{}     `json:"proposer_slashings"`
	AttesterSlashings []interface{}     `json:"attester_slashings"`
	Attestations      []interface{}     `json:"attestations"`
	Deposits          []interface{}     `json:"deposits"`
	VoluntaryExits    []interface{}     `json:"voluntary_exits"`
	SyncAggregate     *SyncAggregate    `json:"sync_aggregate,omitempty"`
	ExecutionPayload  *ExecutionPayload `json:"execution_payload,omitempty"`
}

type Eth1Data struct {
	DepositRoot  string `json:"deposit_root"`
	DepositCount string `json:"deposit_count"`
	BlockHash    string `json:"block_hash"`
}

type SyncAggregate struct {
	SyncCommitteeBits      string `json:"sync_committee_bits"`
	SyncCommitteeSignature string `json:"sync_committee_signature"`
}

type ExecutionPayload struct {
	ParentHash    string        `json:"parent_hash"`
	FeeRecipient  string        `json:"fee_recipient"`
	StateRoot     string        `json:"state_root"`
	ReceiptsRoot  string        `json:"receipts_root"`
	LogsBloom     string        `json:"logs_bloom"`
	PrevRandao    string        `json:"prev_randao"`
	BlockNumber   string        `json:"block_number"`
	GasLimit      string        `json:"gas_limit"`
	GasUsed       string        `json:"gas_used"`
	Timestamp     string        `json:"timestamp"`
	ExtraData     string        `json:"extra_data"`
	BaseFeePerGas string        `json:"base_fee_per_gas"`
	BlockHash     string        `json:"block_hash"`
	Transactions  []string      `json:"transactions"`
	Withdrawals   []interface{} `json:"withdrawals,omitempty"`
}

type SyncCommittee struct {
	Validators          []string   `json:"validators"`
	ValidatorAggregates [][]string `json:"validator_aggregates"`
}

type BeaconState struct {
	GenesisTime                 string         `json:"genesis_time"`
	GenesisValidatorsRoot       string         `json:"genesis_validators_root"`
	Slot                        string         `json:"slot"`
	Fork                        Fork           `json:"fork"`
	LatestBlockHeader           BlockHeader    `json:"latest_block_header"`
	BlockRoots                  []string       `json:"block_roots"`
	StateRoots                  []string       `json:"state_roots"`
	HistoricalRoots             []string       `json:"historical_roots"`
	Eth1Data                    Eth1Data       `json:"eth1_data"`
	Eth1DataVotes               []Eth1Data     `json:"eth1_data_votes"`
	Eth1DepositIndex            string         `json:"eth1_deposit_index"`
	Validators                  []Validator    `json:"validators"`
	Balances                    []string       `json:"balances"`
	RandaoMixes                 []string       `json:"randao_mixes"`
	Slashings                   []string       `json:"slashings"`
	PreviousEpochParticipation  []string       `json:"previous_epoch_participation"`
	CurrentEpochParticipation   []string       `json:"current_epoch_participation"`
	JustificationBits           string         `json:"justification_bits"`
	PreviousJustifiedCheckpoint Checkpoint     `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint  Checkpoint     `json:"current_justified_checkpoint"`
	FinalizedCheckpoint         Checkpoint     `json:"finalized_checkpoint"`
	InactivityScores            []string       `json:"inactivity_scores"`
	CurrentSyncCommittee        *SyncCommittee `json:"current_sync_committee"`
	NextSyncCommittee           *SyncCommittee `json:"next_sync_committee"`
}

type Fork struct {
	PreviousVersion string `json:"previous_version"`
	CurrentVersion  string `json:"current_version"`
	Epoch           string `json:"epoch"`
}

type BlockHeader struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	BodyRoot      string `json:"body_root"`
}

type Validator struct {
	Pubkey                     string `json:"pubkey"`
	WithdrawalCredentials      string `json:"withdrawal_credentials"`
	EffectiveBalance           string `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch string `json:"activation_eligibility_epoch"`
	ActivationEpoch            string `json:"activation_epoch"`
	ExitEpoch                  string `json:"exit_epoch"`
	WithdrawableEpoch          string `json:"withdrawable_epoch"`
}

type Checkpoint struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root"`
}

type ProposerDuty struct {
	Pubkey         string `json:"pubkey"`
	ValidatorIndex string `json:"validator_index"`
	Slot           string `json:"slot"`
}

type BlockInfo struct {
	Slot                uint64 `json:"slot"`
	Epoch               uint64 `json:"epoch"`
	BlockRoot           string `json:"block_root"`
	ParentRoot          string `json:"parent_root"`
	StateRoot           string `json:"state_root"`
	ProposerIndex       uint64 `json:"proposer_index"`
	ProposerSlashings   int    `json:"proposer_slashings"`
	AttesterSlashings   int    `json:"attester_slashings"`
	Attestations        int    `json:"attestations"`
	Deposits            int    `json:"deposits"`
	VoluntaryExits      int    `json:"voluntary_exits"`
	SyncAggregate       bool   `json:"sync_aggregate"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
	Finalized           bool   `json:"finalized"`
}
