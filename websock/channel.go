package websock

import (
	"encoding/json"
)

type Channel struct {
	id            string
	Stats         Stats    `json:"stats"`
	Subscriptions []string `json:"subscriptions,omitempty"`
	log           *Log
}

func (s Channel) Id() string {
	return s.id
}

func (s Channel) MarshalJSON() ([]byte, error) {
	type alias Channel
	a := struct {
		alias
		Id string `json:"id"`
	}{
		alias: alias(s),
		Id:    s.id,
	}
	return json.Marshal(a)
}
