package go_cielo_conecta

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type SaleInterface interface {
	Authorize(ctx context.Context) (Sale, error)
	Confirm(ctx context.Context, body ConfirmRequest) (ConfirmResponse, error)

	SetInstallments(installments int) SaleInterface
	SetInterest(interestType Interest) SaleInterface
	SetPinPadInfo(pinPad PinPadInformation) SaleInterface
	SetSoftDescriptor(softDesc string) SaleInterface
	SetCard(cardType string, card interface{}) SaleInterface

	Get() Sale
}

type SaleHandler struct {
	client *Client
	Sale   Sale
}

func (h *SaleHandler) Get() Sale {
	return h.Sale
}

// SetCard sets the card information for the sale based on the provided card type ("credit" or "debit") and card details.
//  It returns the SaleInterface for method chaining.
func (h *SaleHandler) SetCard(cardType string, card interface{}) SaleInterface {
	switch strings.ToLower(cardType) {
	case "credit":
		if cc, ok := card.(CreditCard); ok {
			return h.setCreditCard(cc)
		}
	case "debit":
		if dc, ok := card.(DebitCard); ok {
			return h.setDebitCard(dc)
		}
	default:
		h.client.LogError("invalid card type provided", "card_type", cardType)
	}

	return h
}

func (h *SaleHandler) setCreditCard(cc CreditCard) SaleInterface {
	h.Sale.Payment.DebitCard = nil
	h.Sale.Payment.CreditCard = &cc
	h.Sale.Payment.Type = "PhysicalCreditCard"
	h.client.LogInfo("setting credit card for sale", "order_id", h.Sale.MerchantOrderId)
	return h
}

func (h *SaleHandler) setDebitCard(dc DebitCard) SaleInterface {
	h.Sale.Payment.CreditCard = nil
	h.Sale.Payment.DebitCard = &dc
	h.Sale.Payment.Type = "PhysicalDebitCard"
	h.client.LogInfo("setting debit card for sale", "order_id", h.Sale.MerchantOrderId)
	return h
}

func (h *SaleHandler) SetSoftDescriptor(softDesc string) SaleInterface {
	h.Sale.Payment.SoftDescriptor = softDesc
	return h
}

func (h *SaleHandler) SetPinPadInfo(pinPad PinPadInformation) SaleInterface {
	h.Sale.Payment.PinPadInformation = &pinPad
	return h
}

func (h *SaleHandler) SetInterest(interestType Interest) SaleInterface {
	h.Sale.Payment.Interest = interestType
	return h
}

func (h *SaleHandler) SetInstallments(installments int) SaleInterface {
	h.Sale.Payment.Installments = installments
	return h
}

// Authorize validates the sale info and sends a requestBody to the API to authorize the payment.
// Returns the authorized sale with payment details or an error if the validation fails or if there is an issue with the API requestBody.
func (h *SaleHandler) Authorize(ctx context.Context) (Sale, error) {
	created := Sale{}

	h.client.LogInfo("creating sale", "sale", h.Sale)

	if err := h.validate(); err != nil {
		h.client.LogError("failed to validate sale", "sale", h.Sale, "error", err)
		return created, err
	}

	h.client.LogInfo("sending sale request", "sale", h.Sale)

	req, err := h.client.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s%s", h.client.env.APIUrl, "/1/physicalSales/"), h.Sale)
	if err != nil {
		h.client.LogError("failed to create sale request", "error", err)
		return Sale{}, err
	}

	h.client.LogInfo("sale request created", "method", req.Method, "url", req.URL.String())

	err = h.client.Send(req, &created)
	if err != nil {
		h.client.LogError("failed to send sale request", "error", err)
		return Sale{}, err
	}

	h.client.LogInfo("sale request response received", "created", created)
	h.Sale.Payment.ID = created.Payment.ID
	h.Sale.Payment.Status = created.Payment.Status
	h.Sale.Payment.ConfirmationStatus = created.Payment.ConfirmationStatus

	if created.Payment.Status != StatusPaymentConfirmed {
		h.client.LogError("sale request returned an unconfirmed payment", "status", created.Payment.Status)
		return created, errors.Join(ErrPaymentIsNotConfirmed, fmt.Errorf("status: %s", created.Payment.Status))
	}

	h.client.LogInfo("payment status is authorized", "status", created.Payment.Status)
	return created, nil
}

// ConfirmPayment confirms a payment with the provided issuer script results.
// Returns the confirmation result or an error if the validation fails or if there is an issue with the API requestBody.
//
// PUT /1/physicalSales/{ID}/confirmation
func (h *SaleHandler) Confirm(ctx context.Context, body ConfirmRequest) (ConfirmResponse, error) {
	var response ConfirmResponse

	h.client.LogInfo("confirming payment", "sale_id", h.Sale.Payment.ID)

	req, err := h.client.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/1/physicalSales/%s/confirmation", h.client.env.APIUrl, h.Sale.Payment.ID), body)
	if err != nil {
		h.client.LogError("failed to create confirm payment request", "sale_id", h.Sale.Payment.ID, "error", err)
		return response, err
	}

	h.client.LogInfo("confirm payment request created", "method", req.Method, "url", req.URL.String())

	err = h.client.Send(req, &response)
	if err != nil {
		h.client.LogError("failed to send confirm payment request", "sale_id", h.Sale.Payment.ID, "error", err)
		return response, err
	}

	h.client.LogInfo("confirm payment response received", "confirm_response", response)
	return response, nil
}

func (h *SaleHandler) validate() error {
	var errs error

	if h.Sale.MerchantOrderId == "" {
		errs = errors.Join(errs, ErrOrderIDRequired)
	}

	if h.Sale.Payment.Type == "" {
		errs = errors.Join(errs, ErrPaymentTypeRequired)
	}

	if h.Sale.Payment.SoftDescriptor == "" {
		errs = errors.Join(errs, ErrSoftDescriptorRequired)
	}

	if h.Sale.Payment.CreditCard == nil && h.Sale.Payment.DebitCard == nil {
		errs = errors.Join(errs, ErrCardRequired)
	}

	return errs
}
