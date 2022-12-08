package aptos

import (
	"fmt"
	"time"
	"context"
	"net/http"
	"io/ioutil"
	"encoding/json"

	"golang.org/x/xerrors"
)

type AptosClient struct {
	baseUrl string
	httpClient *http.Client
}

func NewAptosClient(url string, timeout time.Duration) *AptosClient {
	var client = &AptosClient{
		baseUrl: url,
		httpClient: &http.Client{ Timeout: timeout },
	}

	return client
}

func (client *AptosClient) GetEventsByEventHandle(
	ctx context.Context, 
	address string, 
	eventHandle string, 
	fieldName string, 
	start uint64, 
	limit uint64, 
) ([]Event, error) {
    // "https://fullnode.devnet.aptoslabs.com/v1/accounts/'address'/events/'event_handle'/'field_name'?start='start'&limit='limit'"
	url := client.baseUrl + fmt.Sprintf("/v1/accounts/%s/events/%s/%s?start=%d&limit=%d", address, eventHandle, fieldName, start, limit)
	// fmt.Println(url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	// fmt.Println(res)
	// fmt.Println(string(body))

	var errMsg ErrorMessage
	err = json.Unmarshal(body, &errMsg)
	if err == nil && len(errMsg.Message) != 0 && len(errMsg.ErrorCode) != 0 {
		return nil, xerrors.Errorf("Respond code[%d], Error message%s", errMsg.ErrorCode, errMsg.Message)
	}

	var events []Event
	err = json.Unmarshal(body, &events)
	if err != nil {
		return nil, err
	}

	return events, err
}