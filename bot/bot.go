package bot

import (
	"time"

	"github.com/MoonSHRD/TelegramNFTWizard/config"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/blockchain"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/kv"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

type Bot struct {
	*tele.Bot
	kv            *kv.KV
	client        *blockchain.Client
	createdAt     int64
	subscriptions map[string]*blockchain.Subscription
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

	pref := tele.Settings{
		// ParseMode: tele.ModeMarkdownV2,
		Token:  config.Token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return &Bot{
		Bot:           b,
		kv:            kv,
		client:        client,
		createdAt:     time.Now().Unix(),
		subscriptions: make(map[string]*blockchain.Subscription),
	}, nil
}

func (bot *Bot) Start() {
	// All handles are asynchronous, keep it in mind

	// Just prints whole message update struct to log, suitable for debug
	bot.Use(middleware.Logger())

	// User first contact with bot
	bot.Handle("/start", bot.StartHandler)

	// When user taping "Create item"
	bot.Handle(&btnCreateItem, bot.CreateItemHandler)

	// When user taping "Create collection"
	bot.Handle(&btnCreateCollection, bot.CreateCollectionHandler)

	// When user is sending NFTs for collection
	bot.Handle(tele.OnDocument, bot.OnDocumentHandler)

	// When user taping "That's all files"
	bot.Handle(&btnCompleteFiles, bot.OnCompleteFilesHandler)

	// There should fall all text input steps
	bot.Handle(tele.OnText, bot.OnTextHandler)

	// Skip button handler (currently only for skipping symbol input)
	bot.Handle(&btnSkip, bot.SkipHandler)

	// Final step
	bot.Handle(&btnMinted, bot.MintCheckHandler)

	bot.Bot.Start()
}
