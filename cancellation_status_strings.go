package go_cielo_conecta

import (
	"encoding/json"
	"errors"
	"strings"
)

const (
	CancellationStatusNotFinished CancellationStatus = iota
	CancellationStatusAuthorized
	CancellationStatusDenied
	CancellationStatusConfirmed
	CancellationStatusReversed
)

func (c CancellationStatus) String() string {
	return [...]string{"NotFinished", "Authorized", "Denied", "Confirmed", "Reversed"}[c]
}

func ParseCancellationStatus(s string) (c CancellationStatus, err error) {
	switch strings.ToUpper(s) {
	case "NOTFINISHED":
		c = CancellationStatusNotFinished
	case "AUTHORIZED":
		c = CancellationStatusAuthorized
	case "DENIED":
		c = CancellationStatusDenied
	case "CONFIRMED":
		c = CancellationStatusConfirmed
	case "REVERSED":
		c = CancellationStatusReversed
	default:
		return CancellationStatus(0), errors.New("invalid CancellationStatus: " + s)
	}
	return c, nil
}

func (c *CancellationStatus) UnmarshalJSON(b []byte) error {
	var asInt uint
	if err := json.Unmarshal(b, &asInt); err == nil {
		*c = CancellationStatus(asInt)
		return nil
	}

	var asString string
	if err := json.Unmarshal(b, &asString); err != nil {
		return err
	}

	cs, err := ParseCancellationStatus(asString)
	if err != nil {
		return err
	}

	*c = cs
	return nil
}
