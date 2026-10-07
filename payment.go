package go_cielo_conecta

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type SaleInfo struct {
	MerchantOrderID string // Unique identifier for the order. Default is a numeric string of the current timestamp in milliseconds if not provided.
	Amount          uint32 // Amount in BRL cents. e.g., for R$ 10.50, Amount should be 1050.
	ProductID       uint
	SoftDescriptor  string // Optional. A string that will appear on the cardholder's statement. Max length is 13 characters.
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
		SoftDescriptor:         info.SoftDescriptor,
		SubordinatedMerchantId: c.env.merchant.ID,
	}

	s := Sale{
		MerchantOrderId: info.MerchantOrderID,
		Payment:         p,
	}

	c.LogInfo("creating sale", "sale", s)

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

func (c *Client) GetPaymentByOrderID(ctx context.Context, orderID string) (Sale, error) {
	url := fmt.Sprintf("%s/1/physicalSales/MerchantOrderId/%s", c.env.APIQueryUrl, orderID)

	var sale []Sale

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

func (c *Client) ReversePayment(ctx context.Context, reverseInfo ReverseRequest) (ConfirmResponse, error) {
	var (
		url    string
		result ConfirmResponse
	)

	if reverseInfo.PaymentID != "" {
		url = fmt.Sprintf("%s/1/physicalSales/%s", c.env.APIUrl, reverseInfo.PaymentID)
	} else {
		url = fmt.Sprintf("%s/1/physicalSales/orderId/%s", c.env.APIUrl, reverseInfo.MerchantOrderId)
	}

	c.LogInfo("reversing payment", "merchant_order_id", reverseInfo.MerchantOrderId)

	req, err := c.NewRequestWithContext(ctx, http.MethodDelete, url, reverseInfo)
	if err != nil {
		c.LogError("failed to create reverse payment request", "body", reverseInfo, "error", err)
		return ConfirmResponse{}, err
	}
	c.LogInfo("reverse payment request created", "method", req.Method, "url", req.URL.String())

	err = c.Send(req, &result)
	if err != nil {
		c.LogError("failed to send reverse payment request", "body", reverseInfo, "error", err)
		return ConfirmResponse{}, err
	}

	c.LogInfo("reverse payment response received", "confirm_response", result)
	return result, nil
}
