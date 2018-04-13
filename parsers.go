package main

import (
	"encoding/json"
	"net/http"
	"github.com/nlopes/slack"
)

func ParseInteractiveCallback(r *http.Request) slack.AttachmentActionCallback {
	payload := r.PostForm.Get("payload")
	s := slack.AttachmentActionCallback{}
	_ = json.Unmarshal([]byte(payload), &s)
	return s
}
