package websock2

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"io"
	"net"
	"net/http"
)

func NewServer(out io.Writer, logFlags int) *Server {
	srv := &Server{
		CentralLock: &CentralLocking{},
		clients:     map[string]*Client{},
		protoHandlers: map[string]MessageHandler{
			"open": nil,
			"close": func(req Request, rw *ResponseWriter) {
				req.client.closeChannel(req.channel)
			},
			"subscribe": func(req Request, rw *ResponseWriter) {
				var topic string
				if req.Unpack(&topic) {
					req.client.subscribe(req.channel, topic)
				}
			},
		},
		handlers: map[string]MessageHandler{},
		log:      NewLog(out, logFlags, "", ""),
	}
	srv.clientsLock = srv.CentralLock.CreateLock("server.clients")
	srv.handlersLock = srv.CentralLock.CreateLock("server.handlers")
	return srv
}

type Server struct {
	clients       map[string]*Client
	clientsLock   *Lock
	protoHandlers map[string]MessageHandler
	handlers      map[string]MessageHandler
	handlersLock  *Lock
	CentralLock   *CentralLocking
	log           *Log
}

func (srv *Server) MarshalJSON() ([]byte, error) {
	srv.clientsLock.RLock()
	srv.handlersLock.RLock()
	defer func() {
		srv.clientsLock.RUnlock()
		srv.handlersLock.RUnlock()
	}()

	clients := map[string]interface{}{}
	for _, c := range srv.clients {
		clients[c.id] = c
	}
	return json.Marshal(clients)
}

func (srv *Server) newClient(id string, conn net.Conn) *Client {
	srv.clientsLock.Lock()
	defer srv.clientsLock.Unlock()

	client := &Client{
		server:   srv,
		id:       id,
		r:        wsutil.NewReader(conn, ws.StateServerSide),
		w:        wsutil.NewWriter(conn, ws.StateServerSide, ws.OpText),
		conn:     conn,
		channels: map[string]*Channel{},
		iolock:   srv.CentralLock.CreateLock(id + ".io"),
		lock:     srv.CentralLock.CreateLock(id),
		log:      NewLog(srv.log.Writer, srv.log.Flags, id, ""),
	}
	client.enc = json.NewEncoder(client.w)
	client.dec = json.NewDecoder(client.r)
	srv.clients[id] = client
	srv.log.Info.Println("open client", id)
	return client
}

func (srv *Server) deleteClient(id string) {
	srv.clientsLock.Lock()
	defer srv.clientsLock.Unlock()

	srv.log.Info.Println("close client", id)
	delete(srv.clients, id)
	srv.CentralLock.Delete(id)
	srv.CentralLock.Delete(id + ".io")
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
	id := string(msg)

	client := srv.newClient(
		id,
		conn,
	)

	defer srv.deleteClient(id)

	for {
		if err = client.nextFrame(); err != nil {
			if !errors.Is(err, io.EOF) {
				srv.log.Err.Println(err)
			}
			return
		}
	}
}

// HandleDebug sucks and really needs to be fixed
func (srv *Server) HandleDebug(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	enc := json.NewEncoder(w)

	enc.SetIndent("", "  ")
	if err := enc.Encode(srv); err != nil {
		fmt.Fprintln(w, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (srv *Server) Write(client string, channel string, t string, body interface{}) {
	srv.clientsLock.RLock()
	defer srv.clientsLock.RUnlock()

	if c, ok := srv.clients[client]; ok {
		srv.clientsLock.RLock()
		c.write(channel, t, body)
		srv.clientsLock.RUnlock()
	}
}

func (srv *Server) Handle(t string, fn MessageHandler) {
	srv.handlersLock.Lock()
	defer srv.handlersLock.Unlock()

	srv.handlers[t] = fn
}

func (srv *Server) Publish(topic string, t string, body interface{}) {
	srv.clientsLock.RLock()
	defer srv.clientsLock.RUnlock()

	for _, client := range srv.clients {
		client.lock.RLock()
		for chName, ch := range client.channels {
			for _, sub := range ch.Subscriptions {
				if sub == topic {
					client.write(chName, t, body)
				}
			}
		}
		client.lock.RUnlock()
	}
}
