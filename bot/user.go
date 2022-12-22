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
	CreatedAt time.Time `json:"created_at"`
	State     State     `json:"state"`
	Name      string    `json:"name"`
	Symbol    string    `json:"symbol"`
	FileIDs   []string  `json:"file_ids"`
}
