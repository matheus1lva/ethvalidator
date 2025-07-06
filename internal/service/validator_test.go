package service

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/matheus/eth-validator-api/internal/domain"
	pkgerrors "github.com/matheus/eth-validator-api/pkg/errors"
	"github.com/matheus/eth-validator-api/pkg/ethereum"
	"github.com/matheus/eth-validator-api/pkg/logger"
)

type mockEthClient struct {
	mock.Mock
}

func (m *mockEthClient) GetBlockBySlot(ctx context.Context, slot uint64) (*ethereum.BeaconBlock, error) {
	args := m.Called(ctx, slot)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ethereum.BeaconBlock), args.Error(1)
}

func (m *mockEthClient) GetSyncCommittee(ctx context.Context, slot uint64) ([]string, error) {
	args := m.Called(ctx, slot)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockEthClient) GetCurrentSlot(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *mockEthClient) GetBlockRewards(ctx context.Context, slot uint64) (*ethereum.BlockRewards, error) {
	args := m.Called(ctx, slot)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ethereum.BlockRewards), args.Error(1)
}

func (m *mockEthClient) GetProposerDuties(ctx context.Context, epoch uint64) ([]ethereum.ProposerDuty, error) {
	args := m.Called(ctx, epoch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ethereum.ProposerDuty), args.Error(1)
}

type mockCache struct {
	mock.Mock
}

func (m *mockCache) Get(key string) (interface{}, bool) {
	args := m.Called(key)
	return args.Get(0), args.Bool(1)
}

func (m *mockCache) Set(key string, value interface{}) {
	m.Called(key, value)
}

func TestValidatorService_GetBlockReward(t *testing.T) {
	tests := []struct {
		name           string
		slot           uint64
		setupMocks     func(*mockEthClient, *mockCache)
		expectedReward *domain.BlockReward
		expectedError  error
	}{
		{
			name: "successful MEV block",
			slot: 12345,
			setupMocks: func(client *mockEthClient, cache *mockCache) {
				cache.On("Get", "block_reward:12345").Return(nil, false)
				client.On("GetCurrentSlot", mock.Anything).Return(uint64(20000), nil)
				client.On("GetBlockBySlot", mock.Anything, uint64(12345)).Return(&ethereum.BeaconBlock{
					Data: ethereum.BeaconBlockData{
						Message: ethereum.BlockMessage{
							Body: ethereum.BlockBody{
								ExecutionPayload: &ethereum.ExecutionPayload{
									FeeRecipient: "0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5",
									Transactions: []string{"0xa22cb465..."},
								},
							},
						},
					},
				}, nil)
				client.On("GetBlockRewards", mock.Anything, uint64(12345)).Return(&ethereum.BlockRewards{
					Total: "1000000000000000000",
				}, nil)
				cache.On("Set", "block_reward:12345", mock.Anything)
			},
			expectedReward: &domain.BlockReward{
				Status: "mev",
				Reward: big.NewInt(1000000000000000000),
			},
		},
		{
			name: "successful vanilla block",
			slot: 12346,
			setupMocks: func(client *mockEthClient, cache *mockCache) {
				cache.On("Get", "block_reward:12346").Return(nil, false)
				client.On("GetCurrentSlot", mock.Anything).Return(uint64(20000), nil)
				client.On("GetBlockBySlot", mock.Anything, uint64(12346)).Return(&ethereum.BeaconBlock{
					Data: ethereum.BeaconBlockData{
						Message: ethereum.BlockMessage{
							Body: ethereum.BlockBody{
								ExecutionPayload: &ethereum.ExecutionPayload{
									FeeRecipient: "0x1234567890abcdef",
									Transactions: []string{},
								},
							},
						},
					},
				}, nil)
				client.On("GetBlockRewards", mock.Anything, uint64(12346)).Return(&ethereum.BlockRewards{
					Total: "500000000000000000",
				}, nil)
				cache.On("Set", "block_reward:12346", mock.Anything)
			},
			expectedReward: &domain.BlockReward{
				Status: "vanilla",
				Reward: big.NewInt(500000000000000000),
			},
		},
		{
			name: "cached result",
			slot: 12347,
			setupMocks: func(client *mockEthClient, cache *mockCache) {
				cachedReward := &domain.BlockReward{
					Status: "mev",
					Reward: big.NewInt(2000000000000000000),
				}
				cache.On("Get", "block_reward:12347").Return(cachedReward, true)
			},
			expectedReward: &domain.BlockReward{
				Status: "mev",
				Reward: big.NewInt(2000000000000000000),
			},
		},
		{
			name: "future slot error",
			slot: 30000,
			setupMocks: func(client *mockEthClient, cache *mockCache) {
				cache.On("Get", "block_reward:30000").Return(nil, false)
				client.On("GetCurrentSlot", mock.Anything).Return(uint64(20000), nil)
			},
			expectedError: pkgerrors.ErrFutureSlot,
		},
		{
			name: "slot not found",
			slot: 12348,
			setupMocks: func(client *mockEthClient, cache *mockCache) {
				cache.On("Get", "block_reward:12348").Return(nil, false)
				client.On("GetCurrentSlot", mock.Anything).Return(uint64(20000), nil)
				client.On("GetBlockBySlot", mock.Anything, uint64(12348)).Return(nil, pkgerrors.ErrSlotNotFound)
			},
			expectedError: pkgerrors.ErrSlotNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := new(mockEthClient)
			cache := new(mockCache)
			log := logger.New("error")

			tt.setupMocks(client, cache)

			service, err := NewValidatorService(client, log, cache)
			assert.NoError(t, err)

			result, err := service.GetBlockReward(context.Background(), tt.slot)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedReward.Status, result.Status)
				assert.Equal(t, 0, tt.expectedReward.Reward.Cmp(result.Reward))
			}

			client.AssertExpectations(t)
			cache.AssertExpectations(t)
		})
	}
}

