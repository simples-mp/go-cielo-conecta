package go_cielo_conecta

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type (
	Client struct {
		sync.Mutex

		Client *http.Client
		env    Environment
		token  *tokenResponse
		log    *slog.Logger

		cancel context.CancelFunc
		wg     sync.WaitGroup
		once   sync.Once
	}

	Environment struct {
		OAuthURL     string
		ParamsURL    string
		APIUrl       string
		APIQueryUrl  string
		Homologation bool
		merchant     Merchant
	}

	tokenResponse struct {
		AccessToken string        `json:"access_token"`
		TokenType   string        `json:"token_type"`
		ExpiresIn   time.Duration `json:"expires_in"`
	}

	Merchant struct {
		ID, Secret string
	}

	ErrCielo struct {
		Code    int    `json:",omitempty"`
		Message string `json:",omitempty"`
		Err     error  `json:"-"`
	}

	MultiErr []ErrCielo

	Sale struct {
		MerchantOrderId string    `json:"MerchantOrderId"`
		Customer        *Customer `json:"Customer,omitempty"`
		Payment         Payment   `json:"Payment"`
	}

	Customer struct {
		Name            string       `json:",omitempty"`
		Identity        string       `json:",omitempty"`
		IdentityType    IdentityType `json:",omitempty"`
		Email           string       `json:",omitempty"`
		Birthday        string       `json:",omitempty"`
		Address         *Address     `json:",omitempty"`
		DeliveryAddress *Address     `json:",omitempty"`
	}

	Address struct {
		Street, Number, Complement, City, State, ZipCode, Country string `json:",omitempty"`
	}

	Payment struct {
		ID                        string                `json:"PaymentId,omitempty"`
		Installments              int                   `json:",omitempty"` // Installments Quantidade de parcelas: varia de 2 a 99 para transação de financiamento.
		Type                      string                `json:",omitempty"`
		Interest                  Interest              `json:",omitempty"`
		Capture                   bool                  `json:",omitempty"` // Capture identifica que a autorização deve ser com captura automática. A autorização sem captura automática é conhecida também como pré-autorização.
		SoftDescriptor            string                `json:",omitempty"`
		CreditCard                *CreditCard           `json:",omitempty"`
		DebitCard                 *DebitCard            `json:",omitempty"`
		PaymentDateTime           string                `json:",omitempty"`
		Amount                    uint32                `json:",omitempty"`
		ProductId                 uint                  `json:",omitempty"`
		ReceivedDate              string                `json:",omitempty"`
		CapturedAmount            uint32                `json:",omitempty"`
		CapturedDate              string                `json:",omitempty"`
		Provider                  string                `json:",omitempty"`
		Status                    StatusPayment         `json:",omitempty"`
		PhysicalTransactionStatus uint                  `json:",omitempty"`
		IsSplitted                bool                  `json:",omitempty"`
		ReturnMessage             string                `json:",omitempty"`
		ExtendedMessage           string                `json:",omitempty"`
		ReturnCode                string                `json:",omitempty"`
		Currency                  string                `json:",omitempty"`
		Country                   string                `json:",omitempty"`
		Links                     []Link                `json:",omitempty"`
		ServiceTaxAmount          uint64                `json:",omitempty"`
		PinPadInformation         *PinPadInformation    `json:",omitempty"`
		PrintMessage              any                   `json:",omitempty"`
		ReceiptInformation        []*ReceiptInformation `json:",omitempty"`
		Receipt                   map[string]string     `json:",omitempty"`
		AuthorizationCode         string                `json:",omitempty"`
		ProofOfSale               string                `json:",omitempty"`
		InitializationVersion     int64                 `json:",omitempty"`
		ConfirmationStatus        ConfirmationStatus    `json:",omitempty"`
		EmvResponseData           string                `json:",omitempty"`
		SubordinatedMerchantId    string                `json:",omitempty"`
		OfflinePaymentType        string                `json:",omitempty"`
		MerchantAcquirerId        string                `json:",omitempty"`
		TerminalAcquirerId        string                `json:",omitempty"`
	}

	ReceiptInformation struct {
		Field   string `json:",omitempty"`
		Label   string `json:",omitempty"`
		Content string `json:",omitempty"`
	}

	Link struct {
		Method string `json:",omitempty"`
		Rel    string `json:",omitempty"`
		Href   string `json:",omitempty"`
	}

	CreditCard struct {
		InputMode                      InputMode            `json:",omitempty"`
		ExpirationDate                 string               `json:",omitempty"`
		AuthenticationMethod           AuthenticationMethod `json:",omitempty"`
		IssuerId                       int                  `json:",omitempty"`
		BrandId                        int                  `json:",omitempty"`
		TrackOneData                   string               `json:",omitempty"`
		TrackTwoData                   string               `json:",omitempty"`
		EmvData                        string               `json:",omitempty"`
		EncryptedCardData              EncryptedCardData    `json:",omitempty"`
		SecurityCodeStatus             SecurityCodeStatus   `json:",omitempty"`
		SecurityCode                   string               `json:",omitempty"`
		TruncateCardNumberWhenPrinting bool                 `json:",omitempty"`
		SaveCard                       bool                 `json:",omitempty"`
		PanSequenceNumber              uint                 `json:",omitempty"`
		IsFallback                     bool                 `json:",omitempty"`
		BrandInformation               BrandInformation     `json:",omitempty"`
		PinBlock                       PinBlock             `json:",omitempty"`
	}

	DebitCard struct {
		InputMode                      InputMode            `json:",omitempty"`
		ExpirationDate                 string               `json:",omitempty"`
		AuthenticationMethod           AuthenticationMethod `json:",omitempty"`
		IssuerId                       uint                 `json:",omitempty"`
		BrandId                        uint                 `json:",omitempty"`
		TruncateCardNumberWhenPrinting bool                 `json:",omitempty"`
		PanSequenceNumber              uint                 `json:",omitempty"`
		SaveCard                       bool                 `json:",omitempty"`
		EmvData                        string               `json:",omitempty"`
		TrackOneData                   string               `json:",omitempty"`
		TrackTwoData                   string               `json:",omitempty"`
		EncryptedCardData              EncryptedCardData    `json:",omitempty"`
		PinBlock                       PinBlock             `json:",omitempty"`
		IsFallback                     bool                 `json:",omitempty"`
		CardToken                      string               `json:",omitempty"`
		BrandInformation               BrandInformation     `json:",omitempty"`
		SecurityCodeStatus             SecurityCodeStatus   `json:",omitempty"`
		SecurityCode                   string               `json:",omitempty"`
	}

	BrandInformation struct {
		Type string `json:",omitempty"`
		Name string `json:",omitempty"`
	}

	PinBlock struct {
		EncryptedPinBlock string         `json:",omitempty"`
		EncryptionType    EncryptionType `json:",omitempty"`
		KsnIdentification string         `json:",omitempty"`
	}

	EncryptedCardData struct {
		EncryptionType       EncryptionType `json:"EncryptionType,omitempty"`
		TrackOneDataKSN      string         `json:"TrackOneDataKSN,omitempty"`
		TrackTwoDataKSN      string         `json:"TrackTwoDataKSN,omitempty"`
		InitializationVector string         `json:"InitializationVector,omitempty"`
		IsDataInTLVFormat    bool           `json:"IsDataInTLVFormat,omitempty"`
	}

	PinPadInformation struct {
		PhysicalCharacteristics PhysicalCharacteristics `json:",omitempty"`
		ReturnDataInfo          string                  `json:",omitempty"`
		SerialNumber            string                  `json:",omitempty"`
		TerminalID              string                  `json:",omitempty"`
	}

	ConfirmResponse struct {
		CancellationStatus CancellationStatus `json:"CancellationStatus,omitempty"`
		ConfirmationStatus ConfirmationStatus `json:"ConfirmationStatus,omitempty"`
		Status             TransactionStatus  `json:"Status,omitempty"`
		ReasonCode         uint               `json:"ReasonCode,omitempty"`
		ReturnCode         string             `json:"ReturnCode,omitempty"`
		ReturnMessage      string             `json:"ReturnMessage,omitempty"`
		Links              []*Link            `json:"Links,omitempty"`
	}

	Void struct {
		MerchantVoidId   string   `json:"MerchantVoidId"`
		MerchantVoidDate string   `json:"MerchantVoidDate"`
		Card             CardVoid `json:"Card"`
	}

	VoidResponse struct {
		VoidId                    string             `json:"VoidId,omitempty"`
		CancellationStatus        CancellationStatus `json:"CancellationStatus,omitempty"`
		InitializationVersion     int64              `json:"InitializationVersion,omitempty"`
		PrintMessage              any                `json:"PrintMessage,omitempty"`
		Receipt                   map[string]string  `json:"Receipt,omitempty"`
		ConfirmationStatus        ConfirmationStatus `json:"ConfirmationStatus,omitempty"`
		ExtendedMessage           string             `json:"ExtendedMessage,omitempty"`
		Status                    TransactionStatus  `json:"Status,omitempty"`
		PhysicalTransactionStatus uint               `json:"PhysicalTransactionStatus,omitempty"`
		ReasonCode                uint               `json:"ReasonCode,omitempty"`
		ReasonMessage             string             `json:"ReasonMessage,omitempty"`
		ReturnCode                string             `json:"ReturnCode,omitempty"`
		ReturnMessage             string             `json:"ReturnMessage,omitempty"`
		Links                     []*Link            `json:"Links,omitempty"`
	}

	CardVoid struct {
		InputMode         InputMode         `json:"InputMode"`
		EmvData           string            `json:"EmvData"`
		TrackOneData      string            `json:"TrackOneData,omitempty"`
		TrackTwoData      string            `json:"TrackTwoData"`
		EncryptedCardData EncryptedCardData `json:"EncryptedCardData"`
	}

	CancelRequest struct {
		PaymentID       string
		MerchantOrderId string
		EmvData         string
		CardVoid        CardVoid
	}
)
type (
	IdentityType            string
	Interest                string
	InputMode               string
	AuthenticationMethod    string
	SecurityCodeStatus      string
	PhysicalCharacteristics string

	currency string
)

