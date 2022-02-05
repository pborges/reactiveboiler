package main

import (
	"errors"
	"fmt"
	"github.com/pborges/reactiveboiler/websock"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Jira struct {
	Issue       string `json:"issue"`
	Description string `json:"description"`
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime | log.Lshortfile)

	ws := websock.NewServer(os.Stdout, log.Ltime|log.Lmsgprefix)

	jiras := map[string]*Jira{}

	go func() {
		for {
			for _, j := range jiras {
				time.Sleep(5 * time.Second)
				j.Description = fmt.Sprintf("%s - %d", j.Issue, time.Now().Unix())
				ws.Publish(fmt.Sprintf("jira.%s", j.Issue), "jira", j)
			}
		}
	}()

	ws.HandleFunc("jira.get", func(req websock.Request) error {
		var issue string
		if req.Unpack(&issue) {
			if strings.HasPrefix(issue, "foobar") {
				return errors.New("unknown issue: " + issue)
			}

			if _, ok := jiras[issue]; !ok {
				jiras[issue] = &Jira{
					Issue:       issue,
					Description: "do work",
				}
			}
			time.Sleep(250 * time.Millisecond)
			req.Client.Write(req.Channel, "jira", jiras[issue])
		}
		return nil
	})

	http.HandleFunc("/", debugInfo(ws))
	http.HandleFunc("/ws", ws.HandleWS)

	log.Fatalln(http.ListenAndServe(":8080", nil))
}

func debugInfo(ws *websock.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		fmt.Fprintln(w, "<html><head><style>")
		fmt.Fprint(w,
			`
table {
	border:1px solid black;
	border-collapse: collapse;
	width: 100%;
}
table tr th {
	text-align: left
}
`)
		fmt.Fprintln(w, `border:1px solid black;border-collapse: collapse;width: 100%;`)
		fmt.Fprintln(w, "}")

		fmt.Fprintln(w, "</style></head><body>")

		clients := ws.Clients()
		sort.Slice(clients, func(i, j int) bool {
			return strings.Compare(clients[i].Id(), clients[j].Id()) > 0
		})
		for _, c := range clients {
			channels := c.Channels()
			sort.Slice(channels, func(i, j int) bool {
				return strings.Compare(channels[i].Id(), channels[j].Id()) > 0
			})

			fmt.Fprintln(w, `<table>`)
			fmt.Fprintf(w, "<tr><th>CLIENT</th><th>%s</th><th>Sent(%d) %s</th><th>Recv(%d) %s</th><th>Subscriptions</th></tr>",
				c.Id(),
				c.Stats.Sent,
				c.Stats.LastSent.Format("15:04:05 MST"),
				c.Stats.Received,
				c.Stats.LastReceived.Format("15:04:05 MST"),
			)
			for _, ch := range channels {
				fmt.Fprintf(w, "<tr><th>CHANNEL</th><td>%s</td><td>(%d) %s</td><td>(%d) %s</td><td>%s</td></tr>",
					ch.Id(),
					ch.Stats.Sent,
					ch.Stats.LastSent.Format("15:04:05 MST"),
					ch.Stats.Received,
					ch.Stats.LastReceived.Format("15:04:05 MST"),
					strings.Join(ch.Subscriptions, ","),
				)
			}
			fmt.Fprintln(w, "</table><br>")
		}
		fmt.Fprintln(w, "</html></body>")
	}
}
