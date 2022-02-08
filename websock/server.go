package websock

import (
	"encoding/json"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"log"
	"net/http"
	"os"
	"sync"
)

func NewServer() *Server {
	srv := Server{
		log:          NewLog(os.Stdout, log.Ltime|log.Lshortfile, ""),
		addClient:    make(chan Client),
		removeClient: make(chan Client),
		clients:      map[string]*Client{},
		publishCh:    make(chan PublishMessage),
		handlers:     map[string]MessageHandlerFunc{},
	}

	go srv.handle()
	return &srv
}

type Server struct {
	log          *Log
	clients      map[string]*Client
	handlers     map[string]MessageHandlerFunc
	publishCh    chan PublishMessage
	addClient    chan Client
	removeClient chan Client
}

func (srv *Server) loadHandler(t string) (MessageHandlerFunc, bool) {
	fn, ok := srv.handlers[t]
	return fn, ok
}

func (srv *Server) handle() {
	for {
		select {
		case msg := <-srv.publishCh:
			var wg sync.WaitGroup
			for _, client := range srv.clients {
				wg.Add(1)
				go func(client *Client) {
					defer wg.Done()
					client.subscriptionCh <- msg
				}(client)
			}
			wg.Wait()
		case client := <-srv.addClient:
			client.Log.Info.Println("open", client.id)
			client.handlerFn = srv.loadHandler
			srv.clients[client.id] = &client
			go client.handle()
		case client := <-srv.removeClient:
			delete(srv.clients, client.id)
			client.Log.Info.Println("close", client.id)
		}
	}
}

func (srv *Server) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		srv.log.Err.Println("unable to upgrade client", err)
		return
	}

	defer conn.Close()

	msg, _, err := wsutil.ReadClientData(conn)
	if err != nil {
		srv.log.Err.Println("unable to read initial frame", err)
		return
	}

	readCh := make(chan InboundMessage)
	writeCh := make(chan OutboundMessage)

	connR := wsutil.NewReader(conn, ws.StateServerSide)
	connDec := json.NewDecoder(connR)

	connW := wsutil.NewWriter(conn, ws.StateServerSide, ws.OpText)
	connEnc := json.NewEncoder(connW)

	client := Client{
		id:             string(msg),
		Log:            NewLog(srv.log.Writer, srv.log.Flags, string(msg)),
		channels:       map[string]*Channel{},
		conn:           conn,
		readCh:         readCh,
		writeCh:        writeCh,
		publishCh:      srv.publishCh,
		subscriptionCh: make(chan PublishMessage),
		openRequests:   new(sync.WaitGroup),
	}
	srv.addClient <- client

	go func() {
		for {
			msg, ok := <-writeCh
			if ok {
				client.Log.Send(msg)
				if err := connEnc.Encode(msg); err != nil {
					srv.log.Err.Println("error encoding frame", err)
					return
				}
				if err := connW.Flush(); err != nil {
					srv.log.Err.Println("error flushing frame", err)
					return
				}
			} else {
				return
			}
		}
	}()

	defer func() {
		srv.removeClient <- client
		client.Log.Info.Println("waiting for open requests to finish...")
		client.openRequests.Wait()
		srv.log.Info.Println("close", client.id)
		close(writeCh)
	}()

	for {
		hdr, err := connR.NextFrame()
		if err != nil {
			srv.log.Err.Println("error reading frame", err)
			return
		}

		if hdr.OpCode == ws.OpClose {
			return
		}

		var req InboundMessage
		if err := connDec.Decode(&req); err != nil {
			srv.log.Err.Println("error decoding frame", err)
			return
		}
		client.Log.Recv(req)
		readCh <- req
	}
}

func (srv *Server) Handle(t string, fn MessageHandlerFunc) {
	srv.handlers[t] = fn
}

func (srv *Server) Publish(topic string, t string, body interface{}) {
	srv.publishCh <- PublishMessage{
		Topic: topic,
		Type:  t,
		Body:  body,
	}
}
