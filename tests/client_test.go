package tests

import (
	"os"
	"testing"

	. "github.com/simples-mp/go-cielo-conecta"
)

func TestNewClient(t *testing.T) {
	merchant := Merchant{
		ID:     os.Getenv("MERCHANT_ID"),
		Secret: os.Getenv("MERCHANT_SECRET"),
	}
	if merchant.ID == "" || merchant.Secret == "" {
		t.Skip("MERCHANT_ID and MERCHANT_SECRET are required for this integration test")
	}

	client, err := NewClient(SandBoxEnv.WithMerchant(merchant))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Successfully got token: %v", client)
}

func TestBinTables(t *testing.T) {
	merchant := Merchant{
		ID:     os.Getenv("MERCHANT_ID"),
		Secret: os.Getenv("MERCHANT_SECRET"),
	}
	if merchant.ID == "" || merchant.Secret == "" {
		t.Fatal("MERCHANT_ID and MERCHANT_SECRET are required for this integration test")
	}

	client, err := NewClient(HmlEnv.WithMerchant(merchant))
	if err != nil {
		t.Fatal(err)
	}

	data, err := client.SharedLibrary("00000001")
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := data["Merchant"]; !ok {
		t.Fatal("expected Merchant")
	}

	t.Log(data["Merchant"])
}
