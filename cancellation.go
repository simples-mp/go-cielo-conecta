package go_cielo_conecta

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type CancelInterface interface {
	TryReversePayment(ctx context.Context) (ConfirmResponse, error)
	CancelPayment(ctx context.Context, merchantVoidId string) (VoidResponse, error)
	ConfirmCancel(ctx context.Context, voidID string) (ConfirmResponse, error)
}

type CancelHandler struct {
	client       *Client
	info         CancelRequest
	hasPaymentID bool
}

func newCancelHandler(c *Client, request CancelRequest) CancelInterface {
	hasPaymentID := false

	if request.PaymentID != "" {
		hasPaymentID = true
	}

	return &CancelHandler{
		client:       c,
		info:         request,
		hasPaymentID: hasPaymentID,
	}
}

func (h *CancelHandler) CancelPayment(ctx context.Context, merchantVoidId string) (VoidResponse, error) {
	var voidResponse = VoidResponse{}

	body := Void{
		MerchantVoidId:   merchantVoidId,
		MerchantVoidDate: time.Now().Format("2006-01-02T15:04:05"),
		Card:             h.info.CardVoid,
	}

	req, err := h.client.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/1/physicalSales/%s/voids/", h.client.env.APIUrl, h.info.PaymentID),
		body,
	)
	if err != nil {
		return voidResponse, err
	}

	err = h.client.Send(req, &voidResponse)
	if err != nil {
		return voidResponse, err
	}

	return voidResponse, nil
}

func (h *CancelHandler) ConfirmCancel(ctx context.Context, voidID string) (ConfirmResponse, error) {
	var confirmResponse = ConfirmResponse{}

	req, err := h.client.NewRequestWithContext(ctx, http.MethodPut,
		fmt.Sprintf("%s/1/physicalSales/%s/voids/%s/confirmation", h.client.env.APIUrl, h.info.PaymentID, voidID),
		nil,
	)
	if err != nil {
		return confirmResponse, err
	}

	err = h.client.Send(req, &confirmResponse)
	if err != nil {
		return confirmResponse, err
	}

	return confirmResponse, nil
}

func (h *CancelHandler) TryReversePayment(ctx context.Context) (ConfirmResponse, error) {
	var (
		result ConfirmResponse
		req    *http.Request
		err    error
	)

	body := map[string]string{"EmvData": h.info.EmvData}

	if h.hasPaymentID {
		req, err = h.client.NewRequestWithContext(ctx, http.MethodDelete,
			fmt.Sprintf("%s/1/physicalSales/%s", h.client.env.APIUrl, h.info.PaymentID),
			body,
		)
	} else {
		req, err = h.client.NewRequestWithContext(ctx, http.MethodDelete,
			fmt.Sprintf("%s/1/physicalSales/MerchantOrderId/%s", h.client.env.APIUrl, h.info.MerchantOrderId),
			body,
		)
	}

	if err != nil {
		return ConfirmResponse{}, err
	}

	err = h.client.Send(req, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
