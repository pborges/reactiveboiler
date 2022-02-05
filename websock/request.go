package websock

import "encoding/json"

type TypeHandlerFunc func(req Request) error

type Message struct {
	Client  string      `json:"client"`
	Channel string      `json:"channel,omitempty"`
	Type    string      `json:"type"`
	Body    interface{} `json:"body"`
}

type Request struct {
	Client  *Client
	Channel string
	Type    string
	Raw     json.RawMessage
}

func (r Request) Unpack(dst interface{}) bool {
	if err := json.Unmarshal(r.Raw, dst); err != nil {
		r.Client.Error(r.Channel, err)
		return false
	}
	return true
}
