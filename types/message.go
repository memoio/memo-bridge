package types

type RequestMessage struct {
	Jsonrpc string           `json:"jsonrpc"`
	Id      int              `json:"id"`
	Method  string           `json:"method"`
	Params  []interface{}    `json:"params"`
}

type RespondMessage struct {
	Jsonrpc string           `json:"jsonrpc"`
	Result  uint64           `json:"result"`
	Id      int              `json:"id"`
}

type ErrorMessage struct {
	Jsonrpc string           `json:"jsonrpc"`
	Error   ErrorM           `json:"error"`
	id      int              `json:"id"`
}

type ErrorM struct {
	ErrorCode int            `json:"code"`
	Message   string         `json:"message"`
}

type EventMessage struct {
	Jsonrpc string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  EventParams      `json:"params"`
}