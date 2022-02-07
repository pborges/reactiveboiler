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

func NewLog(out io.Writer, logFlags int, client string, channel string) *Log {
	newLog := func(symbol, client, channel string) *log.Logger {
		var identifier string
		if channel != "" {
			identifier = fmt.Sprintf("[%s -> %s]", client, channel)
		} else if client != "" {
			identifier = fmt.Sprintf("[%s]", client)
		}
		return log.New(out, fmt.Sprintf("[WS %s]%s ", symbol, identifier), logFlags)
	}

	return &Log{
		Writer: out,
		Flags:  logFlags,
		Info:   newLog(SymbolInfo, client, channel),
		Err:    newLog(SymbolErr, client, channel),
		send:   newLog(SymbolSend, client, channel),
		recv:   newLog(SymbolRecv, client, channel),
	}
}

type Log struct {
	Writer io.Writer
	Flags  int
	Info   *log.Logger
	Err    *log.Logger
	send   *log.Logger
	recv   *log.Logger
}

func (l Log) Send(m Message) {
	var buf []byte
	if m.Body != nil {
		buf, _ = json.Marshal(m.Body)
	}
	l.send.Output(2, fmt.Sprintf("%s %s", m.Type, string(buf)))
}

func (l Log) Recv(m Message) {
	var buf []byte
	if m.Body != nil {
		buf, _ = json.Marshal(m.Body)
	}
	l.recv.Output(2, fmt.Sprintf("%s %s", m.Type, string(buf)))
}
