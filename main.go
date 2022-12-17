package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"os"

	"github.com/joho/godotenv"

	passport "github.com/MoonSHRD/IKY-telegram-bot/artifacts/TGPassport"

	SingletonNFT "github.com/MoonSHRD/TelegramNFT-Wizard-Contracts/go/SingletonNFT"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"

	"github.com/StarkBotsIndustries/telegraph/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//http://t.me/NFT_Wizard_bot
var yesNoKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Yes"),
		tgbotapi.NewKeyboardButton("No")),
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
var ch = make(chan *SingletonNFT.SingletonNFTItemCreated)
//var ch_index = make(chan *passport.PassportPassportAppliedIndexed)



const telegrap_base_url = "https://telegra.ph/"

//main database for dialogs, key (int64) is telegram user id
var userDatabase = make(map[int64]user) // consider to change in persistend data storage?

var msgTemplates = make(map[string]string)



var BASEURL = "https://telegram-nft-wizard.vercel.app/"
var nft_single_url = "createnft/"
var nft_collection_url = "createcollection/"



var myenv map[string]string

// file with settings for enviroment
const envLoc = ".env"

func main() {

	loadEnv()
	ctx := context.Background()
	pk := myenv["PK"] // load private key from env

	msgTemplates["hello"] = "Hey, this bot is allowing you to create NFT"
	msgTemplates["case0"] = "Open following link in metamask broswer"
	msgTemplates["await"] = "Awaiting for NFT mint"
	msgTemplates["case1"] = "Send me a _single_ file which u want to transform into NFT"
	msgTemplates["who_is"] = "Input wallet address to know it's associated telegram nickname"
	msgTemplates["not_registred"] = "You are not registred yet, first attach your wallet to your tg account via this bot https://t.me/E_Passport_bot"



	//var baseURL = "http://localhost:3000/"
	//var baseURL = myenv["BASEURL"];
	var baseURL = "https://telegram-nft-wizard.vercel.app/"



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

	singletonCollection, err := SingletonNFT.NewSingletonNFT(common.HexToAddress(myenv["SINGLETON_ADDRESS"]), client)
	if err != nil {
		log.Fatalf("Failed to instantiate a SingletonNFT contract: %v", err)
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


	//Wrap SingletonNFT contract instance into a session
	session_single_nft := &SingletonNFT.SingletonNFTSession{
		Contract: singletonCollection,
		CallOpts: bind.CallOpts{
			Pending: true,
			From: auth.From,
			Context: context.Background(),
		},
		TransactOpts: bind.TransactOpts{
			From: auth.From,
			Signer: auth.Signer,
			GasLimit: 0,
			GasFeeCap: nil,
			GasTipCap: nil,
			Context: context.Background(),
		},
	}
	log.Printf("session with singleton NFT initialized")

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
							msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, msgTemplates["hello"])
							msg.ReplyMarkup = mainKeyboard
							bot.Send(msg)
							updateDb.dialog_status = 1
							userDatabase[update.Message.From.ID] = updateDb
						}
						
						//msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, msgTemplates["case0"])
						//bot.Send(msg)

	

						//updateDb.dialog_status = 4
						//userDatabase[update.Message.From.ID] = updateDb
						
					}
				//	fallthrough //
				// choose option
				case 1:
					if updateDb, ok := userDatabase[update.Message.From.ID]; ok {
						msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, msgTemplates["case1"])
						//msg.ReplyMarkup = optionKeyboard
						bot.Send(msg)
						updateDb.dialog_status = 2
						userDatabase[update.Message.From.ID] = updateDb
						
					}
				// download file from user and upload to telegraph
				case 2:
					if updateDb, ok := userDatabase[update.Message.From.ID]; ok {
						 
						if update.Message.Document != nil && update.Message.Document.FileSize <= 5e+6 {
							caption := update.Message.Caption
							file_id := update.Message.Document.FileID
							u_file_id := update.Message.Document.FileUniqueID
							file_name :=update.Message.Document.FileName
							file_type := update.Message.Document.MimeType
							file_size := update.Message.Document.FileSize
							file_size_string := strconv.Itoa(file_size)
							msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, "unique_file id is:" + u_file_id)
							//msg.ReplyMarkup = 
							bot.Send(msg)
							
							log.Println("caption:" + caption)
							log.Println("unique file id: " + u_file_id)
							log.Println("file id: " + file_id)
							log.Println("file_type " + file_type)
							log.Println("file size os: " + file_size_string)
							
							direct_url, err := bot.GetFileDirectURL(file_id)
							if err != nil {
								log.Println(err)
							}
							
							// download a file
							file := createFile(file_name)
							GetFile(file,httpClient(),direct_url,file_name)
							telegraph_link := UploadFileToTelegraph(file_name)

							

							
							fmt.Println(direct_url)
							log.Println(direct_url)


							//bot.UploadFiles("https://telegra.ph/upload",tgbotapi.Params{},[]tgbotapi.RequestFile{})
							msg = tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, "telegraph URL is:" + telegrap_base_url + telegraph_link)
							bot.Send(msg)
							log.Println("telegraph url: " + telegrap_base_url + telegraph_link)
							// remove file locally after upload 
							deleteFile(file_name)

							uri_short := strings.Split(telegraph_link,"/")


							// create link to mint NFT
							createLink(userDatabase[update.Message.From.ID].tgid,uri_short[2],baseURL)

							subscription, err := SubscribeForCreateItem(session_single_nft, ch) // this is subscription to UNINDEXED event. 
							if err != nil {
								log.Println(err)
							}

							go AsyncCreateItemListener(ctx,subscription,userDatabase[update.Message.From.ID].tgid,auth,userDatabase)
							
							updateDb.dialog_status = 3
							userDatabase[update.Message.From.ID] = updateDb
						} else {
							msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, "you should send me a file AS A DOCUMENT and it should be less then 5mbytes")
							bot.Send(msg)
							updateDb.dialog_status = 2
							userDatabase[update.Message.From.ID] = updateDb
						}
					}

				// await for mint
				case 3:
					if updateDb, ok := userDatabase[update.Message.From.ID]; ok {


						msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, msgTemplates["await"])
						//msg.ReplyMarkup = optionKeyboard
						bot.Send(msg)

						updateDb.dialog_status = 3
						userDatabase[update.Message.From.ID] = updateDb

					}

				// mint successfull(?)
				case 4:
					if updateDb, ok := userDatabase[update.Message.From.ID]; ok {
						msg := tgbotapi.NewMessage(userDatabase[update.Message.From.ID].tgid, "Your NFT is succesfully minted, you will redirect to main menu")
						//msg.ReplyMarkup = optionKeyboard
						bot.Send(msg)
						updateDb.dialog_status = 1
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

func createLink(tgid int64,file_name string, base_url string)  {
	
	msg := tgbotapi.NewMessage(tgid, "Open following link in metamask browser and click CreateNFT!")
	bot.Send(msg)
	link := base_url + nft_single_url + "?file_id=" + file_name
	msg = tgbotapi.NewMessage(userDatabase[tgid].tgid, link)
	bot.Send(msg)
}

func createLinkCollection (tgid int64, file_names []string) {
	msg := tgbotapi.NewMessage(tgid, "Open following link in metamask browser and click CreateNFT!")
	bot.Send(msg)
	link := BASEURL + nft_single_url + "?file_id="
	lenght := len(file_names)
	for i := 0; i< lenght; i++{
		link = link + "?file_id="+ file_names[i]
	}
	msg = tgbotapi.NewMessage(userDatabase[tgid].tgid, link)
	bot.Send(msg)
}

// download file
func GetFile(file *os.File, client *http.Client, url string, file_name string) {
    resp, err := client.Get(url)
    checkError(err)
    defer resp.Body.Close()
    size, err := io.Copy(file, resp.Body)
    defer file.Close()
    checkError(err)

    fmt.Println("Just Downloaded a file %s with size %d", file_name, size)
}


func UploadFileToTelegraph(file_name string) (string)  {

	file, _ := os.Open(file_name)
	// os.File is io.Reader so just pass it.
	link, _ := telegraph.Upload(file, "photo")
	log.Println(link)
	return link
}





func httpClient() *http.Client {
    client := http.Client{
        CheckRedirect: func(r *http.Request, via []*http.Request) error {
            r.URL.Opaque = r.URL.Path
            return nil
        },
    }

    return &client
}

// create blank file
func createFile(file_name string) *os.File {
    file, err := os.Create(file_name)

    checkError(err)
    return file
}

// delete file locally
func deleteFile(file_name string) (bool,error) {
	err := os.Remove(file_name)
	if err != nil {
		return false, err
	} else {
		return true,nil
	}
}

func checkError(err error) {
    if err != nil {
        panic(err)
    }
}

// subscribing for CreateItem events. We use watchers without fast-forwarding past events
func SubscribeForCreateItem(session *SingletonNFT.SingletonNFTSession, listenChannel chan<- *SingletonNFT.SingletonNFTItemCreated) (event.Subscription, error) {
	subscription, err := session.Contract.WatchItemCreated(&bind.WatchOpts{
		Start:   nil, //last block
		Context: nil, // nil = no timeout
	}, listenChannel,
//		applierTGID,
	)
	if err != nil {
		return nil, err
	}
	return subscription, err
}

func AsyncCreateItemListener(ctx context.Context,subscription event.Subscription, tgid int64, auth *bind.TransactOpts, userDatabase map[int64]user) {
	EventLoop:
						for {
							select {
							case <-ctx.Done():
								{
									subscription.Unsubscribe()
									break EventLoop
								}
							case eventResult := <-ch:
								{
									fmt.Println("NFT token ID:", eventResult.TokenId)
									fmt.Println("NFT collection address:", eventResult.Raw.Address)
									fmt.Println("File_ID string: ", eventResult.FileId)
									
									msg := tgbotapi.NewMessage(userDatabase[tgid].tgid, " your NFT token has been created, token ID is: " + eventResult.TokenId.String())
									bot.Send(msg)
									msg = tgbotapi.NewMessage(tgid,"address of NFT collection (add it to metamask): " + eventResult.Raw.Address.Hex())
									bot.Send(msg)
									subscription.Unsubscribe()
									break EventLoop
								}
								}
						}
						updateDb := userDatabase[tgid]
						updateDb.dialog_status = 4
						userDatabase[tgid] = updateDb
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

