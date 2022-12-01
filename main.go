package main

import (
	"context"
	"log"
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

var subscriptions = make(map[uint64]func(ctx context.Context, event types.SuiEvent) error)

func main() {
	// interrupt := make(chan os.Signal, 1)
	// signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: "103.39.231.220:19001"}
	log.Println("connecting to ", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("dial error:", err)
	}
	defer c.Close()

	// subscribe deposit event
	err = subscribeEvent(c, handleDepositEvent, []interface{} {depositType}...)
	if err != nil {
		log.Println("Subscribe error:", err)
		return
	}

	// subscribe withdraw event
	err = subscribeEvent(c, handlePrepayEvent, []interface{} {prepayType}...)
	if err != nil {
		log.Println("Subscribe error:", err)
		return
	}

	// subscribe withdraw event
	err = subscribeEvent(c, handleWithdrawEvent, []interface{} {withdrawType}...)
	if err != nil {
		log.Println("Subscribe error:", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			ctx, cancle := context.WithCancel(context.Background())
			_, message, err := c.ReadMessage()
			// TODO: receive close message and return
			if err != nil {
				log.Println("read:", err)
				cancle()
				return
			}

			go handleEventMessage(ctx, message)
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

func subscribeEvent(c *websocket.Conn, handle func(ctx context.Context, event types.SuiEvent) error, params ...interface{}) error {
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
	subscriptions[respMsg.Result] = handle
	return nil
}

func handleEventMessage(ctx context.Context, message []byte) {
	var parsed types.EventMessage
	err := json.Unmarshal(message, &parsed)
	if err != nil {
		log.Println(err)
		return
	}

	handle, ok := subscriptions[parsed.Params.Subscription]
	if !ok {
		log.Println(xerrors.Errorf("Unexpect Message, don't regist subscription[%v]", parsed.Params.Subscription))
		return
	}

	// event id
	// eventID := parsed.Params.Result.ID

	err = handle(ctx, parsed.Params.Result.Event)
	if err != nil {
		log.Println(err)
	}
	return
}

func handleDepositEvent(ctx context.Context, event types.SuiEvent) error {
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

	log.Println("Deposit:", deposit)

	// 大概率string的问题
	funcSignature := []byte("storeDeposit(address,uint256,string)")

	hash := sha3.NewLegacyKeccak256()
	hash.Write(funcSignature)
	methodID := hash.Sum(nil)[:4]

	// calculate Chain ID (type:string) padded bytes
	paddedChainIDLen := common.LeftPadBytes(big.NewInt(int64(len(SuiChain))).Bytes(), 32)
	paddedChainID := common.RightPadBytes([]byte(SuiChain), 32)

	toAddress := common.HexToAddress(deposit.Sender)
	paddedToAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	paddedAmount := common.LeftPadBytes(big.NewInt(int64(deposit.Amount)).Bytes(), 32)
	paddedChainIDOffset := common.LeftPadBytes(big.NewInt(32 * 3).Bytes(), 32)

	var contractData []byte
	contractData = append(contractData, methodID...)
	contractData = append(contractData, paddedToAddress...)
	contractData = append(contractData, paddedAmount...)
	contractData = append(contractData, paddedChainIDOffset...)
	contractData = append(contractData, paddedChainIDLen...)
	contractData = append(contractData, paddedChainID...)

	return memo.Call(ctx, contractData)
}

func handlePrepayEvent(ctx context.Context, event types.SuiEvent) error {
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
	
	log.Println("prepay:", prepay)

	funcSignature := []byte("storeOrderpay(address,string,uint256,uint256)")

	hash := sha3.NewLegacyKeccak256()
	hash.Write(funcSignature)
	methodID := hash.Sum(nil)[:4]

	if len(prepay.Hash) != 32 {
		return xerrors.Errorf("Hash size is not correct")
	}
	paddedHashLen := common.LeftPadBytes(big.NewInt(int64(len(prepay.Hash))).Bytes(), 32)
	paddedHash := common.RightPadBytes([]byte(prepay.Hash), 32)

	toAddress := common.HexToAddress(prepay.Sender)
	paddedToAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	paddedHashOffset := common.LeftPadBytes(big.NewInt(32 * 4).Bytes(), 32)
	paddedAmount := common.LeftPadBytes(big.NewInt(int64(prepay.Amount)).Bytes(), 32)
	paddedSize := common.LeftPadBytes(big.NewInt(int64(prepay.Size)).Bytes(), 32)

	var contractData []byte
	contractData = append(contractData, methodID...)
	contractData = append(contractData, paddedToAddress...)
	contractData = append(contractData, paddedHashOffset...)
	contractData = append(contractData, paddedAmount...)
	contractData = append(contractData, paddedSize...)
	contractData = append(contractData, paddedHashLen...)
	contractData = append(contractData, paddedHash...)
	
	return memo.Call(ctx, contractData)
}

func handleWithdrawEvent(ctx context.Context, event types.SuiEvent) error {
	log.Println("handling:", event)
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
	
	log.Println("Withdraw:", withdraw)

	funcSignature := []byte("storeWithdraw(address,uint256,string)")

	hash := sha3.NewLegacyKeccak256()
	hash.Write(funcSignature)
	methodID := hash.Sum(nil)[:4]

	// calculate Chain ID (type:string) padded bytes
	paddedChainIDLen := common.LeftPadBytes(big.NewInt(int64(len(SuiChain))).Bytes(), 32)
	paddedChainID := common.RightPadBytes([]byte(SuiChain), 32)

	toAddress := common.HexToAddress(withdraw.Receiver)
	paddedToAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	paddedAmount := common.LeftPadBytes(big.NewInt(int64(withdraw.Amount)).Bytes(), 32)
	paddedChainIDOffset := common.LeftPadBytes(big.NewInt(32 * 3).Bytes(), 32)

	var contractData []byte
	contractData = append(contractData, methodID...)
	contractData = append(contractData, paddedToAddress...)
	contractData = append(contractData, paddedAmount...)
	contractData = append(contractData, paddedChainIDOffset...)
	contractData = append(contractData, paddedChainIDLen...)
	contractData = append(contractData, paddedChainID...)

	return memo.Call(ctx, contractData)
}