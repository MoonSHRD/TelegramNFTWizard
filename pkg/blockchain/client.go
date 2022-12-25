package blockchain

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	passport "github.com/MoonSHRD/IKY-telegram-bot/artifacts/TGPassport"

	SingletonNFT "github.com/MoonSHRD/TelegramNFT-Wizard-Contracts/go/SingletonNFT"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	Passport  *passport.PassportSession
	Signleton *SingletonNFT.SingletonNFTSession
}

type Config struct {
	PrivateKey       string `env:"PRIVATE_KEY,notEmpty"`
	Gateway          string `env:"GATEWAY,notEmpty"`
	AccountAddress   string `env:"ACCOUNT_ADDRESS,notEmpty"`
	PassportAddress  string `env:"PASSPORT_ADDRESS,notEmpty"`
	SingletonAddress string `env:"SINGLETON_ADDRESS,notEmpty"`
	FactoryAddress   string `env:"FACTORY_ADDRESS,notEmpty"`
}

func NewClient(config Config) (*Client, error) {
	// Connecting to blockchain network
	client, err := ethclient.Dial(config.Gateway) // load from local .env file
	if err != nil {
		return nil, fmt.Errorf("could not connect to Ethereum gateway: %v\n", err)
	}

	// setting up private key in proper format
	privateKey, err := crypto.HexToECDSA(config.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Creating an auth transactor
	auth, _ := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(137))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(3))
	accountAddress := common.HexToAddress(config.AccountAddress)
	balance, err := client.BalanceAt(ctx, accountAddress, nil) //our balance
	cancel()
	if err != nil {
		return nil, err
	}

	log.Printf("Balance of the validator bot: %d\n", balance)

	// Setting up Passport Contract
	passportCenter, err := passport.NewPassport(common.HexToAddress(config.PassportAddress), client)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate a TGPassport contract: %v", err)
	}

	singletonCollection, err := SingletonNFT.NewSingletonNFT(common.HexToAddress(config.SingletonAddress), client)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate a SingletonNFT contract: %v", err)
	}

	// Wrap the Passport contract instance into a session
	passport := &passport.PassportSession{
		Contract: passportCenter,
		CallOpts: bind.CallOpts{
			Pending: true,
			From:    auth.From,
			Context: context.Background(),
		},
		TransactOpts: bind.TransactOpts{
			From:     auth.From,
			Signer:   auth.Signer,
			GasLimit: 0,   // 0 automatically estimates gas limit
			GasPrice: nil, // nil automatically suggests gas price
			Context:  context.Background(),
		},
	}

	//Wrap SingletonNFT contract instance into a session
	singleton := &SingletonNFT.SingletonNFTSession{
		Contract: singletonCollection,
		CallOpts: bind.CallOpts{
			Pending: true,
			From:    auth.From,
			Context: context.Background(),
		},
		TransactOpts: bind.TransactOpts{
			From:      auth.From,
			Signer:    auth.Signer,
			GasLimit:  0,
			GasFeeCap: nil,
			GasTipCap: nil,
			Context:   context.Background(),
		},
	}

	return &Client{
		Passport:  passport,
		Signleton: singleton,
	}, nil
}

func (client *Client) IsRegistered(user_id int64) bool {
	passport_address, err := client.Passport.GetPassportWalletByID(user_id)
	if err != nil {
		log.Println(err)
		return false
	}

	log.Println("check that user with this id:", user_id)
	log.Println("have associated wallet address:", passport_address)

	if passport_address == common.HexToAddress("0x0000000000000000000000000000000000000000") {
		log.Println("passport is null, user is not registred")
		return false
	} else {
		return true
	}
}

func (client *Client) CheckItemCreated(ctx context.Context, fileID string, start time.Time) (bool, error) {
	iter, err := client.Signleton.Contract.FilterItemCreated(&bind.FilterOpts{
		Start: uint64(start.Unix()),
	}, []string{fileID})
	if err != nil {
		return false, err
	}

	return iter.Next(), iter.Error()
}

// Returns count of remaining items, if remaining == 0 all files was created.
func (client *Client) CheckItemsCreated(ctx context.Context, fileIDs []string, start time.Time) (int, error) {
	count := len(fileIDs)
	iter, err := client.Signleton.Contract.FilterItemCreated(&bind.FilterOpts{
		Start: uint64(start.Unix()),
	}, fileIDs)
	if err != nil {
		return count, err
	}

	for iter.Next() {
		count -= 1
	}
	return count, iter.Error()
}
