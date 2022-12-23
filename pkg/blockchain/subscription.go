package blockchain

import (
	"context"
	"errors"
	"log"

	SingletonNFT "github.com/MoonSHRD/TelegramNFT-Wizard-Contracts/go/SingletonNFT"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/event"
)

type Subscription struct {
	events    event.Subscription
	sink      <-chan *SingletonNFT.SingletonNFTItemCreated
	release   chan struct{}
	err       chan error
	remaining int
}

// Remaining files count
func (sub *Subscription) Remaining() int {
	return sub.remaining
}

// Released channel will emit one unit struct at finish
func (sub *Subscription) Released() <-chan struct{} {
	return sub.release
}

// Error if it happened subscription is closed
func (sub *Subscription) Err() <-chan error {
	return sub.err
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

	sub := &Subscription{
		remaining: len(fileIDs),
		events:    subscription,
		sink:      sink,
		release:   make(chan struct{}),
		err:       make(chan error, 1),
	}

	go client.waitForFiles(sub)

	return sub, nil
}

func (client *Client) waitForFiles(subscription *Subscription) {
	defer subscription.events.Unsubscribe()
	defer close(subscription.err)
	defer close(subscription.release)
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
