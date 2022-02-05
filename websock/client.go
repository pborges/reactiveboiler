package websock

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

type StringArray []string

func (sa StringArray) Contains(str string) bool {
	for _, s := range sa {
		if s == str {
			return true
		}
	}
	return false
}

var protocolTypes StringArray = []string{"open", "close"}

type Client struct {
	server   *Server
	id       string
	conn     *websocket.Conn
	lock     *sync.Mutex
	channels map[string]*Channel
	log      *Log
	Stats    Stats `json:"stats"`
}

func (c Client) Channels() []*Channel {
	res := make([]*Channel, 0, len(c.channels))
	for _, v := range c.channels {
		res = append(res, v)
	}
	return res
}

func (c Client) Id() string {
	return c.id
}

func (c Client) MarshalJSON() ([]byte, error) {
	type alias Client
	a := struct {
		alias
		Id       string    `json:"id"`
		Channels []Channel `json:"channels,omitempty"`
	}{
		alias: alias(c),
		Id:    c.id,
	}
	for _, s := range c.Channels() {
		a.Channels = append(a.Channels, *s)
	}
	return json.Marshal(a)
}

func (c *Client) Handle() {
	c.log.Info.Println("open")
	defer func() {
		c.log.Info.Println("close")
	}()

	for {
		_, buff, err := c.conn.ReadMessage()
		if err != nil {
			c.log.Err.Println("read error", err)
			return
		}

		msg := struct {
			Message
			Body json.RawMessage `json:"body"`
		}{}
		if err := json.Unmarshal(buff, &msg); err != nil {
			c.log.Err.Println("unable to unmarshal", err)
			continue
		}
		if len(msg.Body) > 0 {
			msg.Message.Body = msg.Body
		}

		c.Stats.IncrementRecv()

		if _, ok := c.channels[msg.Channel]; !ok {
			// create new channel
			c.lock.Lock()
			c.channels[msg.Channel] = &Channel{
				id: msg.Channel,
				Stats: Stats{
					LastSent:     time.Now(),
					LastReceived: time.Now(),
				},
				log: NewLog(c.log.Writer, c.log.Flags, c.id, msg.Channel),
			}
			c.lock.Unlock()
		}

		c.channels[msg.Channel].log.Recv(msg.Message)
		c.channels[msg.Channel].Stats.IncrementRecv()

		go c.server.handlers.Handle(c, msg.Message, msg.Body)
	}
}

func (c *Client) Error(ch string, err error) {
	channel, ok := c.channels[ch]
	if ok {
		channel.log.Err.Println(err)
	} else {
		c.log.Err.Println(err)
	}
	c.Write(ch, "error", err.Error())
}

func (c *Client) Write(channel string, t string, body interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.write(channel, t, body)
}

func (c *Client) write(channel string, t string, body interface{}) {
	msg := Message{
		Client:  c.id,
		Channel: channel,
		Type:    t,
		Body:    body,
	}

	ch, ok := c.channels[channel]
	if !ok {
		c.log.Err.Println("unregistered channel", channel)
		return
	}

	ch.log.Send(msg)

	if err := c.conn.WriteJSON(msg); err == nil {
		c.Stats.IncrementSent()
		ch.Stats.IncrementSent()
	} else {
		ch.log.Err.Println("write error", err)
	}
}
