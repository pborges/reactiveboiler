package websock

import (
	"encoding/json"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"io"
	"net"
)

type Client struct {
	id       string
	server   *Server
	conn     net.Conn
	r        *wsutil.Reader
	w        *wsutil.Writer
	enc      *json.Encoder
	dec      *json.Decoder
	log      *Log
	channels map[string]*Channel
	lock     *Lock
	iolock   *Lock
}

func (c *Client) error(channel string, err error) {
	c.write(channel, "error", err.Error())
}
func (c *Client) write(channel string, t string, body interface{}) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	c.iolock.Lock()
	defer c.iolock.Unlock()

	ch, ok := c.channels[channel]
	if ok {
		ch.IncrementSent()
		resp := Message{
			Channel: channel,
			Type:    t,
			Body:    body,
		}
		ch.log.Send(resp)

		c.w.Reset(c.conn, ws.StateServerSide, ws.OpText)
		if err := c.enc.Encode(resp); err == nil {
			if err := c.w.Flush(); err != nil {
				c.log.Err.Println("unable to flush", err)
			}
		} else {
			c.log.Err.Println("unable to encode", body, err)
		}
	} else {
		c.log.Info.Println("unknown channel", channel)
	}
}

func (c *Client) handleAndDispatch(req Message, buf []byte) error {
	c.server.handlersLock.RLock()
	defer c.server.handlersLock.RUnlock()

	fn, ok := c.server.protoHandlers[req.Type]
	if !ok {
		fn, ok = c.server.handlers[req.Type]
	}

	if ok {
		if fn != nil {
			go func() {
				r := Request{
					client:  c,
					channel: req.Channel,
					Raw:     buf,
				}
				rw := &ResponseWriter{
					client:  c,
					channel: req.Channel,
				}
				fn(r, rw)
			}()
		}
	} else {
		c.log.Err.Println("unknown type", req.Type)
	}
	return nil
}

func (c *Client) newChannel(id string) *Channel {
	c.lock.Lock()
	defer c.lock.Unlock()
	ch := &Channel{
		Id:    id,
		Stats: &Stats{},
		log:   NewLog(c.server.log.Writer, c.server.log.Flags, c.id, id),
	}
	c.channels[id] = ch
	return ch
}

func (c *Client) nextFrame() error {
	hdr, err := c.r.NextFrame()
	if err != nil {
		return err
	}

	if hdr.OpCode == ws.OpClose {
		return io.EOF
	}

	req := struct {
		Message
		Body json.RawMessage `json:"body"`
	}{}
	if err := c.dec.Decode(&req); err != nil {
		return err
	}

	c.lock.RLock()
	channel, ok := c.channels[req.Channel]
	c.lock.RUnlock()

	if !ok {
		channel = c.newChannel(req.Channel)
	}
	channel.IncrementRecv()
	if len(req.Body) > 0 {
		req.Message.Body = req.Body
	}
	channel.log.Recv(req.Message)

	return c.handleAndDispatch(req.Message, req.Body)
}

func (c *Client) subscribe(channel string, topic string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.channels[channel].Subscriptions = append(c.channels[channel].Subscriptions, topic)
}

func (c *Client) MarshalJSON() ([]byte, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	data := map[string]interface{}{}
	channels := make([]*Channel, 0, len(c.channels))
	for _, ch := range c.channels {
		channels = append(channels, ch)
	}
	data["channels"] = channels
	return json.Marshal(data)
}

func (c *Client) closeChannel(channel string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.channels, channel)
}
