package main

import (
	"fmt"
	"encoding/json"
	"net/http"
	"time"
	"strings"
	"math/rand"
	"strconv"
	"os"
	"bytes"
	"github.com/nlopes/slack"
)

type ProdJob struct {
	job_id		int
	phab_task	string
	summary		string
	owner		string
	backup_owner	string
	lead_approver	string
	diff_uri	string
}

type JobExecution struct {
	exec_id		int
	start_time	time.Time
	end_time	time.Time
	job_id		int
	run_user	string
	one_off		bool
	writes		bool
	primary_read	bool
	host		string
	command		string
}

var (
	prod_jobs 	[]ProdJob		= []ProdJob{}
	execution_log 	[]JobExecution		= []JobExecution{}
	api		*slack.Client 		= slack.New(os.Getenv("SLACK_TOKEN"))
	prod_channel_id string			= "CA60G6WRH"
	floating_execs	map[int]JobExecution	= make(map[int]JobExecution)
	msg_timestamp	map[int]string		= make(map[int]string)
)

func sendProdMessage(msg string) string {
	params := slack.PostMessageParameters{LinkNames: 1, Markdown: true}
	_, timestamp, _ := api.PostMessage(prod_channel_id, msg, params)
	fmt.Printf("sendProdMessage: %v\n", timestamp)
	return timestamp
}

func replyToSlash(s slack.SlashCommand, msg string) string {
	timestamp, _ := api.PostEphemeral(s.ChannelID, s.UserID, slack.MsgOptionText(msg, false))
	fmt.Printf("replyToSlash: %v\n", timestamp)
	return timestamp
}

