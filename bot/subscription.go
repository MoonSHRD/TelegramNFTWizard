package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/MoonSHRD/TelegramNFTWizard/pkg/binary"
	"github.com/MoonSHRD/TelegramNFTWizard/pkg/blockchain"
	tele "gopkg.in/telebot.v3"
)

func (bot *Bot) subscribe(r *tele.User, user User) error {
	bot.ls.Lock()
	_, exists := bot.subscriptions[r.ID]
	bot.ls.Unlock()

	if exists {
		return nil
	}

	ctx := context.Background()
	start := uint64(user.StartedAt.Unix())
	var sub *blockchain.Subscription
	var err error
	if user.IsSingleFile {
		sub, err = bot.client.SubscribeToItems(ctx, user.FileIDs, &start)
		if err != nil {
			return err
		}
	} else {
		passport, err := bot.client.Passport.GetPassportByTgId(r.ID)
		if err != nil {
			log.Println("failed to get user passport:", err)
			return err
		}

		sub, err = bot.client.SubscribeToCreator(ctx, passport.UserAddress, &start)
		if err != nil {
			return err
		}
	}
	user.SubscriptionInstance = bot.createdAt

	bot.ls.Lock()
	bot.subscriptions[r.ID] = sub
	bot.ls.Unlock()

	// Save user
	if err := bot.kv.PutJson(binary.From(r.ID), user); err != nil {
		log.Println("failed to put user in kv:", err)
		_, err := bot.Send(r, messages["fail"])
		if err != nil {
			return err
		}
	}

	go func(r *tele.User) {
		// Waiting for event
		_, ok := <-sub.Released()
		if !ok {
			return
		}

		var res []string
		for _, t := range sub.Tokens() {
			res = append(res, fmt.Sprintf("tokenID: %s\naddress: %s", t.TokenID, t.Address))
		}

		// Success
		_, err := bot.Send(r, messages["collectionCreated"]+strings.Join(res, "\n\n"))
		if err != nil {
			log.Println("failed to send collection created message:", err)
		}

		// Clear subscription
		bot.ls.Lock()
		delete(bot.subscriptions, r.ID)
		bot.ls.Unlock()

		// Reset user
		if err := bot.kv.PutJson(binary.From(r.ID), UserDefaults()); err != nil {
			log.Println("failed to put user in kv:", err)
			_, err := bot.Send(r, messages["fail"])
			if err != nil {
				log.Println("failed to send fail message:", err)
			}
		}
	}(r)

	return nil
}
