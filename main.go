package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"

	"os"

	"github.com/joho/godotenv"

	passport "github.com/MoonSHRD/IKY-telegram-bot/artifacts/TGPassport"
	//passport "IKY-telegram-bot/artifacts/TGPassport"

	//SingletonNFT "github.com/MoonSHRD/TelegramNFT-Wizard-Contracts/go/SingletonNFT"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//http://t.me/NFT_Wizard_bot
var yesNoKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Yes"),
		tgbotapi.NewKeyboardButton("No")),
)

var optionKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("WhoIs"),),
)


var mainKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("CreateNFT")),
)

//to operate the bot, put a text file containing key for your bot acquired from telegram "botfather" to the same directory with this file
var tgApiKey, err = os.ReadFile(".secret")
var bot, error1 = tgbotapi.NewBotAPI(string(tgApiKey))

//type containing all the info about user input
type user struct {
	tgid          int64
	tg_username   string
	dialog_status int64
}





// channel to get this event from blockchain
var ch = make(chan *passport.PassportPassportApplied)
var ch_index = make(chan *passport.PassportPassportAppliedIndexed)

var ch_approved = make(chan *passport.PassportPassportApproved)

//main database for dialogs, key (int64) is telegram user id
var userDatabase = make(map[int64]user) // consider to change in persistend data storage?

var msgTemplates = make(map[string]string)


var tg_id_query = "?user_tg_id="
var tg_username_query = "&user_tg_name="

var myenv map[string]string

// file with settings for enviroment
const envLoc = ".env"

