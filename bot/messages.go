package bot

var messages = map[string]string{
	"welcome":                  "Hey, this bot is allowing you to create NFT",
	"awaitingRegistration":     "You are not registred yet, first attach your wallet to your tg account via this bot https://t.me/E_Passport_bot",
	"collectionCreation":       `You in one step before your own NFT collection, just tap "Create collection" on keyboard`,
	"awaitingFiles":            "Send me a _files_ which u want to transform into NFT. \nYou can upload up to 10 files per collection",
	"filesLimitReached":        "Reached limit of files for collection",
	"notAnImage":               "Not an image, supported extensions - 'png', 'jpg', 'webp'",
	"awaitingCollectionName":   "Choose name for collection, example 'Nice kitties'",
	"awaitingCollectionSymbol": "Choose symbol for collection, example '😸', but you can skip it (probably)",
	"awaitingCollectionMint":   "Last step, mint!",
	"checkMint":                "None of files are minted at the moment, wait for confirmations or check your transactions",
	"filesInProcess":           "Some files already processed by contract, one moment and your collection will be ready to see the world",
	"collectionCreated":        "Congrats! Your collection is minted and published",
	"fail":                     "Something went wrong, please retry in 5 minutes",
}
