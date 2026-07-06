package go_cielo_conecta

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type SaleInfo struct {
	OrderID   string
	Amount    uint32 // Amount in BRL cents. e.g., for R$ 10.50, Amount should be 1050.
	ProductID uint
}

// CreateSale initializes a new payment with the provided order ID, amount (in cents), and product ID.
// It sets default values for installments, interest, capture, and payment date/time.
// The amount is converted to cents and rounding to the nearest integer.
//
// The method returns a SaleInterface that can be used to further customize the sale or execute it.
func (c *Client) CreateSale(info SaleInfo) SaleInterface {
	p := Payment{
		Installments:           1,          // Can be changed with SetInstallments().
		Interest:               ByMerchant, // Can be changed with SetInterest().
		Capture:                true,
		PaymentDateTime:        time.Now().Format("2006-01-02T15:04:05"),
		Amount:                 info.Amount,
		ProductId:              info.ProductID,
		SubordinatedMerchantId: c.env.merchant.ID,
	}

	s := Sale{
		MerchantOrderId: info.OrderID,
		Payment:         p,
	}

	return &SaleHandler{client: c, Sale: s}
}

func (c *Client) GetPaymentByID(ctx context.Context, paymentId string) (Sale, error) {
	var sale Sale

	c.LogInfo("getting payment by id", "payment_id", paymentId)

	req, err := c.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/1/physicalSales/%s", c.env.APIQueryUrl, paymentId), nil)
	if err != nil {
		c.LogError("failed to create get payment by id request", "payment_id", paymentId, "error", err)
		return sale, err
	}

	c.LogInfo("get payment by id request created", "method", req.Method, "url", req.URL.String())

	err = c.Send(req, &sale)
	if err != nil {
		c.LogError("failed to send get payment by id request", "payment_id", paymentId, "error", err)
		return sale, err
	}

	c.LogInfo("payment by id response received", "sale", sale)
	return sale, nil
}

func (c *Client) GetPaymentByOrderID(ctx context.Context, orderID string, date ...time.Time) (Sale, error) {
	url := fmt.Sprintf("%s/1/physicalSales/MerchantOrderId/%s", c.env.APIQueryUrl, orderID)

	var sale []Sale

	if len(date) > 0 {
		url = fmt.Sprintf("%s?transactionDate=%s", url, date[0].Format("2006/01/02"))
	}

	req, err := c.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.LogError("failed to create get payment by order id request", "order_id", orderID, "error", err)
		return Sale{}, err
	}

	c.LogInfo("get payment by order id request created", "method", req.Method, "url", req.URL.String())

	err = c.Send(req, &sale)
	if err != nil {
		c.LogError("failed to send get payment by order id request", "order_id", orderID, "error", err)
		return Sale{}, err
	}

	if len(sale) > 0 {
		c.LogInfo("payment by order id response received", "sale", sale[0])
		return sale[0], nil
	}

	c.LogInfo("payment by order id response received with no sales", "order_id", orderID)
	return Sale{}, nil
}

func (c *Client) ReversePayment(ctx context.Context, sale Sale) (ConfirmResponse, error) {
	cancel := newCancelHandler(c, CancelRequest{
		PaymentID:       sale.Payment.ID,
		MerchantOrderId: sale.MerchantOrderId,
		EmvData:         sale.Payment.getEmvData(),
	})

	c.LogInfo("reversing payment", "sale", sale)
	return cancel.TryReversePayment(ctx)
}

func (c *Client) CancelPayment(ctx context.Context, sale Sale, merchantVoidId string) (ConfirmResponse, error) {
	cancel := newCancelHandler(c, CancelRequest{
		PaymentID:       sale.Payment.ID,
		MerchantOrderId: sale.MerchantOrderId,
		CardVoid:        sale.Payment.toCardVoid(),
	})

	var confirmResponse ConfirmResponse
	c.LogInfo("canceling payment", "sale", sale)

	voidResponse, err := cancel.CancelPayment(ctx, merchantVoidId)
	if err != nil {
		c.LogError("failed to cancel payment", "sale", sale, "error", err)
		return confirmResponse, err
	}

	confirmResponse = ConfirmResponse{
		CancellationStatus: voidResponse.CancellationStatus,
		Status:             voidResponse.Status,
		ReturnMessage:      voidResponse.ExtendedMessage,
		ConfirmationStatus: voidResponse.ConfirmationStatus,
	}

	c.LogInfo("payment cancellation response received", "void_response", voidResponse)

	if voidResponse.CancellationStatus != CancellationStatusAuthorized {
		c.LogError("payment cancellation status not authorized", "cancellation_status", voidResponse.CancellationStatus)
		return confirmResponse, ErrCancellationStatusNotAuthorized
	}

	c.LogInfo("confirming payment cancellation", "void_id", voidResponse.VoidId)
	return cancel.ConfirmCancel(ctx, voidResponse.VoidId)
}
