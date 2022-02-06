package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pborges/reactiveboiler/websock2"
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

	srv := websock2.NewServer(log.Writer(), log.Flags())
	//srv.CentralLock.Log = log.Default()

	srv.Handle("jira.get", jiraGet)
	go backgroundJiraDemo(srv)

	http.HandleFunc("/clients", srv.HandleDebug)
	http.HandleFunc("/lock", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(srv.CentralLock)
	})
	http.HandleFunc("/ws", srv.HandleUpgrade)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type Jira struct {
	Issue       string `json:"issue"`
	Description string `json:"description"`
}

var jiras = map[string]*Jira{}
var lock sync.Mutex

func backgroundJiraDemo(ws *websock2.Server) {
	for {
		time.Sleep(5 * time.Second)
		lock.Lock()
		for _, jira := range jiras {
			jira.Description = fmt.Sprintf("%s - %d", jira.Issue, time.Now().Unix())
			ws.Publish(fmt.Sprintf("jira.%s", jira.Issue), "jira", *jira)
		}
		lock.Unlock()
	}
}

func jiraGet(req websock2.Request, rw *websock2.ResponseWriter) {
	var issue string
	if req.Unpack(&issue) {
		if strings.HasPrefix(issue, "foobar") {
			rw.Error(errors.New("unknown issue: " + issue))
			return
		}

		if _, ok := jiras[issue]; !ok {
			lock.Lock()
			jiras[issue] = &Jira{
				Issue:       issue,
				Description: "do work",
			}
			lock.Unlock()
		}
		lock.Lock()
		rw.Write("jira", jiras[issue])
		lock.Unlock()
	}
}
