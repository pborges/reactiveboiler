package main

import (
	"errors"
	"fmt"
	"github.com/pborges/reactiveboiler/websock"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime | log.Lshortfile)

	srv := websock.NewServer()
	//srv.CentralLock.Log = log.Default()

	srv.Handle("object.get", func(req websock.Request, rw websock.ResponseWriter) {
		go objectGet(req, rw)
	})
	go backgroundObjectDemo(srv)

	http.HandleFunc("/ws", srv.HandleUpgrade)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type Object struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var objects = map[string]*Object{}
var lock sync.Mutex

func backgroundObjectDemo(ws *websock.Server) {
	for {
		time.Sleep(5 * time.Second)
		lock.Lock()
		for _, obj := range objects {
			obj.Description = fmt.Sprintf("%s - %d", obj.Name, time.Now().Unix())
			ws.Publish(fmt.Sprintf("object.%s", obj.Name), "object", *obj)
		}
		lock.Unlock()
	}
}

func objectGet(req websock.Request, rw websock.ResponseWriter) {
	var name string
	if req.Unpack(&name) {
		if strings.HasPrefix(name, "foobar") {
			rw.Error(errors.New("unknown object: " + name))
			return
		}

		var obj Object
		lock.Lock()
		if j, ok := objects[name]; ok {
			obj = *j
		} else {
			lock.Unlock()
			time.Sleep(3 * time.Second)
			lock.Lock()
			objects[name] = &Object{
				Name:        name,
				Description: "do work",
			}
			obj = *objects[name]
		}
		lock.Unlock()

		rw.Write("object", obj)
	}
}
