package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/0xPolygonID/refresh-service/service"
	"github.com/google/uuid"
	"github.com/iden3/iden3comm/v2"
	"github.com/iden3/iden3comm/v2/packers"
	iden3Protocol "github.com/iden3/iden3comm/v2/protocol"
)

type Handlers struct {
	packageManager *iden3comm.PackageManager
	refreshService *service.RefreshService
}

func NewHandlers(
	packageManager *iden3comm.PackageManager,
	refreshService *service.RefreshService,
) *Handlers {
	return &Handlers{
		packageManager: packageManager,
		refreshService: refreshService,
	}
}

func (h *Handlers) Run(port int) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		envelop, err := io.ReadAll(r.Body)
		if err != nil {
			internalError(w, err)
			return
		}
		message, _, err := h.packageManager.Unpack(envelop)
		if err != nil {
			internalError(w, err)
			return
		}
		if err := verifyMessageAttributes(message); err != nil {
			internalError(w, err)
			return
		}

		switch message.Type {
		case iden3Protocol.CredentialRefreshMessageType:
			var bodyMessage iden3Protocol.CredentialRefreshMessageBody
			err := json.Unmarshal(message.Body, &bodyMessage)
			if err != nil {
				internalError(w, err)
				return
			}
			ids := make([]string, 0, len(bodyMessage.Credentials))
			for _, credential := range bodyMessage.Credentials {
				ids = append(ids, credential.ID)
			}

			refreshed, err := h.refreshService.Process(message.To, message.From, ids)
			if err != nil {
				log.Printf("failed to process refresh: %v", err)
				internalError(w, err)
				return
			}

			// TODO (illia-korotia): currenly our issuance response supports only one credential
			r := refreshed[0]
			issuenceResponse := iden3Protocol.CredentialIssuanceMessage{
				ID:       uuid.New().String(),
				Type:     iden3Protocol.CredentialIssuanceResponseMessageType,
				ThreadID: message.ThreadID,
				Body: iden3Protocol.IssuanceMessageBody{
					Credential: *r,
				},
				From: message.To,
				To:   message.From,
			}
			payload, err := json.Marshal(issuenceResponse)
			if err != nil {
				internalError(w, err)
				return
			}

			envelop, err := h.packageManager.Pack(packers.MediaTypePlainMessage, payload, nil)
			if err != nil {
				internalError(w, err)
				return
			}

			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write(envelop)
			return
		default:
			internalError(w, fmt.Errorf("invalid message type"))
			return
		}
	})

	host := fmt.Sprintf("localhost:%d", port)
	log.Printf("listening on %s\n", host)
	return http.ListenAndServe(host, nil)
}

func verifyMessageAttributes(message *iden3comm.BasicMessage) error {
	if message.From == "" {
		return fmt.Errorf("missing from")
	}
	if message.To == "" {
		return fmt.Errorf("missing to")
	}
	return nil
}