func main() {

	loadEnv()
	ctx := context.Background()
	pk := myenv["PK"] // load private key from env

	msgTemplates["hello"] = "Hey, this bot is allowing you to create NFT"
	msgTemplates["case0"] = "Open following link in metamask broswer"
	msgTemplates["await"] = "Awaiting for verification"
	msgTemplates["case1"] = "Send me a _single_ file which u want to transform into NFT"
	msgTemplates["who_is"] = "Input wallet address to know it's associated telegram nickname"
	msgTemplates["not_registred"] = "You are not registred yet, first attach your wallet to your tg account via this bot https://t.me/E_Passport_bot"



	//var baseURL = "http://localhost:3000/"
	//var baseURL = "https://ikytest-gw0gy01is-s0lidarnost.vercel.app/"
	//var baseURL = myenv["BASEURL"];



	bot, err = tgbotapi.NewBotAPI(string(tgApiKey))
	if err != nil {
		log.Panic(err)
	}

	// Connecting to blockchain network
	//  client, err := ethclient.Dial(os.Getenv("GATEWAY"))	// for global env config
	client, err := ethclient.Dial(myenv["GATEWAY_GOERLI_WS"]) // load from local .env file
	if err != nil {
		log.Fatalf("could not connect to Ethereum gateway: %v\n", err)
	}
	defer client.Close()

	// setting up private key in proper format
	privateKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		log.Fatal(err)
	}

	// Creating an auth transactor
	auth, _ := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(5))

	// check calls
	// check balance
	accountAddress := common.HexToAddress(myenv["ACCOUNT_ADDRESS"])
	balance, _ := client.BalanceAt(ctx, accountAddress, nil) //our balance
	fmt.Printf("Balance of the validator bot: %d\n", balance)

	// Setting up Passport Contract
	passportCenter, err := passport.NewPassport(common.HexToAddress(myenv["PASSPORT_ADDRESS"]), client)
	if err != nil {
		log.Fatalf("Failed to instantiate a TGPassport contract: %v", err)
	}

	// Wrap the Passport contract instance into a session
	session := &passport.PassportSession{
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

	log.Printf("session with passport center initialized")

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	//whenever bot gets a new message, check for user id in the database happens, if it's a new user, the entry in the database is created.
	for update := range updates {

		if update.Message != nil {
			if _, ok := userDatabase[update.Message.From.ID]; !ok {

				userDatabase[update.Message.From.ID] = user{update.Message.Chat.ID, update.Message.Chat.UserName, 0}
				msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, msgTemplates["hello"])
				msg.ReplyMarkup = mainKeyboard
				bot.Send(msg)
				// check for registration
				registred := IsAlreadyRegistred(session,update.Message.From.ID)
				if registred == true {
					userDatabase[update.Message.From.ID] = user{update.Message.Chat.ID, update.Message.Chat.UserName, 1}
				}

			} else {

				switch userDatabase[update.Message.From.ID].dialog_status {

				//first check for user status, (for a new user status 0 is set automatically), then user reply for the first bot message is logged to a database as name AND user status is updated
				case 0:
					if updateDb, ok := userDatabase[update.Message.From.ID]; ok {
						registred := IsAlreadyRegistred(session,update.Message.From.ID)
						if registred == false {
						msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, msgTemplates["not_registred"])
						bot.Send(msg)
						updateDb.dialog_status = 0
						userDatabase[update.Message.From.ID] = updateDb
						} else {
							updateDb.dialog_status = 1
							userDatabase[update.Message.From.ID] = updateDb
						}
						
						//msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, msgTemplates["case0"])
						//bot.Send(msg)

						/*
						tgid := userDatabase[update.Message.From.ID].tgid
						user_name := userDatabase[update.Message.From.ID].tg_username
						fmt.Println(user_name)
						tgid_string := fmt.Sprint(tgid)
						tgid_array := make([]int64, 1)
						tgid_array[0] = tgid
						link := baseURL + tg_id_query + tgid_string + tg_username_query + "@" + user_name
						msg = tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, link)
						bot.Send(msg)
						*/

					
						

						//updateDb.dialog_status = 4
						//userDatabase[update.Message.From.ID] = updateDb
						
					}
				//	fallthrough // МЫ ЛЕД ПОД НОГАМИ МАЙОРА!
				case 1:
					if updateDb, ok := userDatabase[update.Message.From.ID]; ok {
						msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, msgTemplates["case1"])
						msg.ReplyMarkup = optionKeyboard
						bot.Send(msg)
						updateDb.dialog_status = 2
						userDatabase[update.Message.From.ID] = updateDb
						
					}

				case 2:
					if updateDb, ok := userDatabase[update.Message.From.ID]; ok {
						if update.Message.Document ==  *&update.Message.Document {
							caption := update.Message.Caption
							//file_id := update.Message.Document.FileID
							//u_file_id := update.Message.Document.FileUniqueID
							file_name :=update.Message.Document.FileName
							file_type := update.Message.Document.MimeType
							file_size := update.Message.Document.FileSize
							file_size_string := strconv.Itoa(file_size)
							//msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, "unique_file id is:" + u_file_id)
							//msg.ReplyMarkup = 
							//bot.Send(msg)
							msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, "caption is:" + caption)
							bot.Send(msg)
							msg = tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, "file_name is:" + file_name)
							bot.Send(msg)
							msg = tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, "file_type is:" + file_type)
							bot.Send(msg)
							msg = tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, "file_size is:" + file_size_string)
							bot.Send(msg)
							updateDb.dialog_status = 3
							userDatabase[update.Message.From.ID] = updateDb
						} else {
							msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, "you should send me a file")
							bot.Send(msg)
							updateDb.dialog_status = 2
							userDatabase[update.Message.From.ID] = updateDb
						}
					}

				// whois
				case 3:
					if updateDb, ok := userDatabase[update.Message.From.ID]; ok {
						updateDb.dialog_status = 3
						userDatabase[update.Message.From.ID] = updateDb

					}

				// 
				case 4:
					if updateDb, ok := userDatabase[update.Message.From.ID]; ok {
						updateDb.dialog_status = 3
						userDatabase[update.Message.From.ID] = updateDb
					}


				}
			}
		}
	}

} // end of main func

// load enviroment variables from .env file
func loadEnv() {
	var err error
	if myenv, err = godotenv.Read(envLoc); err != nil {
		log.Printf("could not load env from %s: %v", envLoc, err)
	}
}





// allow bot to get tg nickname associated with this eth wallet
func WhoIsAddress(session *passport.PassportSession,address_to_check common.Address) (string,error){
	passport, err := session.GetPassportByAddress(address_to_check)
	if err != nil {
		log.Println("cant get passport associated with this address, possible it's not registred yet: ")
		log.Print(err)
		 return "error",err
	}
	nickname := passport.UserName
	return nickname,nil

}

func IsAlreadyRegistred(session *passport.PassportSession, user_id int64) (bool) {
	//GetPassportWalletByID
	passport_address, err := session.GetPassportWalletByID(user_id)
	if err != nil {
		return false
	}
	log.Println("check that user with this id:", user_id)
	log.Println("have associated wallet address:",passport_address)
	if passport_address == common.HexToAddress("0x0000000000000000000000000000000000000000") {
		log.Println("passport is null, user is not registred")
		return false
	} else {
		return true
	}
}

