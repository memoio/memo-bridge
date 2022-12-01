package memo

import(
	"log"
	"time"
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/xerrors"
)

const ENDPOINT string = "https://chain.metamemo.one:8501"
const storeAddr string = "31e7829Ea2054fDF4BCB921eDD3a98a825242267"
const contractAddr string = "Ccf7b7F747100f3393a75DDf6864589f76F4eA25"
const sk string = "8a87053d296a0f0b4600173773c8081b12917cef7419b2675943b0aa99429b62"

const baseGasLimit uint64 = 300000
var baseGasPrice *big.Int = big.NewInt(5000)

func Call(ctx context.Context, data []byte) error {
	client, err := ethclient.Dial(ENDPOINT)
	if err != nil {
		return err
	}
	defer client.Close()

	storeAddress := common.HexToAddress(storeAddr)

	nonce, err := client.PendingNonceAt(ctx, storeAddress)
	if err != nil {
		return err
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return err
	}
	log.Println("chainID: ", chainID)

	contractAddress := common.HexToAddress(contractAddr)

	privateKey, err := crypto.HexToECDSA(sk)
	if err != nil {
		return err
	}

	var tx *types.Transaction
	var signedTx *types.Transaction
	var retry = 10
	var gasLimit = baseGasLimit
	var gasPrice = baseGasPrice
	for {

		if retry == 0 {
			return xerrors.Errorf("Call Contract Failed")
		}
		tx = types.NewTransaction(nonce, contractAddress, big.NewInt(0), gasLimit, gasPrice, data)

		signedTx, err = types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
		if err != nil {
			return err
		}

		err = SendTx(ctx, client, signedTx)
		if err == core.ErrNonceTooLow {
			nonce++
		} else if err == core.ErrNonceTooHigh {
			nonce--
		} else if err == core.ErrIntrinsicGas {
			gasPrice.Add(gasPrice, baseGasPrice)
		} else if err == core.ErrGasLimitReached {
			gasLimit += baseGasLimit
		} else if err != nil {
			return err
		} else {
			break
		}

		retry--
	}

	return nil
}

func SendTx(ctx context.Context, client *ethclient.Client, signedTx *types.Transaction) error {
	err := client.SendTransaction(ctx, signedTx)
	if err != nil {
		return err
	}

	log.Println("waiting tx complete...")
	select {
		case <- ctx.Done():
			return ctx.Err()
		case <- time.After(30 * time.Second):
	}

	receipt, err := client.TransactionReceipt(ctx, signedTx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status != 1 {
		return xerrors.Errorf("Transaction status error [%v]", receipt.Status)
	}

	if len(receipt.Logs) == 0 {
		return xerrors.Errorf("Received messsage from memo but there is no logs")
	}

	if len(receipt.Logs[0].Topics) == 0 {
		return xerrors.Errorf("Received messsage from memo but there is no topics")
	}
	log.Println(receipt.Logs[0].Topics)

	return nil
}

// func fromHex(s string) string {
// 	if s[0] == '0' && s[1] == 'x' {
// 		s = s[2:]
// 	}
// 	if len(s)%2 == 1 {
// 		s = "0" + s
// 	}
// 	return s
// }