package websock

import "encoding/json"

type InboundMessage struct {
	Channel string          `json:"channel"`
	Type    string          `json:"type"`
	Body    json.RawMessage `json:"body"`
}

type OutboundMessage struct {
	Channel string      `json:"channel"`
	Type    string      `json:"type"`
	Body    interface{} `json:"body"`
}

type PublishMessage struct {
	Topic string
	Type  string      `json:"type"`
	Body  interface{} `json:"body"`
}

type MessageHandlerFunc func(r Request, rw ResponseWriter)

type Request struct {
	msg     InboundMessage
	writeCh chan<- OutboundMessage
	Raw     []byte
}

func (r Request) Unpack(dst interface{}) bool {
	if err := json.Unmarshal(r.Raw, dst); err != nil {
		r.writeCh <- OutboundMessage{
			Channel: r.msg.Channel,
			Type:    "error",
			Body:    err.Error(),
		}
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
	writeCh   chan<- OutboundMessage
	publishCh chan<- PublishMessage
	channel   string
}

func (rw *ResponseWriter) Publish(topic string, t string, body interface{}) {
	rw.publishCh <- PublishMessage{
		Topic: topic,
		Type:  t,
		Body:  body,
	}
}

func (rw *ResponseWriter) Write(t string, body interface{}) {
	rw.writeCh <- OutboundMessage{
		Channel: rw.channel,
		Type:    t,
		Body:    body,
	}
}

func (rw *ResponseWriter) Error(err error) {
}
