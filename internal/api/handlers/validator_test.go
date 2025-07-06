package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/matheus/eth-validator-api/internal/api/middleware"
	"github.com/matheus/eth-validator-api/internal/domain"
	pkgerrors "github.com/matheus/eth-validator-api/pkg/errors"
	"github.com/matheus/eth-validator-api/pkg/logger"
)

type mockValidatorService struct {
	mock.Mock
}

func (m *mockValidatorService) GetBlockReward(ctx context.Context, slot uint64) (*domain.BlockReward, error) {
	args := m.Called(ctx, slot)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.BlockReward), args.Error(1)
}

func (m *mockValidatorService) GetSyncCommitteeDuties(ctx context.Context, slot uint64) (*domain.SyncCommitteeDuties, error) {
	args := m.Called(ctx, slot)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SyncCommitteeDuties), args.Error(1)
}

func TestValidatorHandler_GetBlockReward(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setupMock      func(*mockValidatorService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful block reward",
			path: "/blockreward/12345",
			setupMock: func(svc *mockValidatorService) {
				svc.On("GetBlockReward", mock.Anything, uint64(12345)).Return(&domain.BlockReward{
					Status: "mev",
					Reward: big.NewInt(1000000000000000000),
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"data": map[string]interface{}{
					"status": "mev",
					"reward": "1000000000000000000",
				},
			},
		},
		{
			name: "invalid slot format",
			path: "/blockreward/invalid",
			setupMock: func(svc *mockValidatorService) {
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "invalid slot number",
			},
		},
		{
			name: "empty slot",
			path: "/blockreward/",
			setupMock: func(svc *mockValidatorService) {
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "invalid slot number",
			},
		},
		{
			name: "slot not found",
			path: "/blockreward/99999",
			setupMock: func(svc *mockValidatorService) {
				svc.On("GetBlockReward", mock.Anything, uint64(99999)).Return(nil, pkgerrors.ErrSlotNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error": "slot not found",
			},
		},
		{
			name: "future slot",
			path: "/blockreward/999999",
			setupMock: func(svc *mockValidatorService) {
				svc.On("GetBlockReward", mock.Anything, uint64(999999)).Return(nil, pkgerrors.ErrFutureSlot)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "requested slot is in the future",
			},
		},
		{
			name: "internal error",
			path: "/blockreward/12346",
			setupMock: func(svc *mockValidatorService) {
				svc.On("GetBlockReward", mock.Anything, uint64(12346)).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "internal server error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockValidatorService)
			log := logger.New("error")

			handler, err := NewValidatorHandler(svc, log)
			assert.NoError(t, err)

			tt.setupMock(svc)

			req := httptest.NewRequest("GET", tt.path, nil)
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler.GetBlockReward(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedBody["data"] != nil {
				assert.Equal(t, tt.expectedBody["data"], response["data"])
			}
			if tt.expectedBody["error"] != nil {
				assert.Equal(t, tt.expectedBody["error"], response["error"])
			}

			svc.AssertExpectations(t)
		})
	}
}

func TestValidatorHandler_GetSyncDuties(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setupMock      func(*mockValidatorService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful sync duties",
			path: "/syncduties/12345",
			setupMock: func(svc *mockValidatorService) {
				svc.On("GetSyncCommitteeDuties", mock.Anything, uint64(12345)).Return(&domain.SyncCommitteeDuties{
					Validators: []string{"0xvalidator1", "0xvalidator2"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"data": map[string]interface{}{
					"validators": []interface{}{"0xvalidator1", "0xvalidator2"},
				},
			},
		},
		{
			name: "invalid slot format",
			path: "/syncduties/abc",
			setupMock: func(svc *mockValidatorService) {
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "invalid slot number",
			},
		},
		{
			name: "slot not found",
			path: "/syncduties/99999",
			setupMock: func(svc *mockValidatorService) {
				svc.On("GetSyncCommitteeDuties", mock.Anything, uint64(99999)).Return(nil, pkgerrors.ErrSlotNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error": "slot not found",
			},
		},
		{
			name: "slot too far in future",
			path: "/syncduties/999999",
			setupMock: func(svc *mockValidatorService) {
				svc.On("GetSyncCommitteeDuties", mock.Anything, uint64(999999)).Return(nil, pkgerrors.ErrSlotTooFarInFuture)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "requested slot is too far in the future",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockValidatorService)
			log := logger.New("error")

			handler, err := NewValidatorHandler(svc, log)
			assert.NoError(t, err)

			tt.setupMock(svc)

			req := httptest.NewRequest("GET", tt.path, nil)
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler.GetSyncDuties(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedBody["data"] != nil {
				assert.Equal(t, tt.expectedBody["data"], response["data"])
			}
			if tt.expectedBody["error"] != nil {
				assert.Equal(t, tt.expectedBody["error"], response["error"])
			}

			svc.AssertExpectations(t)
		})
	}
}

func TestValidatorHandler_Constructor(t *testing.T) {
	log := logger.New("error")
	svc := new(mockValidatorService)

	t.Run("nil service", func(t *testing.T) {
		_, err := NewValidatorHandler(nil, log)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validator service is required")
	})

	t.Run("nil logger", func(t *testing.T) {
		_, err := NewValidatorHandler(svc, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "logger is required")
	})

	t.Run("valid construction", func(t *testing.T) {
		handler, err := NewValidatorHandler(svc, log)
		assert.NoError(t, err)
		assert.NotNil(t, handler)
	})
}
