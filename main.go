package main


import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"github.com/nlopes/slack"
)

func main() {
	var (
		verificationToken string
	)

	flag.StringVar(&verificationToken, "token", "Yxoxp-347166826087-346016232387-346121505410-a70ff9621df286da1a66442326c3de4", "Your Slash Verification Token")
	flag.Parse()

	http.HandleFunc("/slash", func(w http.ResponseWriter, r *http.Request) {
		s, err := slack.SlashCommandParse(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !s.ValidateToken(verificationToken) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch s.Command {
		case "/echo":
			params := &slack.Msg{Text: s.Text}
			b, err := json.Marshal(params)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	fmt.Println("[INFO] Server listening")
	http.ListenAndServe(":3000", nil)
}
/*
import (
    "fmt"
	"github.com/nlopes/slack"
)

func main() {
	api := slack.New("xoxp-347166826087-346016232387-346121505410-a70ff9621df286da1a66442326c3de4e")
    fmt.Printf("test")
	api.SetDebug(true)
    params := slack.PostMessageParameters{}
    api.PostMessage("CA60G6WRH", "hello friends", params)
}*/
