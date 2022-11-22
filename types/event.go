package types

// received message type
type EventParams struct {
	Subscription uint64               `json:"subscription"`
	Result       SuiEventEnvelope     `json:"result"`
}

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

// request message type
type MoveEventType struct {
	EnventType string                 `json:"MoveEventType"`
}