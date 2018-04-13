package main

import (
	"fmt"
	"net/http"
	"github.com/nlopes/slack"
	"strings"
)

var (
    channel_id  string  =   "CA667FV7U"
    error_string string  =   "Sorry, I don't understand that command. Try `/cherry-pick help` for more info."
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
    if len(words) == 0 {
        w.Write(marshalMessage(error_string))
    } else if len(words) == 1 {
        if strings.ToLower(words[0]) == "help" {
            w.Write(marshalMessage("Hi, I'm your friendly neighbourhood cherry pick bot!\n" +
            "Usage: `/cherry-pick <phab diff link> <sha> <tiers> <needs test?> <project lead for approval>` \n"))
        } else {
            w.Write(marshalMessage(error_string))
        }
    } else if len(words) < 5 {
        w.Write(marshalMessage(error_string))
    } else {
        last_index := len(words) - 1
        if words[last_index][0] != '@' {
            words[last_index] = "@" + words[last_index]
        }
        boolboy := strings.ToLower(words[last_index-1])
        if boolboy != "true" && boolboy != "false" && boolboy != "yes" && boolboy != "no" {
            w.Write(marshalMessage(fmt.Sprintf("Hmm... %v doesn't seem to be a boolean. Try yes/true/no/false for the needs test field.", words[last_index-1])))
        }
        pick_message := "Cherry-pick request from @" + s.UserName + ":\n" +
                       "Phab Diff: " + words[0] + "\n" +
                       "SHA: " + words[1] + "\n" +
                       "Tiers: " + strings.Join(words[2:last_index-1], ", ") + "\n" +
                       "Testing Required: " + words[last_index-1] + "\n" +
                       "Approver: " + words[last_index]
        sendDeploymentMessage(pick_message)
    }
}
