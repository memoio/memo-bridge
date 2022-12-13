package sui

import (
	"log"
	"time"
	"context"
	"math/big"
	"io/ioutil"
	"encoding/json"

	"bridge/memo"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
	"golang.org/x/xerrors"
)

const SuiChain = "sui"
var ratio = big.NewInt(2000000000)

type SuiMonitor struct {
	client *SuiClient
	filename string
	config SuiEventConfig
}

type SuiEventConfig struct {
	EventHandle string
	Start EventID
	Limit uint64
}

func NewSuiMonitor(url string, timeout time.Duration) (*SuiMonitor) {
	var monitor = &SuiMonitor{
		client: NewSuiClient(url, timeout), 
		config: SuiEventConfig{}, 
	}

	return monitor
}

func (monitor *SuiMonitor) Init(path string) error {
	return monitor.readConfig(path)
}

func (monitor SuiMonitor) Start(ctx context.Context) error {
	log.Println("monitor started", monitor.config)
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <- time.After(60 * time.Second):
			events, err := monitor.client.GetEventsByMoveEvent(
				childCtx, 
				monitor.config.EventHandle, 
				monitor.config.Start, 
				monitor.config.Limit, 
				false, 
			)

			if err != nil {
				log.Println("get event error:", err.Error())
			} else {
				// TODO: keep unhandled event (store on HD) when handle error happen,
				for _, event := range events {
					if event.ID.TxSeq <= monitor.config.Start.TxSeq {
						continue
					}
					
					go handleDepositEvent(childCtx, event)

					if event.ID.TxSeq >= monitor.config.Start.TxSeq {
						monitor.config.Start.TxSeq = event.ID.TxSeq + 1
					}
				}

				err := monitor.writeConfig()
				if err != nil {
					return xerrors.Errorf("write error:%v, And max creation number[%v]", err, monitor.config.Start)
				}
			}
		case <- ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (monitor *SuiMonitor) readConfig(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &monitor.config)
	if err != nil {
		return err
	}
	monitor.filename = filename

	return json.Unmarshal(data, &monitor.config)
}

func (monitor *SuiMonitor) writeConfig() error {
	data, err := json.MarshalIndent(monitor.config, "", "\t")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(monitor.filename, data, 0644)
}

func handleDepositEvent(ctx context.Context, event SuiEventEnvelope) error {
	type DepositEvent struct {
		Sender string        `json:"sender"`
		Amount uint64        `json:"amount"`
	}

	var deposit DepositEvent
	data, err := json.Marshal(event.Event.MoveEvent.Fields)
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

	amount := big.NewInt(int64(deposit.Amount))
	amount.Mul(amount, ratio)

	// calculate Chain ID (type:string) padded bytes
	paddedChainIDLen := common.LeftPadBytes(big.NewInt(int64(len(SuiChain))).Bytes(), 32)
	paddedChainID := common.RightPadBytes([]byte(SuiChain), 32)

	toAddress := common.HexToAddress(deposit.Sender)
	paddedToAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)
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