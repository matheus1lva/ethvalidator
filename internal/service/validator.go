package service

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/matheus/eth-validator-api/internal/domain"
	"github.com/matheus/eth-validator-api/pkg/errors"
	"github.com/matheus/eth-validator-api/pkg/ethereum"
	"github.com/matheus/eth-validator-api/pkg/logger"
)

type ValidatorService interface {
	GetBlockReward(ctx context.Context, slot uint64) (*domain.BlockReward, error)
	GetSyncCommitteeDuties(ctx context.Context, slot uint64) (*domain.SyncCommitteeDuties, error)
}

type validatorService struct {
	ethClient ethereum.Client
	logger    logger.Logger
	cache     Cache
}

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}

func NewValidatorService(ethClient ethereum.Client, logger logger.Logger, cache Cache) (ValidatorService, error) {
	if ethClient == nil {
		return nil, fmt.Errorf("ethereum client is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &validatorService{
		ethClient: ethClient,
		logger:    logger,
		cache:     cache,
	}, nil
}

func (s *validatorService) GetBlockReward(ctx context.Context, slot uint64) (*domain.BlockReward, error) {
	s.logger.Info().Uint64("slot", slot).Msg("getting block reward")

	cacheKey := fmt.Sprintf("block_reward:%d", slot)
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.logger.Debug().Uint64("slot", slot).Msg("returning cached block reward")
			return cached.(*domain.BlockReward), nil
		}
	}

	currentSlot, err := s.ethClient.GetCurrentSlot(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get current slot")
		return nil, fmt.Errorf("failed to get current slot: %w", err)
	}

	if slot > currentSlot {
		s.logger.Warn().Uint64("slot", slot).Uint64("current_slot", currentSlot).Msg("requested future slot")
		return nil, errors.ErrFutureSlot
	}

	block, err := s.ethClient.GetBlockBySlot(ctx, slot)
	if err != nil {
		if errors.IsNotFound(err) {
			s.logger.Info().Uint64("slot", slot).Msg("slot not found - likely missed")
			return nil, errors.ErrSlotNotFound
		}
		s.logger.Error().Err(err).Uint64("slot", slot).Msg("failed to get block")
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	rewards, err := s.ethClient.GetBlockRewards(ctx, slot)
	if err != nil {
		s.logger.Error().Err(err).Uint64("slot", slot).Msg("failed to get block rewards")
		return nil, fmt.Errorf("failed to get block rewards: %w", err)
	}

	status := s.determineBlockStatus(block)

	totalReward, err := s.parseReward(rewards.Total)
	if err != nil {
		s.logger.Error().Err(err).Str("reward", rewards.Total).Msg("failed to parse reward")
		return nil, fmt.Errorf("failed to parse reward: %w", err)
	}

	result := &domain.BlockReward{
		Status: status,
		Reward: totalReward,
	}

	if s.cache != nil {
		s.cache.Set(cacheKey, result)
	}

	s.logger.Info().
		Uint64("slot", slot).
		Str("status", status).
		Str("reward", totalReward.String()).
		Msg("block reward retrieved")

	return result, nil
}

func (s *validatorService) GetSyncCommitteeDuties(ctx context.Context, slot uint64) (*domain.SyncCommitteeDuties, error) {
	s.logger.Info().Uint64("slot", slot).Msg("getting sync committee duties")

	cacheKey := fmt.Sprintf("sync_duties:%d", slot)
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.logger.Debug().Uint64("slot", slot).Msg("returning cached sync duties")
			return cached.(*domain.SyncCommitteeDuties), nil
		}
	}

	currentSlot, err := s.ethClient.GetCurrentSlot(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get current slot")
		return nil, fmt.Errorf("failed to get current slot: %w", err)
	}

	if slot > currentSlot+32*256 {
		s.logger.Warn().Uint64("slot", slot).Uint64("current_slot", currentSlot).Msg("slot too far in future")
		return nil, errors.ErrSlotTooFarInFuture
	}

	validators, err := s.ethClient.GetSyncCommittee(ctx, slot)
	if err != nil {
		if errors.IsNotFound(err) {
			s.logger.Info().Uint64("slot", slot).Msg("slot not found")
			return nil, errors.ErrSlotNotFound
		}
		s.logger.Error().Err(err).Uint64("slot", slot).Msg("failed to get sync committee")
		return nil, fmt.Errorf("failed to get sync committee: %w", err)
	}

	result := &domain.SyncCommitteeDuties{
		Validators: validators,
	}

	if s.cache != nil {
		s.cache.Set(cacheKey, result)
	}

	s.logger.Info().
		Uint64("slot", slot).
		Int("validator_count", len(validators)).
		Msg("sync committee duties retrieved")

	return result, nil
}

func (s *validatorService) determineBlockStatus(block *ethereum.BeaconBlock) string {
	if block.Data.Message.Body.ExecutionPayload == nil {
		return "vanilla"
	}

	payload := block.Data.Message.Body.ExecutionPayload

	if len(payload.Transactions) == 0 {
		return "vanilla"
	}

	for _, tx := range payload.Transactions {
		if s.isMEVTransaction(tx) {
			return "mev"
		}
	}

	feeRecipient := strings.ToLower(payload.FeeRecipient)
	knownMEVRelays := []string{
		"0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5",
		"0x388c818ca8b9251b393131c08a736a67ccb19297",
		"0x8b5d7a6055e54e36e8a6e2a128c5d0f38f4e5e83",
	}

	for _, relay := range knownMEVRelays {
		if feeRecipient == relay {
			return "mev"
		}
	}

	return "vanilla"
}

func (s *validatorService) isMEVTransaction(txHex string) bool {
	if len(txHex) < 10 {
		return false
	}

	mevPatterns := []string{
		"0xa22cb465",
		"0x095ea7b3",
		"0x23b872dd",
	}

	for _, pattern := range mevPatterns {
		if strings.HasPrefix(txHex, pattern) {
			return true
		}
	}

	return false
}

func (s *validatorService) parseReward(rewardStr string) (*big.Int, error) {
	reward, ok := new(big.Int).SetString(rewardStr, 10)
	if !ok {
		return nil, fmt.Errorf("invalid reward format: %s", rewardStr)
	}
	return reward, nil
}

func slotToEpoch(slot uint64) uint64 {
	return slot / 32
}

func epochToSyncCommitteePeriod(epoch uint64) uint64 {
	return epoch / 256
}

func syncCommitteePeriodToSlot(period uint64) uint64 {
	return period * 256 * 32
}

func parseSlot(slotStr string) (uint64, error) {
	slot, err := strconv.ParseUint(slotStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid slot format: %w", err)
	}
	return slot, nil
}