func TestValidatorService_GetSyncCommitteeDuties(t *testing.T) {
	tests := []struct {
		name           string
		slot           uint64
		setupMocks     func(*mockEthClient, *mockCache)
		expectedDuties *domain.SyncCommitteeDuties
		expectedError  error
	}{
		{
			name: "successful sync duties",
			slot: 12345,
			setupMocks: func(client *mockEthClient, cache *mockCache) {
				cache.On("Get", "sync_duties:12345").Return(nil, false)
				client.On("GetCurrentSlot", mock.Anything).Return(uint64(20000), nil)
				client.On("GetSyncCommittee", mock.Anything, uint64(12345)).Return([]string{
					"0xvalidator1",
					"0xvalidator2",
					"0xvalidator3",
				}, nil)
				cache.On("Set", "sync_duties:12345", mock.Anything)
			},
			expectedDuties: &domain.SyncCommitteeDuties{
				Validators: []string{
					"0xvalidator1",
					"0xvalidator2",
					"0xvalidator3",
				},
			},
		},
		{
			name: "cached sync duties",
			slot: 12346,
			setupMocks: func(client *mockEthClient, cache *mockCache) {
				cachedDuties := &domain.SyncCommitteeDuties{
					Validators: []string{"0xcached1", "0xcached2"},
				}
				cache.On("Get", "sync_duties:12346").Return(cachedDuties, true)
			},
			expectedDuties: &domain.SyncCommitteeDuties{
				Validators: []string{"0xcached1", "0xcached2"},
			},
		},
		{
			name: "slot too far in future",
			slot: 1000000,
			setupMocks: func(client *mockEthClient, cache *mockCache) {
				cache.On("Get", "sync_duties:1000000").Return(nil, false)
				client.On("GetCurrentSlot", mock.Anything).Return(uint64(20000), nil)
			},
			expectedError: pkgerrors.ErrSlotTooFarInFuture,
		},
		{
			name: "slot not found",
			slot: 12347,
			setupMocks: func(client *mockEthClient, cache *mockCache) {
				cache.On("Get", "sync_duties:12347").Return(nil, false)
				client.On("GetCurrentSlot", mock.Anything).Return(uint64(20000), nil)
				client.On("GetSyncCommittee", mock.Anything, uint64(12347)).Return(nil, pkgerrors.ErrSlotNotFound)
			},
			expectedError: pkgerrors.ErrSlotNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := new(mockEthClient)
			cache := new(mockCache)
			log := logger.New("error")

			tt.setupMocks(client, cache)

			service, err := NewValidatorService(client, log, cache)
			assert.NoError(t, err)

			result, err := service.GetSyncCommitteeDuties(context.Background(), tt.slot)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedDuties.Validators, result.Validators)
			}

			client.AssertExpectations(t)
			cache.AssertExpectations(t)
		})
	}
}

func TestValidatorService_Constructor(t *testing.T) {
	log := logger.New("error")
	client := new(mockEthClient)
	cache := new(mockCache)

	t.Run("nil client", func(t *testing.T) {
		_, err := NewValidatorService(nil, log, cache)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ethereum client is required")
	})

	t.Run("nil logger", func(t *testing.T) {
		_, err := NewValidatorService(client, nil, cache)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "logger is required")
	})

	t.Run("valid construction", func(t *testing.T) {
		service, err := NewValidatorService(client, log, cache)
		assert.NoError(t, err)
		assert.NotNil(t, service)
	})
}
