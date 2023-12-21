package service

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/iden3/iden3comm/v2"
	"github.com/iden3/iden3comm/v2/packers"
	iden3Protocol "github.com/iden3/iden3comm/v2/protocol"
	"github.com/pkg/errors"
)

var (
	ErrInvalidProtocolMessage  = errors.New("invalid protocol message")
	ErrInvalidProtocolResponse = errors.New("invalid protocol response")
)

type AgentService struct {
	refreshService *RefreshService
	packageManager *iden3comm.PackageManager
}

func NewAgentService(refreshService *RefreshService,
	packageManager *iden3comm.PackageManager) *AgentService {
	return &AgentService{
		refreshService: refreshService,
		packageManager: packageManager,
	}
}

func (as *AgentService) Process(envelop []byte) (
	[]byte, error) {
	message, _, err := as.packageManager.Unpack(envelop)
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidProtocolMessage, "failed to unpack message: %v", err)
	}
	if err := verifyMessageAttributes(message); err != nil {
		return nil, errors.Wrapf(ErrInvalidProtocolMessage, "failed to verify message attributes: %v", err)
	}

	switch message.Type {
	case iden3Protocol.CredentialRefreshMessageType:
		var bodyMessage iden3Protocol.CredentialRefreshMessageBody
		err := json.Unmarshal(message.Body, &bodyMessage)
		if err != nil {
			return nil, errors.Wrapf(ErrInvalidProtocolMessage, "failed to unmarshal body: %v", err)
		}

		refreshed, err := as.refreshService.Process(
			message.To,
			message.From,
			convertID(bodyMessage.ID),
		)
		if err != nil {
			return nil, err
		}

		issuenceResponse := iden3Protocol.CredentialIssuanceMessage{
			ID:       uuid.New().String(),
			Type:     iden3Protocol.CredentialIssuanceResponseMessageType,
			ThreadID: message.ThreadID,
			Body: iden3Protocol.IssuanceMessageBody{
				Credential: *refreshed,
			},
			From: message.To,
			To:   message.From,
		}
		payload, err := json.Marshal(issuenceResponse)
		if err != nil {
			return nil, errors.Wrap(ErrInvalidProtocolResponse, err.Error())
		}

		envelop, err := as.packageManager.Pack(packers.MediaTypePlainMessage, payload, nil)
		if err != nil {
			return nil, errors.Wrapf(ErrInvalidProtocolResponse, "failed pack message: %v", err)
		}

		return envelop, nil
	default:
		return nil, errors.Errorf("unknown message type '%s'", message.Type)
	}
}

func verifyMessageAttributes(message *iden3comm.BasicMessage) error {
	if message.From == "" {
		return errors.New("missing 'from' field in message")
	}
	if message.To == "" {
		return errors.New("missing 'to' field in message")
	}
	return nil
}

/*
TODO(illia-korotia): temporary solution,
need to communicate with the mobile team to pass the correct ID
*/
func convertID(id string) string {
	if strings.HasPrefix(id, "urn:uuid:") {
		return strings.TrimPrefix(id, "urn:uuid:")
	}
	parts := strings.Split(id, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return id
}
