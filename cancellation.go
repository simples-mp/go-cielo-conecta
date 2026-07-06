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

	h.client.LogInfo("cancel payment request body created", "body", body)

	req, err := h.client.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/1/physicalSales/%s/voids/", h.client.env.APIUrl, h.info.PaymentID),
		body,
	)
	if err != nil {
		return voidResponse, err
	}

	h.client.LogInfo("cancel payment request created", "method", req.Method, "url", req.URL.String())
	h.client.LogInfo("request headers", "headers", req.Header)
	h.client.LogInfo("request body", "body", req.Body)

	err = h.client.Send(req, &voidResponse)
	if err != nil {
		h.client.LogError("failed to send cancel payment request", "error", err)
		return voidResponse, err
	}

	h.client.LogInfo("cancel payment response received", "void_response", voidResponse)
	return voidResponse, nil
}

func (h *CancelHandler) ConfirmCancel(ctx context.Context, voidID string) (ConfirmResponse, error) {
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
	h.client.LogInfo("request headers", "headers", req.Header)
	h.client.LogInfo("request body", "body", req.Body)

	err = h.client.Send(req, &confirmResponse)
	if err != nil {
		h.client.LogError("failed to send confirm cancellation request", "void_id", voidID, "error", err)
		return confirmResponse, err
	}

	h.client.LogInfo("confirm cancellation response received", "confirm_response", confirmResponse)
	return confirmResponse, nil
}

func (h *CancelHandler) TryReversePayment(ctx context.Context) (ConfirmResponse, error) {
	var (
		result ConfirmResponse
		req    *http.Request
		err    error
	)

	body := map[string]string{"EmvData": h.info.EmvData}

	h.client.LogInfo("reverse payment request body created", "body", body)

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
		h.client.LogError("failed to create reverse payment request", "error", err)
		return ConfirmResponse{}, err
	}

	err = h.client.Send(req, &result)
	if err != nil {
		h.client.LogError("failed to send reverse payment request", "error", err)
		return result, err
	}

	h.client.LogInfo("reverse payment response received", "confirm_response", result)
	return result, nil
}
