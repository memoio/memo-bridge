package main

import (
	"fmt"
	"net/url"
	// "os"
	// "os/signal"
	"sync"
	"encoding/json"

	"github.com/gorilla/websocket"
)

type MoveEventType struct {
	EnventType string    `json:"MoveEventType"`
}

type RequestMessage struct {
	Jsonrpc string           `json:"jsonrpc"`
	Id      int              `json:"id"`
	Method  string           `json:"method"`
	Params  []interface{}    `json:"params"`
}

// Move Event Type defined as {PackageID}::{module name}::{event name}
var depositType = MoveEventType{"0xc8e6eca7ad2434040e9a7a53416cf96cb7ba8763::memo_pool::Deposit"}
var withdrawType = MoveEventType{"0xc8e6eca7ad2434040e9a7a53416cf96cb7ba8763::memo_pool::Withdraw"}

func main() {
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
	err = subscribeEvent(c, []interface{} {depositType}...)
	if err != nil {
		fmt.Println("Subscribe error:", err)
		return
	}

	// subscribe withdraw event
	subscribeEvent(c, []interface{} {withdrawType}...)
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
			fmt.Println("recv:", string(message))
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

func subscribeEvent(c *websocket.Conn, params ...interface{}) error {
	reqMessage := RequestMessage{
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

	_, resbMessage, err := c.ReadMessage()
	if err != nil {
		return err
	}
	fmt.Println("recv:", string(resbMessage))
	return nil
}