type (
	EncryptionType     uint
	StatusPayment      uint
	ConfirmationStatus uint
	TransactionStatus  uint
	CancellationStatus uint
)

func (er ErrCielo) Error() string {
	return fmt.Sprintf("cielo_code=%d, message=%s, err_cielo=%v", er.Code, er.Message, er.Err)
}

func (me MultiErr) Error() string {
	var errStr strings.Builder
	for _, er := range me {
		errStr.WriteString(er.Error())
		errStr.WriteString("\n")
	}
	return errStr.String()
}

func (e Environment) WithMerchant(m Merchant) Environment {
	e.merchant = m
	return e
}

func (s Sale) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("order_id", s.MerchantOrderId),
		slog.Any("payment", s.Payment),
	)
}

func (c ConfirmResponse) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("return_message", c.ReturnMessage),
		slog.String("status", c.Status.String()),
		slog.String("confirmation_status", c.ConfirmationStatus.String()),
		slog.Uint64("reason_code", uint64(c.ReasonCode)),
	)
}

func (p Payment) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("payment_id", p.ID),
		slog.String("status", p.Status.String()),
		slog.String("confirmation_status", p.ConfirmationStatus.String()),
		slog.String("return_message", p.ReturnMessage),
		slog.String("extended_message", p.ExtendedMessage),
	)
}

