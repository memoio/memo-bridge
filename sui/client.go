package sui

import (
	"time"
	"context"
	"encoding/json"

	"github.com/gorilla/websocket"
)

var Subscriptions = make(map[uint64]func(ctx context.Context, event SuiEvent) error)

type SuiClient struct {
	rpcUrl string
	wsUrl string

	rpcClient *http.Client
	wsClient *websocket.Conn
}

func NewSuiClient(rpcUrl string, wsUrl string) (*SuiClient) {
	client = &SuiClient{
		rpcUrl: rpcUrl, 
		wsUrl: wsUrl, 

		rpcClient: nil, 
		wsClient: nil, 
	}

	return client
}

func (client *Client) DailWithContext(ctx context.Context) error {
	wsClient, _, err := websocket.DefaultDialer.DialContext(ctx, client.wsUrl, nil)
	if err != nil {
		return err
	}

	client.wsClient = wsClient
	return nil
}

func (client *Client) Close() {
	if client.wsClient != nil {
		client.wsClient.Close()
		client.wsClient = nil
	}
}

func (client *SuiClient) SubscribeEvent(handle func(ctx context.Context, event types.SuiEvent) error, params ...interface{}) error {
	reqMessage := types.RequestMessage{
		Jsonrpc: "2.0",
		Id:      1,
		Method:  "sui_subscribeEvent",
		Params:  params,
	}
	data, err := json.Marshal(&reqMessage)
	if err != nil {
		return err
	}
	log.Println(string(data))

	err = c.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return err
	}

	_, respBytes, err := c.ReadMessage()
	if err != nil {
		return err
	}
	log.Println("recv:", string(respBytes))

	var respMsg types.RespondMessage

	err = json.Unmarshal(respBytes, &respMsg)
	if err != nil {
		return err
	}
	// regist handle function
	log.Println(respMsg)
	Subscriptions[respMsg.Result] = handle
	return nil
}

func (client *SuiClient) ReadMessage() ([]byte, error) {
	_, message, err:= client.ReadMessage()
	return message, err
}