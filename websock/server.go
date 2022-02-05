package websock

import (
	"io"
	"net/http"
	"sync"
)

func NewServer(out io.Writer, logFlags int) *Server {
	s := Server{
		handlers: HandlerRegistry{
			db: map[string]TypeHandlerFunc{},
		},
		clients: ClientRegistry{
			db: map[string]*Client{},
		},
		log: NewLog(out, logFlags, "", ""),
	}
	s.HandleFunc("subscribe", func(req Request) error {
		var topic string
		if req.Unpack(&topic) {
			req.Client.lock.Lock()
			req.Client.channels[req.Channel].Subscriptions = append(req.Client.channels[req.Channel].Subscriptions, topic)
			req.Client.lock.Unlock()
		}
		return nil
	})
	s.HandleFunc("open", func(req Request) error {
		return nil
	})
	s.HandleFunc("close", func(req Request) error {
		req.Client.lock.Lock()
		delete(req.Client.channels, req.Channel)
		req.Client.lock.Unlock()
		return nil
	})
	return &s
}

type Server struct {
	handlers HandlerRegistry
	clients  ClientRegistry
	log      *Log
	lock     sync.Mutex
}

func (s *Server) HandleFunc(t string, fn TypeHandlerFunc) {
	if err := s.handlers.Set(t, fn); err != nil {
		panic(err)
	}
}

func (s *Server) Clients() []*Client {
	return s.clients.List()
}

func (s *Server) Write(client string, channel string, t string, body interface{}) {
	if c := s.clients.Get(client); c != nil {
		c.Write(channel, t, body)
	}
}

func (s *Server) Publish(topic string, t string, body interface{}) {
	for _, client := range s.Clients() {
		for chName, ch := range client.channels {
			for _, sub := range ch.Subscriptions {
				if sub == topic {
					client.write(chName, t, body)
				}
			}
		}
	}
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Err.Println("unable to upgrade", err)
		return
	}

	defer func() {
		conn.Close()
	}()

	_, buff, err := conn.ReadMessage()
	if err != nil {
		s.log.Err.Println("unable to read hello message", err)
		return
	}

	client := &Client{
		id:       string(buff),
		conn:     conn,
		lock:     new(sync.Mutex),
		channels: map[string]*Channel{},
		server:   s,
		log:      NewLog(s.log.Writer, s.log.Flags, string(buff), ""),
	}

	s.clients.Set(client)

	defer func() {
		s.clients.Delete(client.id)
	}()

	client.Handle()
}
