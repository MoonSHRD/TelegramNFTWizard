package bot

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"github.com/MoonSHRD/TelegramNFTWizard/pkg/binary"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/wizard"
	"github.com/StarkBotsIndustries/telegraph/v2"
	"golang.org/x/exp/slices"
	tele "gopkg.in/telebot.v3"
)

// User first contact with bot
func (bot *Bot) StartHandler(c tele.Context) error {

	// If user already known to bot
	if has, _ := bot.kv.Has(binary.From(c.Sender().ID)); has {
		// Retrieve user
		var user User
		if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
			log.Println("failed to get user from kv:", err)
			return c.Send(messages["fail"])
		}

		// If user stuck he may do `/start` for troubleshooting
		return bot.remindingResponse(c, user)
	}

	// New user
	user := User{
		CreatedAt: time.Now(),
		State:     Freeroam,
	}

	// Save new user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
		return c.Send(messages["fail"])
	}

	return c.Send(messages["welcome"])
}

func (bot *Bot) CreateCollectionHandler(c tele.Context) error {
	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	if user.State != Freeroam {
		return bot.remindingResponse(c, user)
	}

	// Update state
	user.State = CollectionPreparation

	// Save user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
		return c.Send(messages["fail"])
	}

	// Display keyboard
	return c.Send(messages["awaitingFiles"], completeFiles)
}

func (bot *Bot) OnDocumentHandler(c tele.Context) error {
	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	if user.State != CollectionPreparation {
		return bot.remindingResponse(c, user)
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
	reader, err := bot.File(doc.MediaFile())
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
		if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
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

		return bot.remindingResponse(c, user)
	}

	return nil
}

func (bot *Bot) OnTextHandler(c tele.Context) error {
	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
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
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
		return c.Send(messages["fail"])
	}

	return bot.remindingResponse(c, user)
}

func (bot *Bot) SkipHandler(c tele.Context) error {
	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	if user.State != CollectionPreparationSymbol {
		return bot.remindingResponse(c, user)
	}

	// Skip to mint
	user.State = CollectionMint

	// Save user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
		return c.Send(messages["fail"])
	}

	return bot.remindingResponse(c, user)
}

func (bot *Bot) MintHandler(c tele.Context) error {
	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	if user.State != CollectionMint {
		return bot.remindingResponse(c, user)
	}

	// Checking created items
	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute*time.Duration(5))
	remaining, err := bot.client.FilterCreatedItems(ctx, user.CreatedAt, user.FileIDs...)
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
}

// Repeats current state message to user
func (bot *Bot) remindingResponse(c tele.Context, user User) error {
	switch user.State {

	case Freeroam:
		if bot.client.IsRegistered(c.Sender().ID) {
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
