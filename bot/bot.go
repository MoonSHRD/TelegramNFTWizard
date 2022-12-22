package bot

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"github.com/MoonSHRD/TelegramNFTWizard/config"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/binary"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/blockchain"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/kv"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/wizard"
	"github.com/StarkBotsIndustries/telegraph/v2"
	"golang.org/x/exp/slices"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

// Keyboards
var (
	menu      = &tele.ReplyMarkup{}
	btnCreate = menu.Data("Create collection", "create_collection")

	completeFiles    = &tele.ReplyMarkup{}
	btnCompleteFiles = completeFiles.Data("That's all files", "complete_files")

	skip    = &tele.ReplyMarkup{}
	btnSkip = skip.Data("Skip", "skip")

	minted    = &tele.ReplyMarkup{}
	btnMinted = skip.Data("Check mint status", "check_status")
)

// Keyboards init
func init() {
	menu.Inline(
		menu.Row(btnCreate),
	)

	completeFiles.Inline(
		completeFiles.Row(btnCompleteFiles),
	)

	skip.Inline(
		skip.Row(btnSkip),
	)

	minted.Inline(
		minted.Row(btnMinted),
	)
}

func Run(config config.Config) error {
	client, err := blockchain.NewClient(config.Config)
	if err != nil {
		return err
	}

	kv, err := kv.New(config.DatabasePath)
	if err != nil {
		return err
	}
	defer kv.Close()

	pref := tele.Settings{
		ParseMode: tele.ModeMarkdownV2,
		Token:     config.Token,
		Poller:    &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return err
	}

	b.Use(middleware.Logger())

	// User first contact with bot
	b.Handle("/start", func(c tele.Context) error {

		// If user already known to bot
		if has, _ := kv.Has(binary.From(c.Sender().ID)); has {
			// Retrieve user
			var user User
			if err := kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
				log.Println("failed to get user from kv:", err)
				return c.Send(messages["fail"])
			}

			// If user stuck he may do `/start` for troubleshooting
			return remindingResponse(c, client, user)
		}

		// New user
		user := User{
			CreatedAt: time.Now(),
			State:     Freeroam,
		}

		// Save new user
		if err := kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
			log.Println("failed to put user in kv:", err)
			return c.Send(messages["fail"])
		}

		return c.Send(messages["welcome"])
	})

	// When user taping "Create collection"
	b.Handle(&btnCreate, func(c tele.Context) error {
		// Retrieve user
		var user User
		if err := kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
			log.Println("failed to get user from kv:", err)
			return c.Send(messages["fail"])
		}

		if user.State != Freeroam {
			return remindingResponse(c, client, user)
		}

		// Update state
		user.State = CollectionPreparation

		// Save user
		if err := kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
			log.Println("failed to put user in kv:", err)
			return c.Send(messages["fail"])
		}

		// Display keyboard
		return c.Send(messages["awaitingFiles"], completeFiles)
	})

	// When user is sending NFTs for collection
	b.Handle(tele.OnDocument, func(c tele.Context) error {
		// Retrieve user
		var user User
		if err := kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
			log.Println("failed to get user from kv:", err)
			return c.Send(messages["fail"])
		}

		if user.State != CollectionPreparation {
			return remindingResponse(c, client, user)
		}

		// If limit at the end fails
		if len(user.FileIDs) >= 10 {
			log.Println("file limit failed, promting manual complete button")
			return c.Send(messages["filesLimitReached"], completeFiles)
		}

		doc := c.Message().Document

		// Document is image
		if !slices.Contains([]string{"png", "jpg", "webp"}, filepath.Ext(doc.FileName)) {
			return c.Send(messages["notAnImage"])
		}

		// Fetch image
		reader, err := b.File(doc.MediaFile())
		if err != nil {
			log.Println("failed to get file from user:", err)
			return c.Send(messages["fail"])
		}
		defer reader.Close()

		// Upload to telegraph
		telegraphLink, err := telegraph.Upload(reader, "photo")
		if err != nil {
			log.Println("failed to upload file to telegraph:", err)
			return c.Send(messages["fail"])
		}

		user.FileIDs = append(user.FileIDs, doc.FileID)

		if err := c.Send("Saved file at "+telegraphLink, completeFiles); err != nil {
			log.Println("failed to respond to user:", err)
		}

		defer func() {
			// Save user
			if err := kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
				log.Println("failed to put user in kv:", err)
			}
		}()

		// If limit reached force user to next step
		if len(user.FileIDs) >= 10 {
			if err := c.Send(messages["filesLimitReached"]); err != nil {
				log.Println("failed to respond to user:", err)
				return c.Send(messages["fail"])
			}

			user.State = CollectionPreparationName

			return remindingResponse(c, client, user)
		}

		return nil
	})

	// There should fall all text input steps
	b.Handle(tele.OnText, func(c tele.Context) error {
		// Retrieve user
		var user User
		if err := kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
			log.Println("failed to get user from kv:", err)
			return c.Send(messages["fail"])
		}

		switch user.State {
		case CollectionPreparationName:
			// TODO: necessary checks on name validity
			user.Name = c.Text()
			user.State = CollectionPreparationSymbol
		case CollectionPreparationSymbol:
			// TODO: necessary checks on symbol validity
			user.Symbol = c.Text()
			user.State = CollectionMint
		default:
		}

		// Save user
		if err := kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
			log.Println("failed to put user in kv:", err)
			return c.Send(messages["fail"])
		}

		return remindingResponse(c, client, user)
	})

	// Skip button handler (currently only for skipping symbol input)
	b.Handle(&btnSkip, func(c tele.Context) error {
		// Retrieve user
		var user User
		if err := kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
			log.Println("failed to get user from kv:", err)
			return c.Send(messages["fail"])
		}

		if user.State != CollectionPreparationSymbol {
			return remindingResponse(c, client, user)
		}

		// Skip to mint
		user.State = CollectionMint

		// Save user
		if err := kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
			log.Println("failed to put user in kv:", err)
			return c.Send(messages["fail"])
		}

		return remindingResponse(c, client, user)
	})

	b.Handle(&btnMinted, func(c tele.Context) error {
		// Retrieve user
		var user User
		if err := kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
			log.Println("failed to get user from kv:", err)
			return c.Send(messages["fail"])
		}

		if user.State != CollectionMint {
			return remindingResponse(c, client, user)
		}

		// Checking created items
		ctx, cancel := context.WithTimeout(context.TODO(), time.Minute*time.Duration(5))
		remaining, err := client.FilterCreatedItems(ctx, user.CreatedAt, user.FileIDs...)
		if err != nil {
			log.Println("failed checking minted files:", err)
		}
		cancel()

		log.Printf(
			"Checked created items for %s (%d), remaining %+v out of %+v\n",
			c.Sender().Username,
			c.Sender().ID,
			remaining,
			user.FileIDs,
		)

		if len(remaining) == len(user.FileIDs) {
			return c.Send(messages["checkMint"])
		}

		if len(remaining) != 0 {
			return c.Send(messages["filesInProcess"])
		}

		return c.Send(messages["collectionCreated"])
	})

	b.Start()

	return nil
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
