package go_cielo_conecta

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (s StatusPayment) String() string {
	return [...]string{"NotFinished", "Pending", "Confirmed", "Cancelled", "Reversed", "Processing", "Denied", "Unreachable", "WaitingValidation", "WaitingCapture", "RefundedDevolution", "Refunded", "Approved"}[s]
}

func ParseStatusPayment(s string) (sp StatusPayment, err error) {
	switch strings.ToLower(s) {
	case "notfinished":
		sp = StatusPaymentNotFinished
	case "pending":
		sp = StatusPaymentPending
	case "confirmed":
		sp = StatusPaymentConfirmed
	case "cancelled":
		sp = StatusPaymentCancelled
	case "reversed":
		sp = StatusPaymentReversed
	case "processing":
		sp = StatusPaymentProcessing
	case "denied":
		sp = StatusPaymentDenied
	case "unreachable":
		sp = StatusPaymentUnreachable
	case "waitingvalidation":
		sp = StatusPaymentWaitingValidation
	case "waitingcapture":
		sp = StatusPaymentWaitingCapture
	case "refundeddevolution":
		sp = StatusPaymentRefundedDevolution
	case "refunded":
		sp = StatusPaymentRefunded
	case "approved":
		sp = StatusPaymentApproved
	default:
		return StatusPayment(0), fmt.Errorf("invalid StatusPayment: %s", s)
	}

	return sp, err
}

func (s *StatusPayment) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *StatusPayment) UnmarshalJSON(data []byte) error {
	var asInt uint
	if err := json.Unmarshal(data, &asInt); err == nil {
		*s = StatusPayment(asInt)
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err != nil {
		return err
	}

	sp, err := ParseStatusPayment(asString)
	if err != nil {
		return err
	}

	*s = sp

	return nil
}
