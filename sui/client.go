package sui

import (
	"time"
	"bytes"
	"context"
	"net/http"
	"io/ioutil"
	"encoding/json"

	"golang.org/x/xerrors"
)

type SuiClient struct {
	baseUrl string
	httpClient *http.Client
}

func NewSuiClient(url string, timeout time.Duration) *SuiClient {
	var client = &SuiClient{
		baseUrl: url,
		httpClient: &http.Client{ Timeout: timeout },
	}

	return client
}

func (client *SuiClient) GetEventsByMoveEvent(
	ctx context.Context, 
	moveEvent string, 
	eventID EventID,  
	limit uint64, 
	order bool, 
) ([]SuiEventEnvelope, error) {
	reqMessage := RequestMessage{
		Jsonrpc: "2.0",
		Id:      1,
		Method:  "sui_getEvents",
		Params:  []interface{} {
			MoveEventType{ moveEvent }, 
			eventID, 
			limit, 
			order, 
		},
	}
	data, err := json.Marshal(&reqMessage)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", client.baseUrl, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	res, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body) 
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("Respond code[%d]: %s", res.StatusCode, string(body))
	}

	var errMsg ErrorMessage
	err = json.Unmarshal(body, &errMsg)
	if err == nil && len(errMsg.Error.Message) != 0 && errMsg.Error.ErrorCode != 0 {
		return nil, xerrors.Errorf("Respond code[%d], Error message%s", errMsg.Error.ErrorCode, errMsg.Error.Message)
	}

	var resp RespondMessage
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Result.Events, err
}