package blockchain

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	passport "github.com/MoonSHRD/IKY-telegram-bot/artifacts/TGPassport"
	"golang.org/x/exp/slices"

	SingletonNFT "github.com/MoonSHRD/TelegramNFT-Wizard-Contracts/go/SingletonNFT"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
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
}

func NewClient(config Config) (*Client, error) {
	// Connecting to blockchain network
	client, err := ethclient.Dial(config.Gateway) // load from local .env file
	if err != nil {
		return nil, fmt.Errorf("could not connect to Ethereum gateway: %v\n", err)
	}
	defer client.Close()

	// setting up private key in proper format
	privateKey, err := crypto.HexToECDSA(config.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Creating an auth transactor
	auth, _ := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(5))

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

func (client Client) IsRegistered(user_id int64) bool {
	//GetPassportWalletByID
	passport_address, err := client.Passport.GetPassportWalletByID(user_id)
	if err != nil {
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

// SubscribeToCreateItemEvent creates channel which emits one create item event and closes it. In case of error release channel closes
func (client Client) SubscribeToCreateItemEvent(ctx context.Context, fileID string) (chan *SingletonNFT.SingletonNFTItemCreated, error) {
	var ch = make(chan *SingletonNFT.SingletonNFTItemCreated)
	var release = make(chan *SingletonNFT.SingletonNFTItemCreated)

	sub, err := client.Signleton.Contract.WatchItemCreated(&bind.WatchOpts{
		Start:   nil, //last block
		Context: ctx, // nil = no timeout
	}, ch)
	if err != nil {
		return nil, err
	}

	// Basically filtering events until meeting exact file id
	go releaseOnFileID(ch, sub, fileID, release)

	return release, nil
}

// Watches CreateItem event and releases it, after closes channel. In case of error release channel closes
func releaseOnFileID(sink <-chan *SingletonNFT.SingletonNFTItemCreated, sub event.Subscription, file_id string, release chan<- *SingletonNFT.SingletonNFTItemCreated) {
	defer sub.Unsubscribe()
	defer close(release)
	for {
		select {
		case event, ok := <-sink:
			{
				// Checking is channel is closed
				if !ok {
					log.Println("sink is closed")
					return
				}

				// Filtering event with specific file_id
				if event.FileId != file_id {
					continue
				}

				// Recover on release channel closed
				defer func() {
					if r := recover(); r != nil {
						log.Println("failed to release create item event", r)
					}
				}()

				// Release event
				release <- event
				return
			}
		case err := <-sub.Err():
			{
				log.Println("subscription error:", err)
				return
			}
		}
	}
}

func (client Client) FilterCreatedItems(ctx context.Context, from time.Time, fileIDs ...string) ([]string, error) {
	events, err := client.Signleton.Contract.FilterItemCreated(&bind.FilterOpts{
		Start:   uint64(from.Unix()),
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	ids := slices.Clone(fileIDs)
	slices.Sort(ids)
	for events.Next() {
		index, found := slices.BinarySearch(ids, events.Event.FileId)
		if found {
			slices.Delete(ids, index, index+1)
		}
	}

	return ids, nil
}
