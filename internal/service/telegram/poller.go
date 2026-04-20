package telegram

import (
	"fmt"
	"log"
	"time"

	"github.com/AzozzALFiras/Nullhand/internal/model/message"
)

const (
	longPollTimeout = 30  // seconds Telegram waits for a message before returning empty
	retryDelay      = 5  // seconds to wait after a network error before retrying
)

// Handler is called for each incoming update.
type Handler func(update message.Update)

// Poller continuously long-polls Telegram and calls handler for each update.
type Poller struct {
	client  *Client
	handler Handler
	stop    chan struct{}
}

// NewPoller creates a Poller that delivers updates to handler.
func NewPoller(client *Client, handler Handler) *Poller {
	return &Poller{
		client:  client,
		handler: handler,
		stop:    make(chan struct{}),
	}
}

// Start begins polling in the current goroutine. Call Stop() to halt it.
func (p *Poller) Start() {
	offset := 0
	for {
		select {
		case <-p.stop:
			return
		default:
		}

		updates, err := p.client.GetUpdates(offset, longPollTimeout)
		if err != nil {
			log.Printf("poller: %v — retrying in %ds", err, retryDelay)
			select {
			case <-p.stop:
				return
			case <-time.After(time.Duration(retryDelay) * time.Second):
			}
			continue
		}

		for _, u := range updates {
			offset = u.UpdateID + 1
			p.handler(u)
		}
	}
}

// Stop signals the poller to exit after the current long-poll completes.
func (p *Poller) Stop() {
	select {
	case <-p.stop:
		// already stopped
	default:
		close(p.stop)
	}
}

// Validate checks that the bot token is valid by calling getMe.
func Validate(token string) error {
	c := NewClient(token)
	updates, err := c.GetUpdates(0, 1)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}
	_ = updates
	return nil
}
