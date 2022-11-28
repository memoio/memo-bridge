package main

import (
	"context"
	"fmt"
	"net/url"
	"math/big"
	// "os"
	// "os/signal"
	"sync"
	"encoding/json"

	"bridge/types"
	"bridge/memo"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/sha3"
	"golang.org/x/xerrors"
)

var SuiChain = string("Sui")

// Move Event Type defined as {PackageID}::{module name}::{event name}
var depositType = types.MoveEventType{"0x2365eb5eaa5266f557a36f8de90ae5d932cc1097::memo_pool::Deposit"}
var prepayType = types.MoveEventType{"0x2365eb5eaa5266f557a36f8de90ae5d932cc1097::memo_pool::Prepay"}
var withdrawType = types.MoveEventType{"0x2365eb5eaa5266f557a36f8de90ae5d932cc1097::memo_pool::Withdraw"}

var subscriptions = make(map[uint64]func(event types.SuiEvent) error)

func main() {
	// ctx := context.Background()
	// interrupt := make(chan os.Signal, 1)
	// signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: "103.39.231.220:19001"}
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
	err = subscribeEvent(c, handlePrepayEvent, []interface{} {prepayType}...)
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

	funcSignature := []byte("storeDeposit(address,uint256,string)")

	hash := sha3.NewLegacyKeccak256()
	hash.Write(funcSignature)
	methodID := hash.Sum(nil)[:4]

	toAddress := common.HexToAddress(deposit.Sender)
	paddedToAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	paddedAmount := common.LeftPadBytes(big.NewInt(int64(deposit.Amount)).Bytes(), 32)

	var dataConract []byte
	dataConract = append(dataConract, methodID...)
	dataConract = append(dataConract, paddedToAddress...)
	dataConract = append(dataConract, paddedAmount...)
	dataConract = append(dataConract, []byte(SuiChain)...)

	return memo.Call(context.Background(), dataConract)
}

func handlePrepayEvent(event types.SuiEvent) error {
	fmt.Println("handling:", event)
	type PrepayEvent struct {
		Sender string        `json:"sender"`
		Amount uint64        `json:"amount"`
		Size uint64          `json:"size"`
		Hash string          `json:"hash"`
	}

	var prepay PrepayEvent
	data, err := json.Marshal(event.MoveEvent.Fields)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &prepay)
	if err != nil {
		return err
	}
	
	fmt.Println("prepay:", prepay)

	funcSignature := []byte("storeOrderpay(address,string,uint256,uint256)")

	hash := sha3.NewLegacyKeccak256()
	hash.Write(funcSignature)
	methodID := hash.Sum(nil)[:4]

	toAddress := common.HexToAddress(prepay.Sender)
	paddedToAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	paddedAmount := common.LeftPadBytes(big.NewInt(int64(prepay.Amount)).Bytes(), 32)
	paddedSize := common.LeftPadBytes(big.NewInt(int64(prepay.Amount)).Bytes(), 32)

	var dataConract []byte
	dataConract = append(dataConract, methodID...)
	dataConract = append(dataConract, paddedToAddress...)
	dataConract = append(dataConract, []byte(prepay.Hash)...)
	dataConract = append(dataConract, paddedAmount...)
	dataConract = append(dataConract, paddedSize...)
	
	return memo.Call(context.Background(), dataConract)
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

	funcSignature := []byte("storeWithdraw(address,uint256)")

	hash := sha3.NewLegacyKeccak256()
	hash.Write(funcSignature)
	methodID := hash.Sum(nil)[:4]

	toAddress := common.HexToAddress(withdraw.Receiver)
	paddedToAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	paddedAmount := common.LeftPadBytes(big.NewInt(int64(withdraw.Amount)).Bytes(), 32)

	var dataConract []byte
	dataConract = append(dataConract, methodID...)
	dataConract = append(dataConract, paddedToAddress...)
	dataConract = append(dataConract, paddedAmount...)
	dataConract = append(dataConract, []byte(SuiChain)...)

	return memo.Call(context.Background(), dataConract)
}