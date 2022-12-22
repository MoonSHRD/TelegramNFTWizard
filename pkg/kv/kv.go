package kv

import (
	"encoding/json"

	"github.com/akrylysov/pogreb"
)

type KV struct {
	*pogreb.DB
}

func New(databasePath string) (*KV, error) {
	db, err := pogreb.Open(databasePath, nil)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	return &KV{
		DB: db,
	}, nil
}

func (kv *KV) GetJson(key []byte, out any) error {
	data, err := kv.Get(key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}

func (kv *KV) PutJson(key []byte, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return kv.Put(key, data)
}
