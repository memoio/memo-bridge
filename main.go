package main

import (
	"context"
	"fmt"
	"net/url"
	// "os"
	// "os/signal"
	"sync"
	"encoding/json"

	"bridge/types"
	"bridge/memo"

	"github.com/gorilla/websocket"
	"golang.org/x/xerrors"
)

// Move Event Type defined as {PackageID}::{module name}::{event name}
var depositType = types.MoveEventType{"0xb07780c67810a5099461bece8e2fbcfd6dc3f20d::memo_pool::Deposit"}
var withdrawType = types.MoveEventType{"0xb07780c67810a5099461bece8e2fbcfd6dc3f20d::memo_pool::Withdraw"}

var subscriptions = make(map[uint64]func(event types.SuiEvent) error)

func main() {
	// ctx := context.Background()
	// interrupt := make(chan os.Signal, 1)
	// signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: "0.0.0.0:9002"}
	fmt.Println("connecting to ", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Println("dial error:", err)
	}
	defer c.Close()

	// subscribe deposit event
	err = subscribeEvent(c, handleDepositEvent, []interface{} {depositType}...)
	if err != nil {
		fmt.Println("Subscribe error:", err)
		return
	}

	// subscribe withdraw event
	err = subscribeEvent(c, handleWithdrawEvent, []interface{} {withdrawType}...)
	if err != nil {
		fmt.Println("Subscribe error:", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			_, message, err := c.ReadMessage()
			// TODO: receive close message and return
			if err != nil {
				fmt.Println("read:", err)
				return
			}
			fmt.Println(string(message))

			err = handleEventMessage(message)
			if err != nil {
				fmt.Println("handle event message:", err)
				return
			}
		}
	}()

	// for {
	// 	switch {
	// 	case <- interrupt:
	// 		// TODO: send close msg and wait
	// 	}
	// }
	wg.Wait()
}

func subscribeEvent(c *websocket.Conn, handle func(event types.SuiEvent) error, params ...interface{}) error {
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
	fmt.Println(string(data))

	err = c.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return err
	}

	_, respBytes, err := c.ReadMessage()
	if err != nil {
		return err
	}
	fmt.Println("recv:", string(respBytes))

	var respMsg types.RespondMessage

	err = json.Unmarshal(respBytes, &respMsg)
	if err != nil {
		return err
	}
	// regist handle function
	fmt.Println(respMsg)
	subscriptions[respMsg.Result] = handle
	return nil
}

func handleEventMessage(message []byte) error {
	var parsed types.EventMessage
	err := json.Unmarshal(message, &parsed)
	if err != nil {
		return err
	}

	handle, ok := subscriptions[parsed.Params.Subscription]
	if !ok {
		return xerrors.Errorf("Unexpect Message, don't regist subscription[%v]", parsed.Params.Subscription)
	}

	// event id
	// eventID := parsed.Params.Result.ID

	return handle(parsed.Params.Result.Event)
}

func handleDepositEvent(event types.SuiEvent) error {
	fmt.Println("handling:", event)
	type DepositEvent struct {
		Sender string        `json:"sender"`
		Amount uint64        `json:"amount"`
	}

	var deposit DepositEvent
	data, err := json.Marshal(event.MoveEvent.Fields)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &deposit)
	if err != nil {
		return err
	}

	fmt.Println("Deposit:", deposit)

	return memo.Transfer(context.Background(), deposit.Sender, deposit.Amount)
}

func handleWithdrawEvent(event types.SuiEvent) error {
	fmt.Println("handling:", event)
	type WithdrawEvent struct {
		Receiver string      `json:"receiver"`
		Amount uint64        `json:"amount"`
	}

	var withdraw WithdrawEvent
	data, err := json.Marshal(event.MoveEvent.Fields)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &withdraw)
	if err != nil {
		return err
	}
	
	fmt.Println("Withdraw:", withdraw)

	return memo.Transfer(context.Background(), withdraw.Receiver, withdraw.Amount)
}