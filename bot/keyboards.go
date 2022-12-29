package bot

import tele "gopkg.in/telebot.v3"

// Keyboards
var (
	menu                = &tele.ReplyMarkup{}
	btnCreateItem       = menu.Data("Create one item", "create_item")
	btnCreateCollection = menu.Data("Create collection", "create_collection")

	completeFiles    = &tele.ReplyMarkup{}
	btnCompleteFiles = completeFiles.Data("That's all files", "complete_files")

	skip    = &tele.ReplyMarkup{}
	btnSkip = skip.Data("Skip", "skip")
)

// Keyboards init
func init() {
	menu.Inline(
		menu.Row(btnCreateItem),
		menu.Row(btnCreateCollection),
	)

	completeFiles.Inline(
		completeFiles.Row(btnCompleteFiles),
	)

	skip.Inline(
		skip.Row(btnSkip),
	)
}
