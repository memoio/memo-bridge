package sui

type MoveEventType struct {
	MoveEvent string                  `json:"MoveEvent"`
}

type RequestMessage struct {
	Jsonrpc string                    `json:"jsonrpc"`
	Id      int                       `json:"id"`
	Method  string                    `json:"method"`
	Params  []interface{}             `json:"params"`
}

type RespondMessage struct { 
	Jsonrpc string                    `json:"jsonrpc"`
	Result  SuiEventResult            `json:"result"`
	Id      int                       `json:"id"`
}

type SuiEventResult struct {
	Events []SuiEventEnvelope         `json:"data"`
	NextCursor EventID                `json:"nextCoursor"`
}

// type SubscribeEvent struct {
// 	Jsonrpc string                    `json:"jsonrpc"`
// 	Method  string                    `json:"method"`
// 	Params  EventParams               `json:"params"`
// }

// received message type
// type EventParams struct {
// 	Subscription uint64               `json:"subscription"`
// 	Result       SuiEventEnvelope     `json:"result"`
// }

type SuiEventEnvelope struct {
	TimeStamp uint64                  `json:"timestamp"`
	TxDigest  string                  `json:"txDigest"`
	ID        EventID                 `json:"id"`
	Event     SuiEvent                `json:"event"`
}

type EventID struct {
	TxSeq    int                      `json:"txSeq"`
	EventSeq int                      `json:"eventSeq"`
}

type SuiEvent struct {
	// TODO: add more event (PublishEvent)
	MoveEvent MoveEvent               `json:"moveEvent"`
}

type MoveEvent struct {
	PackageID         string          `json:"packageId"`
	TransactionModule string          `json:"transactionModule"`
	Sender            string          `json:"sender"`
	Type              string          `json:"type"`
	Fields            map[string]interface{}          `json:"fields"`
}

type ErrorMessage struct { 
	Jsonrpc string                    `json:"jsonrpc"`
	Error   ErrorM                    `json:"error"`
	id      int                       `json:"id"`
}

type ErrorM struct {
	ErrorCode int                     `json:"code"`
	Message   string                  `json:"message"`
}