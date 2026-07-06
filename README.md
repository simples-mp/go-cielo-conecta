# Go Cielo Conecta
___
Golang lib to communicate with Cielo Conecta API.

### Prerequisite knowledge
- <a href="https://developercielo.github.io/manual/cielo-conecta" target="_blank">Docs Cielo Conecta</a> 

### Requirements
- Go 1.18 or higher
- Cielo Conecta API credentials

### TODOs:
- Better tests
- Implement endpoints:
  - Terminals  
  - Stores
  - Equipments

### Installation

```shell
go get github.com/simples-mp/go-cielo-conecta
```

### New Client

```go
var merchant = cieloConecta.Merchant{
  ID:     "your_merchant_id",
  Secret: "your_merchant_secret",
}

// Use client to make API calls
client, err := NewClient(SandBox.WithMerchant(merchant))
if err != nil {
  log.Fatalf("failed to initialize client: %v", err)
}

// Remember to close the client when you're done (this will stop the goroutine that refreshes the token)
defer client.Close()

// By default, the requests will be logged to the standard output with log/slog.
// You can disable this by setting the logger to nil.
client.SetLogger(nil)

// By the way, you can also set a custom logger that implements the Logger interface.
client.SetLogger(&MyCustomLogger{})
```

### Create a new payment:

```go
// Read more about creating payments in the documentation: https://developercielo.github.io/manual/cielo-conecta#fluxo-de-pagamento

// Fill in the credit card details
cc := &cieloConecta.CreditCard{}

// Create a new sale with the order ID, amount, and product ID
saleInfo := cieloConecta.SaleInfo{OrderID: OrderID, Amount: 5000, ProductID: 1}

// Create a new sale handler using the client and the sale information
saleHandler := client.CreateSale(saleInfo)

// Set additional information for the sale, such as soft descriptor, credit card/debit card details, installments, and pin pad information
saleHandler.SetSoftDescriptor("Test").
  WithCreditCard(cc).
  SetInstallments(1).
  SetPinPadInformation(cieloConecta.PinPadInformation{})

// You can also set additional information, such as customer data, shipping address, etc. Check the documentation for more details.
authorizedSale, err := saleHandler.Authorize(context.Background())  
if err != nil {
  log.Fatal(err)
}

log.Println("Payment created successfully: ", authorizedSale)

response, err := saleHandler.Confirm(context.Background())
if err != nil {
  log.Fatal(err)
}

log.Println("Payment confirmed successfully: ", authorizedSale)
```

### Cancel a payment:

```go
// Read more about canceling payments in the documentation: https://developercielo.github.io/manual/cielo-conecta#cancelamento-de-pagamento

// You need to have the original sale information to cancel a payment, such as the OrderID/PaymentID and Card details.
// You can use GetPayment() to retrieve the payment information if you don't have it stored.
sale, err := client.GetPaymentByOrderID(context.Background(), OrderID)
if err != nil {
  log.Fatal(err)
}

// Fill in the credit card details (you can also use debit card details if it was a debit payment)
cc := &cieloConecta.CreditCard{}
// or 
dc := &cieloConecta.DebitCard{}

// Try to cancel using the original sale information and a unique merchantVoidID (you can use a timestamp or incremental ID)
confirmResponse, err := client.CancelPayment(context.Background(), sale, merchantVoidID)
if err != nil {
  // Handle the error, which could be due to various reasons such as invalid sale information, cancellation not authorized, etc.
  log.Fatal(err)
}

log.Println("Payment cancellation response: ", confirmResponse)
```
