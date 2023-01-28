package bot

import tele "gopkg.in/telebot.v3"

// Keyboards
var (
	menu = &tele.ReplyMarkup{
		OneTimeKeyboard: true,
		ResizeKeyboard:  true,
	}
	btnCreateItemText = "Create NFT item"
	btnCreateItem     = menu.Text(btnCreateItemText)

	btnCancel = menu.Data("Cancel", "cancel_nft_creation")
)

// Keyboards init
func init() {
	menu.Reply(
		menu.Row(btnCreateItem),
	)
}
