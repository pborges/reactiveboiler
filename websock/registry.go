package websock

import (
	"errors"
	"fmt"
	"sync"
)

type ClientRegistry struct {
	db   map[string]*Client
	lock sync.Mutex
}

func (c *ClientRegistry) List() []*Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	res := make([]*Client, 0, len(c.db))
	for _, client := range c.db {
		res = append(res, client)
	}
	return res
}

func (c *ClientRegistry) Delete(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.db, key)
}

func (c *ClientRegistry) Get(key string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.db[key]
}

func (c *ClientRegistry) Set(client *Client) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.db[client.id] = client
}

type HandlerRegistry struct {
	db   map[string]TypeHandlerFunc
	lock sync.Mutex
}

func (reg *HandlerRegistry) Delete(key string) {
	reg.lock.Lock()
	defer reg.lock.Unlock()
	delete(reg.db, key)
}

func (reg *HandlerRegistry) Handle(c *Client, msg Message, body []byte) {
	reg.lock.Lock()
	defer reg.lock.Unlock()
	fn, ok := reg.db[msg.Type]
	if ok {
		if err := fn(
			Request{
				Client:  c.id,
				Channel: msg.Channel,
				Type:    msg.Type,
				Raw:     body,
				server:  c.server,
			},
			&ResponseWriter{
				server: c.server,
			},
		); err != nil {
			c.server.WriteError(c.id, msg.Channel, err)
		}
	} else {
		c.server.WriteError(c.id, msg.Channel, errors.New("unknown type: "+msg.Type))
	}
}

func (reg *HandlerRegistry) Set(t string, fn TypeHandlerFunc) error {
	reg.lock.Lock()
	defer reg.lock.Unlock()
	if _, ok := reg.db[t]; ok {
		return fmt.Errorf("%s has already been registered", t)
	}
	reg.db[t] = fn
	return nil
}
