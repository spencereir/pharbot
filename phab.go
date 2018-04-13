package main

import (
	"net/http"
	"github.com/nlopes/slack"
	"strings"
)

var (
    channel_id  string  =   "CA667FV7U"
)

func sendDeploymentMessage(msg string) {
    api.SetDebug(true)
    params := slack.PostMessageParameters{}
    api.PostMessage(channel_id, msg, params)
}

func HandlePhabRequest(s slack.SlashCommand, w http.ResponseWriter) {
    // comment
    w.Header().Set("Content-Type", "application/json")
    msg := strings.TrimSpace(s.Text)
    words := strings.Split(msg, " ")
    if len(words) > 0 {
	return
    }
}
