package memo

import(
	"log"
	"time"
	"context"
	"math/big"
	// "encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/xerrors"
)

const ENDPOINT string = "https://chain.metamemo.one:8501"
const sk string = "8a87053d296a0f0b4600173773c8081b12917cef7419b2675943b0aa99429b62"
const caddr string = "1EcF7e4E263C3F16194C22eb4460af6E0E8aDF61"

func Call(ctx context.Context, data []byte) error {
	client, err := ethclient.Dial(ENDPOINT)
	if err != nil {
		return err
	}
	defer client.Close()

	contractAddress := common.HexToAddress(caddr)

	nonce, err := client.PendingNonceAt(ctx, contractAddress)
	if err != nil {
		return err
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return err
	}
	log.Println("chainID: ", chainID)

	gasLimit := uint64(300000)
	gasPrice := big.NewInt(1000)

	tx := types.NewTransaction(nonce, contractAddress, big.NewInt(0), gasLimit, gasPrice, data)
	privateKey, err := crypto.HexToECDSA(sk)
	if err != nil {
		return err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return err
	}

	return SendTx(ctx, client, signedTx)
}

func SendTx(ctx context.Context, client *ethclient.Client, signedTx *types.Transaction) error {
	err := client.SendTransaction(ctx, signedTx)
	if err != nil {
		return err
	}

	log.Println("waiting tx complete...")
	time.Sleep(30 * time.Second)

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

func fromHex(s string) string {
	if s[0] == '0' && s[1] == 'x' {
		s = s[2:]
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return s
}