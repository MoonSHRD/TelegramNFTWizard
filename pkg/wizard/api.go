package wizard

import (
	"net/url"
	"strconv"

	v "github.com/MoonSHRD/TelegramNFTWizard/pkg/validate"
)

var BaseURL = "https://telegram-nft-wizard.vercel.app/"
var SingleItemPath = "/createnft"
var CollectionPath = "/createcollection"

func SingleItemLink(fileID string) (string, error) {
	if err := v.Validate.Var(fileID, "required"); err != nil {
		return "", err
	}

	u, err := url.ParseRequestURI(BaseURL)
	if err != nil {
		return "", err
	}

	u = u.JoinPath(SingleItemPath)

	// Build up URL Query
	var query url.Values
	query.Set("file_id", fileID)
	u.RawQuery = query.Encode()

	// Output URL
	return u.String(), nil
}

type CollectionOptions struct {
	Name    string `validate:"required"`
	Symbol  *string
	FileIDs []string `validate:"required"`
}

func CreateCollectionLink(options CollectionOptions) (string, error) {
	if err := v.Validate.Struct(options); err != nil {
		return "", err
	}

	u, err := url.ParseRequestURI(BaseURL)
	if err != nil {
		return "", err
	}

	u = u.JoinPath(CollectionPath)

	// Build up URL Query
	var query url.Values

	query.Set("name", options.Name)

	if options.Symbol != nil {
		query.Set("symbol", *options.Symbol)
	}

	for index, file_id := range options.FileIDs {
		num := strconv.Itoa(index)
		if index == 0 {
			num = ""
		}

		query.Set("file_id"+num, file_id)
	}

	query.Set("item_count", strconv.Itoa(len(options.FileIDs)))

	u.RawQuery = query.Encode()

	// Output URL
	return u.String(), nil
}
