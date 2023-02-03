package bot

import (
	"log"
	"sync"
	"time"

	"github.com/MoonSHRD/TelegramNFTWizard/config"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/binary"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/blockchain"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/kv"
	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	*tele.Bot
	kv        *kv.KV
	client    *blockchain.Client
	createdAt int64

	ls            *sync.Mutex
	subscriptions map[int64]*blockchain.Subscription

	// File processing mutex, solution for many images from users, but will lock for everyone else
	lp         *sync.Mutex
	processing map[int64]struct{}
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
		ls:            &sync.Mutex{},
		subscriptions: make(map[int64]*blockchain.Subscription),
		lp:            &sync.Mutex{},
		processing:    make(map[int64]struct{}),
	}, nil
}

func (bot *Bot) Start() {
	// All handles are asynchronous, keep it in mind

	// User first contact with bot
	bot.Handle("/start", bot.StartHandler)

	// There should fall all text steps
	bot.Handle(tele.OnText, bot.OnTextHandler)

	// When user is sending NFTs for collection
	bot.Handle(tele.OnDocument, bot.OnDocumentHandler)

	// When user is sending NFTs for collection
	bot.Handle(tele.OnPhoto, bot.OnPhotoHandler)

	bot.Handle(&btnCancel, bot.OnCancel)

	bot.Bot.Start()
}

func (bot *Bot) ResetUser(r *tele.User) error {
	log.Println("reseting...")

	bot.ls.Lock()
	defer bot.ls.Unlock()

	sub, ok := bot.subscriptions[r.ID]
	if ok {
		log.Printf("deleting subscription to %v, userid: %d", sub.Tokens(), r.ID)
		delete(bot.subscriptions, r.ID)
	}

	if err := bot.kv.PutJson(binary.From(r.ID), UserDefaults()); err != nil {
		log.Println("failed to put user in kv:", err)
		_, err := bot.Send(r, messages["fail"])
		if err != nil {
			log.Println("failed to send fail message:", err)
		}
	}

	return nil
}
