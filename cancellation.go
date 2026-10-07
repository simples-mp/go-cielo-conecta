package go_cielo_conecta

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type VoidInterface interface {
	CancelPayment(ctx context.Context, merchantVoidId string, merchantVoidDate time.Time) (VoidResponse, error)
	ConfirmCancel(ctx context.Context, merchantVoidId string) (ConfirmResponse, error)
}

type VoidHandler struct {
	client *Client
	info   Void
}

func NewVoidInformation(paymentID string, cardVoid VoidCard) Void {
	return Void{
		PaymentID:       paymentID,
		CardVoid:        cardVoid,
	}
}

func NewVoidHandler(c *Client, voidInfo Void) VoidInterface {
	return &VoidHandler{
		client: c,
		info:   voidInfo,
	}
}

func (h *VoidHandler) CancelPayment(ctx context.Context, merchantVoidId string, merchantVoidDate time.Time) (VoidResponse, error) {
	var voidResponse = VoidResponse{}

	body := VoidRequest{
		MerchantVoidId:   merchantVoidId,
		MerchantVoidDate: merchantVoidDate.Format("2006-01-02T15:04:05"),
		Card:             h.info.CardVoid,
	}

	h.client.LogInfo("cancel payment request body created", "body", body)

	req, err := h.client.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/1/physicalSales/%s/voids/", h.client.env.APIUrl, h.info.PaymentID),
		body,
	)
	if err != nil {
		return voidResponse, err
	}

	h.client.LogInfo("cancel payment request created", "method", req.Method, "url", req.URL.String())

	err = h.client.Send(req, &voidResponse)
	if err != nil {
		h.client.LogError("failed to send cancel payment request", "error", err)
		return voidResponse, err
	}

	h.client.LogInfo("cancel payment response received", "void_response", voidResponse)
	return voidResponse, nil
}

func (h *VoidHandler) ConfirmCancel(ctx context.Context, voidID string) (ConfirmResponse, error) {
	var confirmResponse = ConfirmResponse{}

	h.client.LogInfo("confirming cancellation", "void_id", voidID)

	req, err := h.client.NewRequestWithContext(ctx, http.MethodPut,
		fmt.Sprintf("%s/1/physicalSales/%s/voids/%s/confirmation", h.client.env.APIUrl, h.info.PaymentID, voidID),
		nil,
	)
	if err != nil {
		h.client.LogError("failed to create confirm cancellation request", "void_id", voidID, "error", err)
		return confirmResponse, err
	}

	h.client.LogInfo("confirm cancellation request created", "method", req.Method, "url", req.URL.String())

	err = h.client.Send(req, &confirmResponse)
	if err != nil {
		h.client.LogError("failed to send confirm cancellation request", "void_id", voidID, "error", err)
		return confirmResponse, err
	}

	h.client.LogInfo("confirm cancellation response received", "confirm_response", confirmResponse)
	return confirmResponse, nil
}
