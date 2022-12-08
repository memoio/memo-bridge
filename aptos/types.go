package aptos

import (
	"bytes"
	"fmt"
	"strconv"
)

type ErrorMessage struct {
	Message     string           `json:"message"`
	ErrorCode   string           `json:"error_code"`
	VmErrorCode string           `json:"vm_error_code"`
}

type Event struct {
	Version        Uint64        `json:"version"`
	GUID           UID           `json:"guid"`
	SequenceNumber Uint64        `json:"sequence_number"`
	Type           string        `json:"type"`
	Data           map[string]interface{} `json:"data"`
}

type UID struct {
	CreationNumber Uint64        `json:"creation_number"`
	AccountAddress string        `json:"account_address"`
}

type Uint64 uint64

func (u *Uint64) UnmarshalJSON(b []byte) error {
	b = bytes.Trim(b, "\"")
	v, err := strconv.ParseUint(string(b), 10, 64)
	if err != nil {
		return err
	}
	*u = Uint64(v)
	return nil
}

func (u Uint64) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%d\"", u)), nil
}