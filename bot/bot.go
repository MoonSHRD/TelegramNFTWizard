package bot

import (
	"log"
	"time"

	"github.com/MoonSHRD/TelegramNFTWizard/config"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/blockchain"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/kv"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/wizard"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

type Bot struct {
	*tele.Bot
	kv     *kv.KV
	client *blockchain.Client
}

func New(config config.Config) (*Bot, error) {
	client, err := blockchain.NewClient(config.Config)
	if err != nil {
		return nil, err
	}

	kv, err := kv.New(config.DatabasePath)
	if err != nil {
		return nil, err
	}
	defer kv.Close()

	pref := tele.Settings{
		ParseMode: tele.ModeMarkdownV2,
		Token:     config.Token,
		Poller:    &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return &Bot{
		Bot:    b,
		kv:     kv,
		client: client,
	}, nil
}

func (bot *Bot) Start() {

	bot.Use(middleware.Logger())

	// User first contact with bot
	bot.Handle("/start", bot.StartHandler)

	// When user taping "Create collection"
	bot.Handle(&btnCreate, bot.CreateCollectionHandler)

	// When user is sending NFTs for collection
	bot.Handle(tele.OnDocument, bot.OnDocumentHandler)

	// There should fall all text input steps
	bot.Handle(tele.OnText, bot.OnTextHandler)

	// Skip button handler (currently only for skipping symbol input)
	bot.Handle(&btnSkip, bot.SkipHandler)

	// Final step
	bot.Handle(&btnMinted, bot.MintHandler)

	bot.Bot.Start()
}

// Repeats current state message to user
func remindingResponse(c tele.Context, client *blockchain.Client, user User) error {
	switch user.State {

	case Freeroam:
		if client.IsRegistered(c.Sender().ID) {
			return c.Send(messages["collectionCreation"], menu)
		} else {
			return c.Send(messages["awaitingRegistration"])
		}

	case CollectionPreparation:
		return c.Send(messages["awaitingFiles"], completeFiles)

	case CollectionPreparationName:
		return c.Send(messages["awaitingCollectionName"])

	case CollectionPreparationSymbol:
		return c.Send(messages["awaitingCollectionSymbol"])

	case CollectionMint:
		url, err := wizard.CreateCollectionLink(wizard.CollectionOptions{
			Name:    user.Name,
			Symbol:  &user.Name,
			FileIDs: user.FileIDs,
		})
		if err != nil {
			log.Println("failed to create collection link:", err)
			return c.Send(messages["fail"])
		}

		mint := &tele.ReplyMarkup{}
		mint.URL("Mint", url)
		return c.Send(messages["awaitingCollectionMint"], mint)

	default:
		// ? - figure out proper response
		return nil
	}
}
