package bot

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"github.com/MoonSHRD/TelegramNFTWizard/pkg/binary"
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
		return remindingResponse(c, bot.client, user)
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
		return remindingResponse(c, bot.client, user)
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
		return remindingResponse(c, bot.client, user)
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

		return remindingResponse(c, bot.client, user)
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

	return remindingResponse(c, bot.client, user)
}

func (bot *Bot) SkipHandler(c tele.Context) error {
	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	if user.State != CollectionPreparationSymbol {
		return remindingResponse(c, bot.client, user)
	}

	// Skip to mint
	user.State = CollectionMint

	// Save user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
		return c.Send(messages["fail"])
	}

	return remindingResponse(c, bot.client, user)
}

func (bot *Bot) MintHandler(c tele.Context) error {
	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	if user.State != CollectionMint {
		return remindingResponse(c, bot.client, user)
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
