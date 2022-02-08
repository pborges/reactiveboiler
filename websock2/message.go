package websock2

import "encoding/json"

type MessageHandler func(req Request, rw *ResponseWriter)

type Request struct {
	client  *Client
	channel string
	Raw     []byte
}

func (r Request) Unpack(dst interface{}) bool {
	if err := json.Unmarshal(r.Raw, dst); err != nil {
		r.client.error(r.channel, err)
		return false
	}
	return true
}

type Message struct {
	Channel string      `json:"channel"`
	Type    string      `json:"type"`
	Body    interface{} `json:"body"`
}

type ResponseWriter struct {
	client  *Client
	channel string
}

func (rw *ResponseWriter) Write(t string, body interface{}) {
	rw.client.server.clientsLock.RLock()
	defer rw.client.server.clientsLock.RUnlock()
	rw.client.write(rw.channel, t, body)
}

func (rw *ResponseWriter) Error(err error) {
	rw.client.server.clientsLock.RLock()
	defer rw.client.server.clientsLock.RUnlock()
	rw.client.error(rw.channel, err)
}
