package bot

import (
	"context"
	"log"
	"net/url"
	"path/filepath"

	"github.com/MoonSHRD/TelegramNFTWizard/pkg/binary"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/wizard"
	"github.com/StarkBotsIndustries/telegraph/v2"
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
	user := UserDefaults()

	// Save new user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
		return c.Send(messages["fail"])
	}

	if err := c.Send(messages["welcome"]); err != nil {
		log.Println("failed to respond to user:", err)
	}

	return bot.remindingResponse(c, user)
}

func (bot *Bot) CreateItemHandler(c tele.Context) error {
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
	user.IsSingleFile = true

	// Save user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
		return c.Send(messages["fail"])
	}

	// Display keyboard
	return c.Send(messages["awaitingFiles"], completeFiles)
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

	if doc.FileSize >= 5e+6 {
		return c.Send(messages["fileSizeLimit"])
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

	u, err := url.Parse(telegraphLink)
	if err != nil {
		log.Println("failed to parse telegraph link:", err)
		return c.Send(messages["fail"])
	}

	user.FileIDs = append(user.FileIDs, filepath.Base(u.Path))

	if err := c.Send("Saved file at "+telegraphLink, completeFiles); err != nil {
		log.Println("failed to respond to user:", err)
	}

	// If limit reached force user to next step
	if len(user.FileIDs) >= 10 {
		user.State = CollectionPreparationName

		// Save user
		if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
			log.Println("failed to put user in kv:", err)
		}

		return bot.remindingResponse(c, user)
	}

	if user.IsSingleFile {
		user.State = CollectionMint

		// Save user
		if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
			log.Println("failed to put user in kv:", err)
		}

		return bot.remindingResponse(c, user)
	}

	// Save user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
	}

	return nil
}

func (bot *Bot) OnCompleteFilesHandler(c tele.Context) error {
	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	if user.State != CollectionPreparation {
		return bot.remindingResponse(c, user)
	}

	if len(user.FileIDs) == 0 {
		return c.Send(messages["filesEmpty"])
	}

	// Updating step
	user.State = CollectionPreparationName

	// Save user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
	}

	return bot.remindingResponse(c, user)
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
		// In default case we do nothin' and responding with reminder
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
	user.SubscriptionInstance = bot.createdAt

	// Save user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
		return c.Send(messages["fail"])
	}

	return bot.remindingResponse(c, user)
}

func (bot *Bot) MintCheckHandler(c tele.Context) error {
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
	ctx := context.Background()
	remaining, err := bot.client.CheckItemsCreated(ctx, user.FileIDs, user.StartedAt)
	if err != nil {
		log.Println("failed checking minted files:", err)
	}

	log.Printf(
		"Checked created items for %s (%d), remaining %+v out of %+v\n",
		c.Sender().Username,
		c.Sender().ID,
		remaining,
		len(user.FileIDs),
	)

	if remaining == len(user.FileIDs) {
		return c.Send(messages["checkMint"])
	}

	if remaining != 0 {
		return c.Send(messages["filesInProcess"])
	}

	// Reset user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), UserDefaults()); err != nil {
		log.Println("failed to put user in kv:", err)
		return c.Send(messages["fail"])
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
		if user.IsSingleFile {
			return c.Send(messages["awaitingFiles"])
		} else {
			return c.Send(messages["awaitingFiles"], completeFiles)
		}

	case CollectionPreparationName:
		return c.Send(messages["awaitingCollectionName"])

	case CollectionPreparationSymbol:
		return c.Send(messages["awaitingCollectionSymbol"], skip)

	case CollectionMint:
		var url string
		var err error

		if user.IsSingleFile {
			if url, err = wizard.CreateSingleItemLink(user.FileIDs[0]); err != nil {
				log.Println("failed to single item link:", err)
				return c.Send(messages["fail"])
			}
		} else {
			if url, err = wizard.CreateCollectionLink(wizard.CollectionOptions{
				Name:    user.Name,
				FileIDs: user.FileIDs,
			}); err != nil {
				log.Println("failed to create collection link:", err)
				return c.Send(messages["fail"])
			}
		}

		mint := &tele.ReplyMarkup{}
		mint.Inline(
			mint.Row(mint.URL("Mint", url)),
			mint.Row(btnMinted),
		)
		return c.Send(messages["awaitingCollectionMint"], mint)

	default:
		// ? - figure out proper response
		return nil
	}
}
