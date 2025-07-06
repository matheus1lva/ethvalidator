package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/matheus/eth-validator-api/internal/api/middleware"
	"github.com/matheus/eth-validator-api/internal/service"
	pkgerrors "github.com/matheus/eth-validator-api/pkg/errors"
	"github.com/matheus/eth-validator-api/pkg/logger"
)

type ValidatorHandler struct {
	service service.ValidatorService
	logger  logger.Logger
}

func NewValidatorHandler(service service.ValidatorService, logger logger.Logger) (*ValidatorHandler, error) {
	if service == nil {
		return nil, errors.New("validator service is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	return &ValidatorHandler{
		service: service,
		logger:  logger,
	}, nil
}

type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func (h *ValidatorHandler) GetBlockReward(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)

	slot, err := h.parseSlotFromPath(r.URL.Path, "/blockreward/")
	if err != nil {
		h.logger.Warn().
			Str("request_id", requestID).
			Err(err).
			Msg("invalid slot parameter")
		h.respondError(w, http.StatusBadRequest, pkgerrors.ErrInvalidSlot)
		return
	}

	h.logger.Info().
		Str("request_id", requestID).
		Uint64("slot", slot).
		Msg("processing block reward request")

	reward, err := h.service.GetBlockReward(ctx, slot)
	if err != nil {
		h.handleServiceError(w, err, requestID)
		return
	}

	h.respondJSON(w, http.StatusOK, reward)
}

func (h *ValidatorHandler) GetSyncDuties(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)

	slot, err := h.parseSlotFromPath(r.URL.Path, "/syncduties/")
	if err != nil {
		h.logger.Warn().
			Str("request_id", requestID).
			Err(err).
			Msg("invalid slot parameter")
		h.respondError(w, http.StatusBadRequest, pkgerrors.ErrInvalidSlot)
		return
	}

	h.logger.Info().
		Str("request_id", requestID).
		Uint64("slot", slot).
		Msg("processing sync duties request")

	duties, err := h.service.GetSyncCommitteeDuties(ctx, slot)
	if err != nil {
		h.handleServiceError(w, err, requestID)
		return
	}

	h.respondJSON(w, http.StatusOK, duties)
}

func (h *ValidatorHandler) parseSlotFromPath(path, prefix string) (uint64, error) {
	if !strings.HasPrefix(path, prefix) {
		return 0, pkgerrors.NewValidationError("path", path, pkgerrors.ErrInvalidSlot)
	}

	slotStr := strings.TrimPrefix(path, prefix)
	slotStr = strings.TrimSuffix(slotStr, "/")

	if slotStr == "" {
		return 0, pkgerrors.NewValidationError("slot", "", pkgerrors.ErrInvalidSlot)
	}

	slot, err := strconv.ParseUint(slotStr, 10, 64)
	if err != nil {
		return 0, pkgerrors.NewValidationError("slot", slotStr, err)
	}

	return slot, nil
}

func (h *ValidatorHandler) handleServiceError(w http.ResponseWriter, err error, requestID string) {
	switch {
	case pkgerrors.IsNotFound(err):
		h.logger.Info().
			Str("request_id", requestID).
			Err(err).
			Msg("resource not found")
		h.respondError(w, http.StatusNotFound, err)

	case pkgerrors.IsBadRequest(err):
		h.logger.Warn().
			Str("request_id", requestID).
			Err(err).
			Msg("bad request")
		h.respondError(w, http.StatusBadRequest, err)

	case pkgerrors.IsTimeout(err):
		h.logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("request timeout")
		h.respondError(w, http.StatusRequestTimeout, err)

	default:
		h.logger.Error().
			Str("request_id", requestID).
			Err(err).
			Msg("internal server error")
		h.respondError(w, http.StatusInternalServerError, pkgerrors.ErrInternal)
	}
}

func (h *ValidatorHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := Response{Data: data}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode response")
	}
}

func (h *ValidatorHandler) respondError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := Response{Error: err.Error()}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode error response")
	}
}
