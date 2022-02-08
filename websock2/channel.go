package websock2

import "encoding/json"

type Channel struct {
	Id            string
	Subscriptions []string
	*Stats
	log *Log
}

func (ch *Channel) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{}
	data["id"] = ch.Id
	data["subscriptions"] = ch.Subscriptions
	data["stats"] = ch.Stats
	return json.Marshal(data)
}
