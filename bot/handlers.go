package bot

import (
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
		return bot.remindingResponse(c)
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

	return bot.remindingResponse(c)
}

func (bot *Bot) OnDocumentHandler(c tele.Context) error {
	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	if user.State != CollectionPreparation {
		return bot.remindingResponse(c)
	}

	return bot.handleFile(c, c.Message().Document.MediaFile())
}

func (bot *Bot) OnPhotoHandler(c tele.Context) error {
	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	if user.State != CollectionPreparation {
		return bot.remindingResponse(c)
	}

	return bot.handleFile(c, c.Message().Photo.MediaFile())
}

func (bot *Bot) handleFile(c tele.Context, file *tele.File) error {
	bot.lp.Lock()
	_, yes := bot.processing[c.Sender().ID]
	if yes {
		return c.Send(messages["fileAlreadyProcessed"])
	}
	bot.processing[c.Sender().ID] = struct{}{}
	bot.lp.Unlock()

	defer func(id int64) {
		bot.lp.Lock()
		delete(bot.processing, id)
		bot.lp.Unlock()
	}(c.Sender().ID)

	if err := c.Send(messages["fileProcessing"]); err != nil {
		return c.Send(messages["fail"])
	}

	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	// File size limit 5MB
	if file.FileSize >= 5e+6 {
		return c.Send(messages["fileSizeLimit"])
	}

	// Fetch image
	reader, err := bot.File(file)
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

	log.Println("Saved file at " + telegraphLink)

	if user.IsSingleFile {
		user.State = CollectionMint

		// Save user
		if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
			log.Println("failed to put user in kv:", err)
		}

		return bot.remindingResponse(c)
	}

	// Save user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
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

	case Freeroam:
		if c.Text() == btnCreateItemText {
			return bot.createItemHandler(c)
		}

	default:
		// In default case we do nothin' and responding with reminder
	}

	// Save user
	if err := bot.kv.PutJson(binary.From(c.Sender().ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
		return c.Send(messages["fail"])
	}

	return bot.remindingResponse(c)
}

func (bot *Bot) createItemHandler(c tele.Context) error {
	// Retrieve user
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	if user.State != Freeroam {
		return bot.remindingResponse(c)
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
	return c.Send(messages["awaitingFiles"])
}

func (bot *Bot) OnCancel(c tele.Context) error {
	// Reset user
	if err := bot.ResetUser(c.Sender()); err != nil {
		log.Println("failed to reset user:", err)
		return c.Send(messages["fail"])
	}

	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	if err := bot.remindingResponse(c); err != nil {
		return err
	}

	return c.Respond()
}

// Repeats current state message to user
func (bot *Bot) remindingResponse(c tele.Context) error {
	var user User
	if err := bot.kv.GetJson(binary.From(c.Sender().ID), &user); err != nil {
		log.Println("failed to get user from kv:", err)
		return c.Send(messages["fail"])
	}

	switch user.State {

	case Freeroam:
		if bot.client.IsRegistered(c.Sender().ID) {
			return c.Send(messages["collectionCreation"], menu)
		} else {
			return c.Send(messages["awaitingRegistration"])
		}

	case CollectionPreparation:
		return c.Send(messages["awaitingFiles"])

	case CollectionMint:
		err := bot.subscribe(c.Sender(), user)
		if err != nil {
			log.Println("failed to sub:", err)
			return c.Send(messages["fail"])
		}

		var url string
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
			mint.Row(btnCancel),
		)
		return c.Send(messages["awaitingCollectionMint"], mint)

	default:
		// ? - figure out proper response
		return nil
	}
}
