package go_cielo_conecta

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (e EncryptionType) String() string {
	return [...]string{"DukptDes", "MasterKey", "Dukpt3Des", "Dukpt3DesCBC"}[e-1]
}

func ParseEncryptionType(s string) (EncryptionType, error) {
	var e EncryptionType

	switch strings.ToLower(s) {
	case "dukptdes":
		e = DukptDes
	case "masterkey":
		e = MasterKey
	case "dukpt3des":
		e = Dukpt3Des
	case "dukpt3descbc":
		e = Dukpt3DesCBC
	default:
		return 0, fmt.Errorf("invalid encryption_type: %s", s)
	}

	return e, nil
}

func (e *EncryptionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

func (e *EncryptionType) UnmarshalJSON(data []byte) error {
	var asInt uint
	if err := json.Unmarshal(data, &asInt); err == nil {
		*e = EncryptionType(asInt)
		return nil
	}

	var asString string
	err := json.Unmarshal(data, &asString)
	if err != nil {
		return err
	}

	value, err := ParseEncryptionType(asString)
	if err != nil {
		return err
	}

	*e = value

	return nil
}
