package eventstream

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/coder/websocket"
)

// This server implementation is a refactor of the chat server example
// from
// https://github.com/coder/websocket/blob/master/internal/examples/chat/chat.go

// EventStream binds all the components of the event streaming server.
type EventStream struct {
	l hclog.Logger

	maxUndelivered int

	subscribersMutex sync.Mutex
	subscribers      map[*subscriber]struct{}
}

// subscriber represents a subscriber.
// Messages are sent on the msgs channel and if the client
// cannot keep up with the messages, closeSlow is called.
type subscriber struct {
	msgs      chan []byte
	closeSlow func()
}

// Handler implements the http.Handler interface so that the
// eventstream can be plugged into a webserver.
func (es *EventStream) Handler(w http.ResponseWriter, r *http.Request) {
	err := es.subscribe(w, r)
	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		es.l.Warn("Error handling subscription request", "error", err)
		return
	}
}

// subscribe subscribes the given WebSocket to all broadcast messages.
// It creates a subscriber with a buffered msgs chan to give some room to slower
// connections and then registers the subscriber. It then listens for all messages
// and writes them to the WebSocket. If the context is cancelled or
// an error occurs, it returns and deletes the subscription.
//
// It uses CloseRead to keep reading from the connection to process control
// messages and cancel the context if the connection drops.
func (es *EventStream) subscribe(w http.ResponseWriter, r *http.Request) error {
	var mu sync.Mutex
	var c *websocket.Conn
	var closed bool
	s := &subscriber{
		msgs: make(chan []byte, es.maxUndelivered),
		closeSlow: func() {
			mu.Lock()
			defer mu.Unlock()
			closed = true
			if c != nil {
				c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
			}
		},
	}
	es.addSubscriber(s)
	defer es.deleteSubscriber(s)

	c2, err := websocket.Accept(w, r, nil)
	if err != nil {
		return err
	}
	mu.Lock()
	if closed {
		mu.Unlock()
		return net.ErrClosed
	}
	c = c2
	mu.Unlock()
	defer c.CloseNow()

	ctx := c.CloseRead(context.Background())

	for {
		select {
		case msg := <-s.msgs:
			err := writeTimeout(ctx, time.Second*5, c, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// publish publishes the msg to all subscribers.
// It never blocks and so messages to slow subscribers
// are dropped.
func (es *EventStream) publish(msg []byte) {
	es.subscribersMutex.Lock()
	defer es.subscribersMutex.Unlock()

	for s := range es.subscribers {
		select {
		case s.msgs <- msg:
		default:
			go s.closeSlow()
		}
	}
}

// addSubscriber registers a subscriber.
func (es *EventStream) addSubscriber(s *subscriber) {
	es.subscribersMutex.Lock()
	es.subscribers[s] = struct{}{}
	es.subscribersMutex.Unlock()
}

// deleteSubscriber deletes the given subscriber.
func (es *EventStream) deleteSubscriber(s *subscriber) {
	es.subscribersMutex.Lock()
	delete(es.subscribers, s)
	es.subscribersMutex.Unlock()
}

func writeTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return c.Write(ctx, websocket.MessageText, msg)
}
