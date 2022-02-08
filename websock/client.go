package websock

import (
	"fmt"
	"net"
	"strings"
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
	switch r.msg.Type {
	case "open":
	case "close":
	case "subscribe":
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
			go fn(r, rw)
		} else {
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
