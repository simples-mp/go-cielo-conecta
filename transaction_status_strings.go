package go_cielo_conecta

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ParseTransactionStatus(s string) (TransactionStatus, error) {
	var c TransactionStatus

	switch strings.ToUpper(s) {
	case "NOTFINISHED":
		c = TransactionStatusNotFinished
	case "AUTHORIZED":
		c = TransactionStatusAuthorized
	case "PAID":
		c = TransactionStatusPaid
	case "DENIED":
		c = TransactionStatusDenied
	case "CANCELLED":
		c = TransactionStatusCancelled
	case "ABORTED":
		c = TransactionStatusAborted
	default:
		return TransactionStatus(0), fmt.Errorf("invalid TransactionStatus: %s", s)
	}

	return c, nil
}

func (s TransactionStatus) String() string {
	return map[uint]string{0: "NotFinished", 1: "Authorized", 2: "Paid", 3: "Denied", 10: "Canceled", 13: "Aborted"}[uint(s)]
}

func (s *TransactionStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *TransactionStatus) UnmarshalJSON(data []byte) error {
	var asInt uint
	if err := json.Unmarshal(data, &asInt); err == nil {
		*s = TransactionStatus(asInt)
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err != nil {
		return err
	}

	ts, err := ParseTransactionStatus(asString)
	if err != nil {
		return err
	}

	*s = ts

	return nil
}
