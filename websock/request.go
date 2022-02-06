package websock

import "encoding/json"

type TypeHandlerFunc func(req Request, rw *ResponseWriter) error

type Message struct {
	Client  string      `json:"client"`
	Channel string      `json:"channel,omitempty"`
	Type    string      `json:"type"`
	Body    interface{} `json:"body"`
}

type Request struct {
	Client  string
	Channel string
	Type    string
	Raw     json.RawMessage
	server  *Server
}

type ResponseWriter struct {
	server *Server
}

func (rw *ResponseWriter) Write(client string, channel string, t string, body interface{}) {
	rw.server.Write(client, channel, t, body)
}

func (rw *ResponseWriter) WriteError(client string, channel string, err error) {
	rw.server.WriteError(client, channel, err)
}

func (rw *ResponseWriter) Publish(topic string, t string, body interface{}) {
	rw.server.Publish(topic, t, body)
}

func (r Request) Unpack(dst interface{}) bool {
	if err := json.Unmarshal(r.Raw, dst); err != nil {
		r.server.WriteError(r.Client, r.Channel, err)
		return false
	}
	return true
}
