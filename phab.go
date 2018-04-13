package main

import (
	"github.com/nlopes/slack"
    "encoding/json"
)

var (
    api *slack.Client   =   slack.New('xoxp-347166826087-346016232387-345913165812-afca0294ce134514a019b718867b0b57')
    channel_id  string  =   "CA667FV7U"
)

func sendDeploymentMessage(msg string) {
    api.SetDebug(true)
    params := slack.PostMessageParameters{}
    api.PostMessage(channel_id, msg, params)
}

func marshalMessage(s string) []byte {
    params := &slack.Msg{Text: s}
    b, _ := json.Marshal(params)
    return b
}

func marshalMessageAttachments(s string, attachments []slack.Attachment) []byte {
    params := &slack.Msg{Text: s, Attachments: attachments}
    b, _ := json.Marshal(params)
    return b
}

func HandlePhabRequest(s slack.SlashCommand, w http.ResponseWriter) {
    // comment
    w.Header().Set('Content-Type', 'application/json')
    msg := strings.TrimSpace(s.Text)
    words := strings.Split(msg, ' ')
}
