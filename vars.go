package go_cielo_conecta

import "errors"

const (
	ByMerchant Interest = "ByMerchant"
	ByIssuer   Interest = "ByIssuer"

	CPF  IdentityType = "CPF"
	CNPJ IdentityType = "CNPJ"

	Typed          InputMode = "Typed"
	MagStripe      InputMode = "MagStripe"
	Emv            InputMode = "Emv"
	ContactlessEmv InputMode = "ContactlessEmv"

	NoPassword            AuthenticationMethod = "NoPassword"
	OnlineAuthentication  AuthenticationMethod = "OnlineAuthentication"
	OfflineAuthentication AuthenticationMethod = "OfflineAuthentication"

	WithoutPinPad                                   PhysicalCharacteristics = "WithoutPinPad"
	PinPadWithoutChipReader                         PhysicalCharacteristics = "PinPadWithoutChipReader"
	PinPadWithChipReaderWithoutSamModule            PhysicalCharacteristics = "PinPadWithChipReaderWithoutSamModule"
	PinPadWithChipReaderWithSamModule               PhysicalCharacteristics = "PinPadWithChipReaderWithSamModule"
	NotCertifiedPinPad                              PhysicalCharacteristics = "NotCertifiedPinPad"
	PinPadWithChipReaderWithoutSamAndContactless    PhysicalCharacteristics = "PinPadWithChipReaderWithoutSamAndContactless"
	PinPadWithChipReaderWithSamModuleAndContactless PhysicalCharacteristics = "PinPadWithChipReaderWithSamAndContactless"

	Collected   SecurityCodeStatus = "Collected"
	Unreadable  SecurityCodeStatus = "Unreadable"
	Nonexistent SecurityCodeStatus = "Nonexistent"

	CurrencyBRL = currency("BRL")
	CurrencyUSD = currency("USD")
)

const (
	DukptDes EncryptionType = iota + 1
	MasterKey
	Dukpt3Des
	Dukpt3DesCBC
)

const (
	StatusPaymentNotFinished StatusPayment = iota
	StatusPaymentPending
	StatusPaymentConfirmed
	StatusPaymentCancelled
	StatusPaymentReversed
	StatusPaymentProcessing
	StatusPaymentDenied
	StatusPaymentUnreachable
	StatusPaymentWaitingValidation
	StatusPaymentWaitingCapture
	StatusPaymentRefundedDevolution
	StatusPaymentRefunded
	StatusPaymentApproved
)

const (
	ConfirmationStatusPending ConfirmationStatus = iota
	ConfirmationStatusConfirmed
	ConfirmationStatusUndone
)

const (
	TransactionStatusNotFinished TransactionStatus = iota
	TransactionStatusAuthorized
	TransactionStatusPaid
	TransactionStatusDenied
	TransactionStatusCancelled = iota + 6
	TransactionStatusAborted   = iota + 8
)

var (
	ErrSendingRequest                  = errors.New("error sending request")
	ErrPaymentIsNotConfirmed           = errors.New("payment_status is not confirmed")
	ErrOrderIDRequired                 = errors.New("merchant_order_id is required")
	ErrPaymentRequired                 = errors.New("payment information is required")
	ErrCardRequired                    = errors.New("card information is required")
	ErrSoftDescriptorRequired          = errors.New("soft descriptor is required")
	ErrPaymentTypeRequired             = errors.New("payment type is required")
	ErrCancellationStatusNotAuthorized = errors.New("cancellation not authorized")
)

var (
	SandBoxEnv = Environment{
		OAuthURL:    "https://authsandbox.cieloecommerce.cielo.com.br/oauth2/token",
		ParamsURL:   "https://parametersdownloadsandbox.cieloecommerce.cielo.com.br/api/v0.1/initialization/{SubordinatedMerchantId}/{TerminalId}",
		APIUrl:      "https://apisandbox.cieloecommerce.cielo.com.br",
		APIQueryUrl: "https://apiquerysandbox.cieloecommerce.cielo.com.br",
	}

	HmlEnv = Environment{
		OAuthURL:     "https://authsandbox.cieloecommerce.cielo.com.br/oauth2/token",
		ParamsURL:    "https://parametersdownloadsandbox.cieloecommerce.cielo.com.br/api/v0.1/initialization/{SubordinatedMerchantId}/{TerminalId}",
		APIUrl:       "https://apisandbox.cieloecommerce.cielo.com.br",
		APIQueryUrl:  "https://apiquerysandbox.cieloecommerce.cielo.com.br",
		Homologation: true,
	}

	ProdEnv = Environment{
		OAuthURL:    "https://auth.cieloecommerce.cielo.com.br/oauth2/token",
		ParamsURL:   "https://parametersdownload.cieloecommerce.cielo.com.br/api/v0.1/initialization/{SubordinatedMerchantId}/{TerminalId}",
		APIUrl:      "https://api.cieloecommerce.cielo.com.br",
		APIQueryUrl: "https://apiquery.cieloecommerce.cielo.com.br/",
	}
)