func (p *Payment) toCardVoid() CardVoid {
	if p.CreditCard != nil {
		return CardVoid{
			InputMode:         p.CreditCard.InputMode,
			EmvData:           p.CreditCard.EmvData,
			TrackOneData:      p.CreditCard.TrackOneData,
			TrackTwoData:      p.CreditCard.TrackTwoData,
			EncryptedCardData: p.CreditCard.EncryptedCardData,
		}
	}

	return CardVoid{
		InputMode:         p.DebitCard.InputMode,
		EmvData:           p.DebitCard.EmvData,
		TrackOneData:      p.DebitCard.TrackOneData,
		TrackTwoData:      p.DebitCard.TrackTwoData,
		EncryptedCardData: p.DebitCard.EncryptedCardData,
	}
}

func (s *Sale) IsReversible() bool {
	return s.Payment.Status == StatusPaymentConfirmed && s.Payment.ConfirmationStatus != ConfirmationStatusConfirmed
}

func (s *Sale) IsCancellable() bool {
	return s.Payment.Status == StatusPaymentConfirmed && s.Payment.ConfirmationStatus == ConfirmationStatusConfirmed
}

func (p *Payment) getEmvData() string {
	if p.CreditCard != nil {
		return p.CreditCard.EmvData
	}

	if p.DebitCard != nil {
		return p.DebitCard.EmvData
	}

	return ""
}
