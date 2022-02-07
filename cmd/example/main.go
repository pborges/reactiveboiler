package main

import (
	"encoding/json"
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

	srv := websock.NewServer(log.Writer(), log.Flags())
	//srv.CentralLock.Log = log.Default()

	srv.Handle("jira.get", func(req websock.Request, rw *websock.ResponseWriter) {
		go jiraGet(req, rw)
	})
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

func backgroundJiraDemo(ws *websock.Server) {
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

func jiraGet(req websock.Request, rw *websock.ResponseWriter) {
	var issue string
	if req.Unpack(&issue) {
		if strings.HasPrefix(issue, "foobar") {
			rw.Error(errors.New("unknown issue: " + issue))
			return
		}

		var jira Jira
		lock.Lock()
		if j, ok := jiras[issue]; ok {
			jira = *j
		} else {
			lock.Unlock()
			time.Sleep(3 * time.Second)
			lock.Lock()
			jiras[issue] = &Jira{
				Issue:       issue,
				Description: "do work",
			}
			jira = *jiras[issue]
		}
		lock.Unlock()

		rw.Write("jira", jira)
	}
}
