package bot

import "time"

// State type for aliasing idents for specific steps of interaction with bot
type State string

// Known states
const (
	// User is in menu
	Freeroam State = "freeroam"
	// User in files uploading step
	CollectionPreparation State = "collectionPreparation"
	// User in collection naming step
	CollectionPreparationName State = "collectionPreparationName"
	// User in collection symbol choosing step
	CollectionPreparationSymbol State = "collectionPreparationSymbol"
	// User in minting step
	CollectionMint State = "collectionMint"
)

// User is state on edge for bot to keep track of interaction pipeline
type User struct {
	// StartedAt time used to track user pipeline start
	// Resets after minting
	StartedAt time.Time `json:"started_at"`
	State     State     `json:"state"`
	// NFT item name
	Name string `json:"name"`
	// NFT symbol (currently unused i guess)
	Symbol string `json:"symbol"`
	// Is it single file NFT
	IsSingleFile bool `json:"is_single_file"`
	// Telegraph File ID's, goes to wizard as is
	FileIDs []string `json:"file_ids"`
	// Stores time when bot instance was created, it needed to restore subscription if bot fails
	SubscriptionInstance int64 `json:"subscription_instance"`
}

// UserDefaults sets StartedAt and Freeroam state
func UserDefaults() User {
	return User{
		StartedAt: time.Now(),
		State:     Freeroam,
	}
}
