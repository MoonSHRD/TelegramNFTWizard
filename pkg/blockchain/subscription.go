package blockchain

import (
	"context"
	"errors"
	"log"

	"github.com/MoonSHRD/TelegramNFT-Wizard-Contracts/go/FactoryNFT"
	SingletonNFT "github.com/MoonSHRD/TelegramNFT-Wizard-Contracts/go/SingletonNFT"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
)

type Subscription struct {
	events    event.Subscription
	sink      <-chan struct{}
	release   chan struct{}
	err       chan error
	remaining int

	tokens []string
}

// Remaining files count
func (sub *Subscription) Remaining() int {
	return sub.remaining
}

// Released channel will emit one unit struct at finish
func (sub *Subscription) Released() <-chan struct{} {
	return sub.release
}

func (sub *Subscription) Tokens() []string {
	return sub.tokens
}

// Error if it happened subscription is closed
func (sub *Subscription) Err() <-chan error {
	return sub.err
}

func (sub *Subscription) Close() {
	close(sub.release)
	close(sub.err)
	sub.events.Unsubscribe()
}

// SubscribeToItems creates subscription to CreateItem event, on Released() returns signal channel which returns one signal on finish.
// In case of error subscription fails and error goes into Err().
// If `start` is nil Watch is started from last block, it's better to use user create time.
func (client *Client) SubscribeToItems(ctx context.Context, fileIDs []string, start *uint64) (*Subscription, error) {
	if len(fileIDs) == 0 {
		return nil, errors.New("zero file id's was provided for watching")
	}

	var sink = make(chan *SingletonNFT.SingletonNFTItemCreated)
	subscription, err := client.Signleton.Contract.WatchItemCreated(&bind.WatchOpts{
		Start:   start, // nil = last block
		Context: ctx,   // nil = no timeout
	}, sink, fileIDs)
	if err != nil {
		return nil, err
	}

	var evSink = make(chan struct{})

	sub := &Subscription{
		remaining: len(fileIDs),
		events:    subscription,
		sink:      evSink,
		release:   make(chan struct{}),
		err:       make(chan error, 1),
		tokens:    []string{},
	}

	// Simple pass through signal channel
	go func() {
		for ev := range sink {
			sub.tokens = append(sub.tokens, ev.TokenId.String())
			evSink <- struct{}{}
		}
	}()

	go client.waitForEvents(sub)

	return sub, nil
}

// SubscribeToCreator creates subscription to CollectionCreated event, on Released() returns signal channel which returns one signal on finish.
// In case of error subscription fails and error goes into Err().
// If `start` is nil Watch is started from last block, it's better to use user create time.
func (client *Client) SubscribeToCreator(ctx context.Context, creator common.Address, start *uint64) (*Subscription, error) {
	var sink = make(chan *FactoryNFT.FactoryNFTCollectionCreated)
	subscription, err := client.Factory.Contract.WatchCollectionCreated(&bind.WatchOpts{
		Start:   start, // nil = last block
		Context: ctx,   // nil = no timeout
	}, sink, []common.Address{creator})
	if err != nil {
		return nil, err
	}

	var evSink = make(chan struct{})

	sub := &Subscription{
		remaining: 1,
		events:    subscription,
		sink:      evSink,
		release:   make(chan struct{}),
		err:       make(chan error, 1),
	}

	// Simple pass through signal channel
	go func() {
		for ev := range sink {
			sub.tokens = append(sub.tokens, ev.Collection.String())
			evSink <- struct{}{}
		}
	}()

	go client.waitForEvents(sub)

	return sub, nil
}

func (client *Client) waitForEvents(subscription *Subscription) {
	defer subscription.Close()
	for {
		select {
		case <-subscription.sink:
			{
				// Tracking files count
				subscription.remaining -= 1

				// Release
				if subscription.remaining <= 0 {
					log.Println("subscription awaited all files, releasing")
					subscription.release <- struct{}{}
					return
				}
			}
		case err := <-subscription.events.Err():
			{
				log.Println("subscription error:", err)
				subscription.err <- err
				return
			}
		}
	}
}
