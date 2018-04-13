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
	helptexts	map[string]string	= map[string]string {
		"start": "*`/prod start`*: Start a new prod job.\n`/prod start <job id>` - start a previously run prod job, copying old parameters over\n`/prod start <job id> <oneoff> <writes> <primary read> <host> <command>` - start a new prod job, manually populating parameters\n`<job id>` must be a valid job ID (i.e., you have added it with `/prod new` or it shows up in `/prod search` or `/prod search`)\n`<oneoff>`, `<writes>`, `<primary read>` must be booleans; yes/no, true/false, 1/0 are accepted",
		"stop": "*`/prod stop`*: Stop a job given the execution ID. This should only be used when the interactive button times out. In this case, run the command with the provided execution ID",
		"list": "*`/prod list`*: List all active jobs. This includes jobs in the [start job / cancel] phase.",
		"new": "*`/prod new`*: Create a new prod job.\n`/prod new <phab task> <diff URI> <owner> <backup owner> <lead approver> <summary>` - create a new prod job with the listed parameters; also returns the ID of the job for use with `/prod start`.",
		"search": "*`/prod search`*: Search prod jobs, execution logs.\n`/prod search execution <query>` - search job execution logs for `query`\n`/prod search jobs <query>` - search prod jobs for `query`",
	}
	helpmsg		string			=
		"Pharbot: A simple bot to help out with (some) Phab and (mostly) Prod related things.\n`/prod start`: start a prod job\n`/prod new`: create a new prod job\n`/prod stop`: stop a prod job\n`/prod list`: list active prod jobs\n`/prod search`: search prod jobs / execution logs"
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

func serializeJobExecutionAndProdJob(exec JobExecution, job ProdJob) string {
	return fmt.Sprintf("*Job ID:* %v (%v)\n*Run User:* @%v\n*Oneoff:* %v\n*Writes*: %v\n*Primary Reads:* %v\n*Host:* `%v`\n*Command:* `%v`",
				exec.job_id, job.summary, exec.run_user, exec.one_off, exec.writes, exec.primary_read, exec.host, exec.command)
}

func serializeJobExecution(exec JobExecution) string {
	return fmt.Sprintf("*Job ID:* %v\n*Run User:* @%v\n*Oneoff:* %v\n*Writes*: %v\n*Primary Reads:* %v\n*Host:* `%v`\n*Command:* `%v`",
				exec.job_id, exec.run_user, exec.one_off, exec.writes, exec.primary_read, exec.host, exec.command)
}

// This has totally different behaviour from its transposed partner. Don't think too hard about it.
func serializeProdJobAndJobExecution(job ProdJob, exec JobExecution) string {
	if (job.owner == exec.run_user) {
		return fmt.Sprintf("Job notification on the behalf of @%v\n[prod] [job id %v] %v", job.owner, job.job_id, job.summary)
	} else {
		return fmt.Sprintf("Job notification on the behalf of @%v (Run User: @%v)\n[prod] [job id %v] %v", job.owner, exec.run_user, job.job_id, job.summary)
	}
}

func serializeProdJob(job ProdJob) string {
	return fmt.Sprintf("*ID:* %v\n*Summary:* %v\n*Owner:* @%v\n*Backup Owner:* @%v\n*Lead Approver:* @%v\n*Phab Task:* %v\n*Diff URI:* %v\n",
		job.job_id, job.summary, job.owner, job.backup_owner, job.lead_approver, job.phab_task, job.diff_uri)
}

func getProdJob(job_id int) ProdJob {
	for _, job := range prod_jobs {
		if job.job_id == job_id {
			return job
		}
	}
	return ProdJob{}
}

