package bot

import tele "gopkg.in/telebot.v3"

// Keyboards
var (
	menu          = &tele.ReplyMarkup{}
	btnCreateItem = menu.Data("Create NFT item", "create_item")

	skip    = &tele.ReplyMarkup{}
	btnSkip = skip.Data("Skip", "skip")

	btnCancel = menu.Data("Cancel NFT creation", "cancel_nft_creation")
)

// Keyboards init
func init() {
	menu.Inline(
		menu.Row(btnCreateItem),
		// menu.Row(btnCreateCollection),
	)

	skip.Inline(
		skip.Row(btnSkip),
	)
}
