package websock

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
)

const SymbolInfo = "**"
const SymbolErr = "!!"
const SymbolSend = ">>"
const SymbolRecv = "<<"

func NewLog(out io.Writer, logFlags int, client string) *Log {
	return &Log{
		Writer: out,
		Flags:  logFlags,
		client: client,
		Info:   newLog(out, logFlags, SymbolInfo, client, ""),
		Err:    newLog(out, logFlags, SymbolErr, client, ""),
	}
}

func newLog(out io.Writer, logFlags int, symbol string, client string, channel string) *log.Logger {
	var identifier string
	if channel != "" {
		identifier = fmt.Sprintf("[%s -> %s]", client, channel)
	} else if client != "" {
		identifier = fmt.Sprintf("[%s]", client)
	}
	return log.New(out, fmt.Sprintf("[WS %s]%s ", symbol, identifier), logFlags)
}

type Log struct {
	Writer io.Writer
	Flags  int
	Info   *log.Logger
	Err    *log.Logger
	client string
	send   *log.Logger
	recv   *log.Logger
}

func (l Log) Send(m OutboundMessage) {
	var buf []byte
	if m.Body != nil {
		buf, _ = json.Marshal(m.Body)
	}
	newLog(l.Writer, l.Flags, SymbolSend, l.client, m.Channel).
		Output(2, fmt.Sprintf("%s %s", m.Type, string(buf)))
}

func (l Log) Recv(m InboundMessage) {
	var buf []byte
	if m.Body != nil {
		buf, _ = json.Marshal(m.Body)
	}
	newLog(l.Writer, l.Flags, SymbolRecv, l.client, m.Channel).
		Output(2, fmt.Sprintf("%s %s", m.Type, string(buf)))
}