func replyToSlashWithAttachments(s slack.SlashCommand, msg string, attachments []slack.Attachment) string {
	timestamp, _ := api.PostEphemeral(s.ChannelID, s.UserID, slack.MsgOptionText(msg, false), slack.MsgOptionAttachments(attachments...))
	fmt.Printf("replyToSlashA: %v\n", timestamp)
	return timestamp
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

func generateExecId() int {
	// I... am satisfied with my chances
	return rand.Int()	
}

func generateExecution(job_id int, user string, one_off, writes, primary_read bool, host, command string) JobExecution {
	return JobExecution{exec_id: generateExecId(), job_id: job_id, run_user: user, one_off: one_off, writes: writes,
				primary_read: primary_read, host: host, command: command}		
}

func generateExecutionFromPreviousExecution(job_id int, user string) JobExecution {
	for _, job := range execution_log {
		if job.job_id == job_id {
			return JobExecution{exec_id: generateExecId(), job_id: job_id, run_user: user, one_off: job.one_off, writes: job.writes,
						primary_read: job.primary_read, host: job.host, command: job.command}
		}
	}
	return JobExecution{exec_id: -1}
}

func serializeJobExecution(exec JobExecution) string {
	return fmt.Sprintf("*Job ID:* %v\n*Run User:* @%v\n*Oneoff:* %v\n*Writes*: %v\n*Primary Reads:* %v\n*Host:* `%v`\n*Command:* `%v`",
				exec.job_id, exec.run_user, exec.one_off, exec.writes, exec.primary_read, exec.host, exec.command)
}

func HandleProdRequest(s slack.SlashCommand, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	msg := strings.TrimSpace(s.Text)
	words := strings.Split(msg, " ")

	switch len(words) {
	case 1:
		switch words[0] {
		case "":
			w.Write(marshalMessage("Generic help message"))
		default:
			// Write help message for specific command and exit
			w.Write(marshalMessage(fmt.Sprintf("Help message specific to %s", words[0])))
		}
	default:
		// Long enough message to do proper commands
		switch words[0] {
		case "start":

			exec := JobExecution{}

			// assume the format is /prod start <job id>
			// search execution log to pull a similar job
			// if we can't find one, inform the user and ask for the full format
			// /prod start <job id> <oneoff> <writes> <primary read> <host> <command>
			if len(words) == 2 {
				job_id, err := strconv.Atoi(words[1])
				if err != nil {
					replyToSlash(s, fmt.Sprintf("Couldn't parse '%s' as a job ID. Please run `/prod start` or `/prod help start` for usage notes", words[1]))
					return
				}
				exec = generateExecutionFromPreviousExecution(job_id, s.UserName)
				if (exec.exec_id < 0) {
					replyToSlash(s, fmt.Sprintf("It looks like job ID %v doesn't have any executions on record. Please create one with `/prod start %v <oneoff> <writes> <primary read> <host> <command>`. For example, `/prod start %v no no yes merchant-backend-master merch-dbshell`", job_id, job_id, job_id))
					return
				}
			} else {
				if len(words) < 7 {
					replyToSlash(s, "I can't parse that format. Please use the command in the form `/prod start <job id>` or `/prod start <job id> <oneoff> <writes> <primary read> <host> <command>`")
					return
				}
				job_id, err := strconv.Atoi(words[1])
				if err != nil {
					replyToSlash(s, fmt.Sprintf("Couldn't parse '%v' as a job ID. Please run `/prod start` or `/prod help start` for usage notes", words[1]))
					return
				}
				oneoff := false
				if words[2] == "yes" || words[2] == "true" || words[2] == "1" {
					oneoff = true
				} else if words[2] == "no" || words[2] == "false" || words[2] == "0" {
					oneoff = false
				} else {
					replyToSlash(s, fmt.Sprintf("Couldn't parse '%v' as a boolean. Please use one of yes/no, true/false, 1/0", words[2]))
					return
				}
				writes := false
				if words[3] == "yes" || words[3] == "true" || words[3] == "1" {
					writes = true
				} else if words[3] == "no" || words[3] == "false" || words[3] == "0" {
					writes = false
				} else {
					replyToSlash(s, fmt.Sprintf("Couldn't parse '%v' as a boolean. Please use one of yes/no, true/false, 1/0", words[3]))
					return
				}
				primary_read := false
				if words[4] == "yes" || words[4] == "true" || words[4] == "1" {
					primary_read = true
				} else if words[4] == "no" || words[4] == "false" || words[4] == "0" {
					primary_read = false
				} else {
					replyToSlash(s, fmt.Sprintf("Couldn't parse '%v' as a boolean. Please use one of yes/no, true/false, 1/0", words[4]))
					return
				}
				host := words[5]
				command := strings.Join(words[6:], " ")
				exec = generateExecution(job_id, s.UserName, oneoff, writes, primary_read, host, command)
			}
			
			start_action := slack.AttachmentAction{Name: "start", Value: "start", Text: "Start Job", Type: "button", Style: "primary"}
			cancel_action := slack.AttachmentAction{Name: "cancel", Value: "cancel", Text: "Cancel", Type: "button", Style: "danger"}
			start_attach := slack.Attachment{Text: serializeJobExecution(exec), Actions: []slack.AttachmentAction{start_action, cancel_action}, CallbackID: fmt.Sprintf("prod_start_%v", exec.exec_id)}
			attachments := []slack.Attachment{start_attach}
			replyToSlashWithAttachments(s, "Please inspect the below job for correctness. Click 'Start Job' to add this job to the spreadsheet in a few minutes, and message #prod immediately. Click 'Cancel' to delete it.", attachments)
			floating_execs[exec.exec_id] = exec
			fmt.Printf("%v\n", msg_timestamp[exec.exec_id])
		case "search":
			w.Write(marshalMessage("Not implemented yet. Sorry!"))
		}
	}	
}

func HandleProdAction(cb slack.AttachmentActionCallback, w http.ResponseWriter) {
	api.SetDebug(true)
	fmt.Printf("%v\n%v\n", cb.CallbackID, len(cb.Actions))
	for _, v := range cb.Actions {
		fmt.Printf("%v\n", v.Name)
	}
	if strings.HasPrefix(cb.CallbackID, "prod_start_") {
		if cb.Actions[0].Name == "start" {
			exec_id, _ := strconv.Atoi(cb.CallbackID[len("prod_start_"):])
			exec := floating_execs[exec_id]
			fmt.Printf("Send message to prod")
			sendProdMessage(fmt.Sprintf("Start Prod Job:\n%v", serializeJobExecution(exec)))
			http.Post(cb.ResponseURL, "application/json", bytes.NewBuffer(marshalMessage("Thanks. Your message has been posted. The prod spreadsheet will update shortly.")))
		} else {
			http.Post(cb.ResponseURL, "application/json", bytes.NewBuffer(marshalMessage("This job has been cancelled.")))
		}
	}
}

func SyncProdJobs() {
	// TODO: sync with Google Sheets
}

func SyncExecutionLog() {
	// TODO: sync with google sheets
}
