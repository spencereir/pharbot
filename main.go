package main


import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"github.com/nlopes/slack"
)

func formatRequest(r *http.Request) string {
	var request []string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	} 
	return strings.Join(request, "\n")
}

func main() {
	http.HandleFunc("/slash", func(w http.ResponseWriter, r *http.Request) {
		s, err := slack.SlashCommandParse(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
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
		case "/prod":
			HandleProdRequest(s, w)
			/*start_action := slack.AttachmentAction{Name: "start", Text: "Start Job", Type: "button"}
			start_attach := slack.Attachment{Text: "test", Actions: []slack.AttachmentAction{start_action}, CallbackID: "prod_start"}
			attachments := []slack.Attachment{start_attach}
			params := &slack.Msg{Text: s.Text, Attachments: attachments}
			b, err := json.Marshal(params)
			//fmt.Printf("%s\n", b)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)*/
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	
	http.HandleFunc("/interact", func(w http.ResponseWriter, r *http.Request) {
		cb := ParseInteractiveCallback(r)
		if (strings.HasPrefix(cb.CallbackID, "prod")) {
			HandleProdAction(cb, w)
		}
	})

	fmt.Println("[INFO] Server listening")
	http.ListenAndServe(":3000", nil)
}
