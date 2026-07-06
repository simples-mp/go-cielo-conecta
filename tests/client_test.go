package tests

import (
	"os"
	"testing"
)
import . "github.com/edmfilho/go-cielo-conecta"

func TestNewClient(t *testing.T) {
	merchant := Merchant{
		ID:     os.Getenv("MERCHANT_ID"),
		Secret: os.Getenv("MERCHANT_SECRET"),
	}

	client, err := NewClient(merchant, SandboxEnvironment)
	if err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	t.Logf("Successfully got token: %v", client)
}

func TestBinTables(t *testing.T) {
	merchant := Merchant{
		ID:     os.Getenv("MERCHANT_ID"),
		Secret: os.Getenv("MERCHANT_SECRET"),
	}

	client, err := NewClient(merchant, HomologationEnvironment)
	if err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	data, err := client.SharedLibrary("00000001")
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := data["Merchant"]; !ok {
		t.Fatal("expected Merchant")
	}

	t.Log(data["Merchant"])
}
