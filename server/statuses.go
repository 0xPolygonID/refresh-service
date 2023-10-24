package server

import (
	"encoding/json"
	"net/http"

	"github.com/0xPolygonID/refresh-service/logger"
	"github.com/0xPolygonID/refresh-service/providers/flexiblehttp"
	"github.com/0xPolygonID/refresh-service/service"
	"github.com/pkg/errors"
)

type jsonError struct {
	Code int    `json:"code"`
	Err  string `json:"error"`
}

func handleError(w http.ResponseWriter, err error) {
	var (
		message  string
		code     int
		httpCode int
	)
	switch {
	case errors.Is(err, flexiblehttp.ErrInvalidRequestSchema):
		code = 1000
		message = "check request schema in provider configuration file"
		httpCode = http.StatusInternalServerError
	case errors.Is(err, flexiblehttp.ErrInvalidResponseSchema):
		code = 1001
		message = "check response schema in provider configuration file"
		httpCode = http.StatusInternalServerError
	case errors.Is(err, flexiblehttp.ErrDataProviderIssue):
		code = 1002
		message = "check data provider to be available"
		httpCode = http.StatusInternalServerError

	case errors.Is(err, service.ErrInvalidProtocolMessage):
		code = 2000
		httpCode = http.StatusBadRequest
	case errors.Is(err, service.ErrInvalidProtocolResponse):
		code = 2001
		httpCode = http.StatusBadRequest

	case errors.Is(err, service.ErrIssuerNotSupported):
		code = 3000
		httpCode = http.StatusNotFound
		message = "check issuer node in refresh service configuration file"
	case errors.Is(err, service.ErrGetClaim):
		code = 3001
		httpCode = http.StatusInternalServerError
	case errors.Is(err, service.ErrCreateClaim):
		code = 3002
		httpCode = http.StatusInternalServerError

	case errors.Is(err, service.ErrCredentialNotUpdatable):
		code = 4000
		httpCode = http.StatusBadRequest
		message = "check that the credential you are trying to update has refreshService and the updatable flag is true"
	default:
		code = 500
		httpCode = http.StatusInternalServerError
	}

	logger.DefaultLogger.Error(err)
	if message != "" {
		logger.DefaultLogger.Info("possible solution: ", message)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	if err := json.NewEncoder(w).Encode(jsonError{
		Code: code,
		Err:  err.Error(),
	}); err != nil {
		logger.DefaultLogger.Errorf("failed to write response: %v", err)
	}
}