func HandleProdRequest(s slack.SlashCommand, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	msg := strings.TrimSpace(s.Text)
	words := strings.Split(msg, " ")

	switch len(words) {
	case 1:
		switch words[0] {
		case "":
			replyToSlash(s, helpmsg)
		case "help":
			replyToSlash(s, helpmsg)
		default:
			// oops, process it anyway
			if words[0] == "list" {
				msg := ""
				for _, v := range floating_execs {
					msg += fmt.Sprintf("@%v, %v\n", v.run_user, v.job_id)
				}
				if msg == "" {
					msg = "No active jobs"
				}
				replyToSlash(s, msg)
				return
			}
			if val, ok := helptexts[words[0]]; ok {
				replyToSlash(s, val)
			} else {
				replyToSlash(s, fmt.Sprintf("`%v` is not a valid command", words[0]))
			}
		}
	default:
		// Long enough message to do proper commands
		switch words[0] {
		case "help":
			if len(words) == 2 {
				if val, ok := helptexts[words[1]]; ok {
					replyToSlash(s, val)
				} else {
					replyToSlash(s, fmt.Sprintf("`%v` is not a valid command", words[1]))
				}
			} else {
				replyToSlash(s, "Help can only be called on one command at a time")
			}
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
			
			job := getProdJob(exec.job_id)
			if (job == ProdJob{}) {
				replyToSlash(s, fmt.Sprintf("Could not find a prod job with ID %v; perhaps try `/prod new`?", exec.job_id))
				return
			}

			start_action := slack.AttachmentAction{Name: "start", Value: "start", Text: "Start Job", Type: "button", Style: "primary"}
			cancel_action := slack.AttachmentAction{Name: "cancel", Value: "cancel", Text: "Cancel", Type: "button", Style: "danger"}
			start_attach := slack.Attachment{Text: serializeJobExecutionAndProdJob(exec, job), Actions: []slack.AttachmentAction{start_action, cancel_action}, CallbackID: fmt.Sprintf("prod_start_%v", exec.exec_id)}
			attachments := []slack.Attachment{start_attach}
			replyToSlashWithAttachments(s, "Please inspect the below job for correctness. Click 'Start Job' to add this job to the spreadsheet in a few minutes, and message #prod immediately. Click 'Cancel' to delete it.", attachments)
				
			floating_execs[exec.exec_id] = exec
			fmt.Printf("%v\n", msg_timestamp[exec.exec_id])
		case "search":
			replyToSlash(s, "Not implemented yet--sorry!")
		case "stop":
			exec_id, _ := strconv.Atoi(words[1])
			if _, ok := floating_execs[exec_id]; ok {
				for _, v := range execution_log {
					if v.exec_id == exec_id {
						v.end_time = time.Now()
					}
				}
				ts := msg_timestamp[exec_id]
				params := slack.PostMessageParameters{ThreadTimestamp: ts}
				msg := "Done"
				api.PostMessage(prod_channel_id, msg, params)
				delete(floating_execs, exec_id)
				replyToSlash(s, fmt.Sprintf("Job stopped."))
			} else {
				replyToSlash(s, "It appears this job has already been completed.")
			}
		case "new":
			new_prod_id := len(prod_jobs) + 1
			if len(words) < 7 {
				replyToSlash(s, "I can't parse that format. Please use the format `/prod new <phab task> <diff URI> <owner> <backup owner> <lead approver> <summary>`. For example, `/prod new https://phab.wish.com/T1234321 https://phab.wish.com/D1234321 jsmith jdoe jdoe Recalibrate Flux Capacitors`")
				return
			}
			phab_task := words[1]
			diff_uri := words[2]
			owner := words[3]
			backup_owner := words[4]
			lead_approver := words[5]
			summary := strings.Join(words[6:], " ")
			
			job := ProdJob{job_id: new_prod_id, phab_task: phab_task, diff_uri: diff_uri, owner: owner, backup_owner: backup_owner, lead_approver: lead_approver, summary: summary}
			//WriteProdJob(job)
			replyToSlash(s, fmt.Sprintf("Created prod job:\n%v", serializeProdJob(job)))
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
			job := getProdJob(exec.job_id)
			ts := sendProdMessage(fmt.Sprintf("%v\n", serializeProdJobAndJobExecution(job, exec)))
			msg_timestamp[exec_id] = ts
			exec.start_time = time.Now()
			execution_log = append(execution_log, exec)
			WriteExecution(exec)
			done_action := slack.AttachmentAction{Name: "done", Value: "done", Text: "Finish Job", Type: "button"}
			done_attach := slack.Attachment{Text: fmt.Sprintf("This button will expire in 30 minutes. If you would like to end the job after this time, please run `/prod stop %v`", exec_id), Actions: []slack.AttachmentAction{done_action}, CallbackID: cb.CallbackID}
			http.Post(cb.ResponseURL, "application/json", bytes.NewBuffer(marshalMessageAttachments("Thanks. Your message has been posted. The prod spreadsheet will update shortly. Click the button below when you have completed the job.", []slack.Attachment{done_attach})))
		} else if cb.Actions[0].Name == "cancel" {
			exec_id, _ := strconv.Atoi(cb.CallbackID[len("prod_start_"):])
			http.Post(cb.ResponseURL, "application/json", bytes.NewBuffer(marshalMessage("This job has been cancelled.")))
			delete(floating_execs, exec_id)
		} else {
			exec_id, _ := strconv.Atoi(cb.CallbackID[len("prod_start_"):])
			if _, ok := floating_execs[exec_id]; ok {
				for _, v := range execution_log {
					if v.exec_id == exec_id {
						v.end_time = time.Now()
					}
				}
				ts := msg_timestamp[exec_id]
				params := slack.PostMessageParameters{ThreadTimestamp: ts}
				msg := "Done"
				api.PostMessage(prod_channel_id, msg, params)
				http.Post(cb.ResponseURL, "application/json", bytes.NewBuffer(marshalMessage("Thanks! This job has been completed.")))
				delete(floating_execs, exec_id)
			} else {
				http.Post(cb.ResponseURL, "application/json", bytes.NewBuffer(marshalMessage("It appears this job has already been completed.")))
			}
		}
	}
}
