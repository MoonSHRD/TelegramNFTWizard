package bot

var messages = map[string]string{

	// Welcome step
	"welcome":              "Hey, this bot is allowing you to create NFT",
	"awaitingRegistration": "You are not registred yet, first attach your wallet to your tg account via this bot https://t.me/E_Passport_bot",
	"collectionCreation":   `You in one step before your own NFT, just tap "Create NFT item"`,

	// Uploading files step
	"awaitingFiles":        "Send me a file which will be your NFT.",
	"fileProcessing":       "Processing file...",
	"fileAlreadyProcessed": "You already uploaded file for NFT, if you want to upload different one, cancel and start over.",
	"filesLimitReached":    "Reached limit of files for collection",
	"filesEmpty":           "No files was uploaded",
	"fileSizeLimit":        "Collection cannot hold files bigger than 5MB, try compress or resize it",
	"notAnImage":           "Not an image, supported extensions - 'png', 'jpg', 'webp'",

	// Text inputs step
	"awaitingCollectionName":   "Choose name for collection, example 'Nice kitties'",
	"awaitingCollectionSymbol": "Choose symbol for collection, example '😸', but you can skip it (probably)",

	// Mint step
	"awaitingCollectionMint": "Last step, mint!",
	"checkMint":              "None of files are minted at the moment, wait for confirmations or check your transactions",
	"filesInProcess":         "Some files already processed by contract, one moment and your NFT will be ready to see the world",

	// Outcomes
	"collectionCreated": "Congrats! Your NFT is minted and published",
	"fail":              "Something went wrong, please retry in 5 minutes",
}
