package websock

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

type Client struct {
	id             string
	channels       map[string]*Channel
	Log            *Log
	conn           net.Conn
	readCh         <-chan InboundMessage
	subscriptionCh chan PublishMessage
	writeCh        chan<- OutboundMessage
	publishCh      chan<- PublishMessage
	handlerFn      func(t string) (MessageHandlerFunc, bool)
	openRequests   *sync.WaitGroup
}

func (c *Client) handle() {
	for {
		select {
		case msg := <-c.readCh:
			if _, ok := c.channels[msg.Channel]; !ok {
				c.channels[msg.Channel] = &Channel{}
			}

			req := Request{
				msg:     msg,
				writeCh: c.writeCh,
				Raw:     msg.Body,
			}
			res := ResponseWriter{
				writeCh:   c.writeCh,
				publishCh: c.publishCh,
				channel:   msg.Channel,
			}
			c.dispatch(req, res)
		case msg := <-c.subscriptionCh:
			for chid, ch := range c.channels {
				for _, sub := range ch.Subscriptions {
					if sub == msg.Topic {
						c.writeCh <- OutboundMessage{
							Channel: chid,
							Type:    msg.Type,
							Body:    msg.Body,
						}
					}
				}
			}
		}
	}
}

func (c *Client) dispatch(r Request, rw ResponseWriter) {
	c.openRequests.Add(1)
	switch r.msg.Type {
	case "open":
		c.openRequests.Done()
	case "close":
		c.openRequests.Done()
	case "subscribe":
		defer c.openRequests.Done()
		var topic string
		if r.Unpack(&topic) {
			topic = strings.ToLower(topic)
			var exists bool
			for _, s := range c.channels[r.msg.Channel].Subscriptions {
				if s == topic {
					exists = true
					break
				}
			}
			if !exists {
				c.channels[r.msg.Channel].Subscriptions = append(c.channels[r.msg.Channel].Subscriptions, topic)
			}
		}
	default:
		if fn, ok := c.handlerFn(r.msg.Type); ok {
			go func() {
				defer c.openRequests.Done()
				fn(r, rw)
			}()
		} else {
			defer c.openRequests.Done()
			c.writeCh <- OutboundMessage{
				Channel: r.msg.Channel,
				Type:    "error",
				Body:    fmt.Sprintf("unknown type %s", r.msg.Type),
			}
		}
	}
}

type Channel struct {
	Subscriptions []string
}
