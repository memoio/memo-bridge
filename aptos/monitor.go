package aptos

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

const AptosChain = "aptos"
const ConfigPath = "./config.json"

var ratio = big.NewInt(20000000000)

type AptosMonitor struct {
	client *AptosClient
	filename string
	config AptosEvnetConfig
}

type AptosEvnetConfig struct {
	Address string
	EventHandle string
	FieldName string
	Start uint64
	Limit uint64
}

func NewAptosMonitor(url string, timeout time.Duration) (*AptosMonitor) {
	var monitor = &AptosMonitor{
		client: NewAptosClient(url, timeout), 
		config: AptosEvnetConfig{}, 
	}

	return monitor
}

func (monitor *AptosMonitor) Init() error {
	return monitor.readConfig(ConfigPath)
}

func (monitor *AptosMonitor) Start(ctx context.Context) error {
	log.Println("monitor started", monitor.config)
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <- time.After(2 * time.Second):
			events, err := monitor.client.GetEventsByEventHandle(
				childCtx, 
				monitor.config.Address, 
				monitor.config.EventHandle, 
				monitor.config.FieldName, 
				monitor.config.Start, 
				monitor.config.Limit)

			if err != nil {
				log.Println("get event error:", err.Error())
			} else {
				// TODO: keep unhandled event (store on HD) when handle error happen,
				for _, event := range events {
					go handleDepositEvent(childCtx, event)

					if uint64(event.GUID.CreationNumber) >= monitor.config.Start {
						monitor.config.Start = uint64(event.GUID.CreationNumber) + 1
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

func (monitor *AptosMonitor) readConfig(filename string) error {
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

func (monitor *AptosMonitor) writeConfig() error {
	data, err := json.MarshalIndent(monitor.config, "", "\t")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(monitor.filename, data, 0644)
}

func handleDepositEvent(ctx context.Context, event Event) {
	type DepositEvent struct {
		Receiver string      `json:"receiver"`
		Amount Uint64        `json:"amount"`
	}

	var deposit DepositEvent

	data, err := json.Marshal(event.Data)
	if err != nil {
		log.Println("Handle deposit ERROR:", err)
		return
	}

	err = json.Unmarshal(data, &deposit)
	if err != nil {
		log.Println("Handle deposit ERROR:", err)
		return
	}
	log.Println("Handle deposit:", deposit)

	funcSignature := []byte("storeDeposit(address,uint256,string)")

	hash := sha3.NewLegacyKeccak256()
	hash.Write(funcSignature)
	methodID := hash.Sum(nil)[:4]

	amount := big.NewInt(int64(deposit.Amount))
	amount.Mul(amount, ratio)

	// calculate Chain ID (type:string) padded bytes
	paddedChainIDLen := common.LeftPadBytes(big.NewInt(int64(len(AptosChain))).Bytes(), 32)
	paddedChainID := common.RightPadBytes([]byte(AptosChain), 32)

	toAddress := common.HexToAddress(deposit.Receiver)
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

	err = memo.Call(ctx, contractData)
	if err != nil {
		// TODO: Add this event into 
		log.Println("Handle deposit ERROR:", err)
		return
	}
}