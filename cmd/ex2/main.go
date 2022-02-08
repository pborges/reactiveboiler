package main

import (
	"fmt"
	"github.com/pborges/reactiveboiler/websock"
	"log"
	"net/http"
	"time"
)

func main() {
	srv := websock.NewServer()
	srv.Handle("jira.get", func(r websock.Request, rw websock.ResponseWriter) {
		var issue string
		if r.Unpack(&issue) {
			rw.Publish(fmt.Sprintf("jira.%s", issue), "jira", struct {
				Issue       string `json:"issue"`
				Description string `json:"description"`
			}{
				Issue:       issue,
				Description: "fkljdsaf",
			})
		}
	})

	go func() {
		issue := "devx-12"
		for {
			time.Sleep(1 * time.Second)
			srv.Publish(fmt.Sprintf("jira.%s", issue), "jira", struct {
				Issue       string `json:"issue"`
				Description string `json:"description"`
			}{
				Issue:       issue,
				Description: time.Now().String(),
			})
		}
	}()

	http.HandleFunc("/ws", srv.HandleUpgrade)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
