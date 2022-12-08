package sui

import (
	"log"
	"time"
	"context"
	"math/big"
	"encoding/json"

	"bridge/memo"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
	"golang.org/x/xerrors"
)

var depositType = MoveEventType{"0xcce4b28a1b62cf5e6a1a1b756dad67a2aa0ef762::memo_pool::Deposit"}

type SuiMonitor struct {
	client *SuiClient
}

func NewSuiMonitor(rpcUrl string, wsUrl string) SuiMonitor {
	client = NewSuiClient(rpcUrl, wsUrl)

	return SuiMonitor{ client: client }
}

func (monitor SuiMonitor) Start(ctx context.Context) {
	err := monitor.client.DialWithContext(ctx)
	if err != nil {
		log.Println(err)
		return
	}
	defer monitor.client.Close()

	err := monitor.client.SubscribeEvent(handleDepositEvent, depositType)
	if err != nil {
		log.Println("Subscribe error:", err)
		return
	}

	chlidCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	
	for {
		message, err := monitor.client.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}

		go handleEventMessage(chlidCtx, message)
	}
}

func handleEventMessage(ctx context.Context, message []byte) {
	var parsed EventMessage
	err := json.Unmarshal(message, &parsed)
	if err != nil {
		log.Println(err)
		return
	}

	handle, ok := Subscriptions[parsed.Params.Subscription]
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