package main

import (
        "encoding/json"
        "fmt"
        "io/ioutil"
        "log"
        "net/http"
        "os"
	"strings"
        "strconv"
        "time"

        "golang.org/x/net/context"
        "golang.org/x/oauth2"
        "golang.org/x/oauth2/google"
        "google.golang.org/api/sheets/v4"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
        tokFile := "token.json"
        tok, err := tokenFromFile(tokFile)
        if err != nil {
                tok = getTokenFromWeb(config)
                saveToken(tokFile, tok)
        }
        return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
        authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
        fmt.Printf("Go to the following link in your browser then type the "+
                "authorization code: \n%v\n", authURL)

        var authCode string
        if _, err := fmt.Scan(&authCode); err != nil {
                log.Fatalf("Unable to read authorization code: %v", err)
        }

        tok, err := config.Exchange(oauth2.NoContext, authCode)
        if err != nil {
                log.Fatalf("Unable to retrieve token from web: %v", err)
        }
        return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
        f, err := os.Open(file)
        defer f.Close()
        if err != nil {
                return nil, err
        }
        tok := &oauth2.Token{}
        err = json.NewDecoder(f).Decode(tok)
        return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
        fmt.Printf("Saving credential file to: %s\n", path)
        f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
        defer f.Close()
        if err != nil {
                log.Fatalf("Unable to cache oauth token: %v", err)
        }
        json.NewEncoder(f).Encode(token)
}

func LoadSheets() {
        b, err := ioutil.ReadFile("client_secret.json")
        if err != nil {
                log.Fatalf("Unable to read client secret file: %v", err)
        }

        // If modifying these scopes, delete your previously saved client_secret.json.
        config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
        if err != nil {
                log.Fatalf("Unable to parse client secret file to config: %v", err)
        }
        client := getClient(config)

        srv, err := sheets.New(client)
        if err != nil {
                log.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

        // Prints the names and majors of students in a sample spreadsheet:
        // https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit
        spreadsheetId := "1B84DImukPyqhSMDJmpE_lFZakMrAktdPfxp-emrR6Gc"
        readRange := "Run Job List!A3:G"
        resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
        if err != nil {
                log.Fatalf("Unable to retrieve data from sheet: %v", err)
        }

        if len(resp.Values) == 0 {
                fmt.Println("No data found.")
        } else {
                for _, row := range resp.Values {
                        job_id := -1
                        phab_task := ""
                        summary := ""
                        owner := ""
                        backup := ""
                        lead_approver := ""
                        diff_uri := ""
                        if len(row) == 0 {
                            continue
                        }
                        job_id, _ = strconv.Atoi(row[0].(string))
                        if len(row) > 1 {
                            phab_task = row[1].(string)
                        }
                        if len(row) > 2 {
                            summary = row[2].(string)
                        }
                        if len(row) > 3 {
                            owner = row[3].(string)
                        }
                        if len(row) > 4 {
                            backup = row[4].(string)
                        }
                        if len(row) > 5 {
                            lead_approver = row[5].(string)
                        }
                        if len(row) > 6 {
                            diff_uri = row[6].(string)
                        }
                        job := ProdJob{job_id: job_id, phab_task: phab_task, summary: summary, owner: owner, backup_owner: backup, lead_approver: lead_approver, diff_uri: diff_uri}
                        prod_jobs = append(prod_jobs, job)
                }
        }

        readRange = "Execution Audit Log!A3:L"
        resp, err = srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
        if err != nil {
            log.Fatalf("Unable to retrieve data from sheet: %v", err)
        } else {
            for id, row := range resp.Values {
                exec_id := id
                start_time := time.Now()
                end_time := time.Now()
                job_id := -1
                run_user := ""
                one_off := false
                writes := false
                primary_read := false
                host := ""
                command := ""

                if len(row) == 0 {
                    continue
                }
                // TODO: figure out how to handle start
                // start_time = row[0].(string)
                if len(row) > 1 {
                    // same with start
                    // end_time = row[1].(string)
                }
                if len(row) > 2 {
                    job_id, _ = strconv.Atoi(row[2].(string))
                }
                if len(row) > 3 {
                    run_user = row[3].(string)
                }
                if len(row) > 4 {
                    one_off = strings.ToLower(row[4].(string)) != "no"
                }
                if len(row) > 5 {
                    writes = strings.ToLower(row[5].(string)) != "no"
                }
                if len(row) > 6 {
                    primary_read = strings.ToLower(row[6].(string)) != "no"
                }
                if len(row) > 7 {
                    host = row[7].(string)
                }
                if len(row) > 8 {
                    command = row[8].(string)
                }

                exec := JobExecution{exec_id: exec_id, start_time: start_time, end_time: end_time, job_id: job_id, run_user: run_user, one_off: one_off, writes: writes, primary_read: primary_read, host: host, command: command}

                execution_log = append(execution_log, exec)
            }
        }
}

func WriteExecution(exec JobExecution) {
    b, err := ioutil.ReadFile("client_secret.json")
    if err != nil {
        log.Fatalf("Unable to read client secret file: %v", err)
    }

    config, _ := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")

    c := getClient(config)

    sheetsService, _ := sheets.New(c)

    spreadsheetId := "1B84DImukPyqhSMDJmpE_lFZakMrAktdPfxp-emrR6Gc"

    rangeData := "Execution Audit Log!A3:L"

    is_one_off := ""
    if exec.one_off {
        is_one_off = "Yes"
    } else {
        is_one_off = "No"
    }
    is_write := ""
    if exec.writes {
        is_write = "Yes"
    } else {
        is_write = "No"
    }
    is_read := ""
    if exec.primary_read {
        is_read = "Yes"
    } else {
        is_read = "No"
    }

    var values [][]interface{}
    var one_row []interface{}
    values = append(values, one_row)
    values[0] = append(values[0], exec.start_time)
    values[0] = append(values[0], exec.end_time)
    values[0] = append(values[0], exec.job_id)
    values[0] = append(values[0], exec.run_user)
    values[0] = append(values[0], is_one_off)
    values[0] = append(values[0], is_write)
    values[0] = append(values[0], is_read)
    values[0] = append(values[0], exec.host)
    values[0] = append(values[0], exec.command)

    rb := &sheets.ValueRange{
        Range: rangeData,
        Values: values,
    }

    valueInputOption := "USER_ENTERED"
    insertDataOption := "INSERT_ROWS"
    ctx := context.Background()
    resp, err := sheetsService.Spreadsheets.Values.Append(spreadsheetId, rangeData, rb).ValueInputOption(valueInputOption).InsertDataOption(insertDataOption).Context(ctx).Do()
    if err != nil {
        fmt.Printf("%v\n", err)
    }
    fmt.Printf("%v\n", resp)
}

func WriteProdJob(job ProdJob) {
    b, err := ioutil.ReadFile("client_secret.json")
    if err != nil {
        log.Fatalf("Unable to read client secret file: %v", err)
    }

    config, _ := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")

    c := getClient(config)

    sheetsService, _ := sheets.New(c)

    spreadsheetId := "1B84DImukPyqhSMDJmpE_lFZakMrAktdPfxp-emrR6Gc"

    rangeData := "Run Job List!A3:G"

    var values [][]interface{}
    var one_row []interface{}
    values = append(values, one_row)
    values[0] = append(values[0], job.job_id)
    values[0] = append(values[0], job.phab_task)
    values[0] = append(values[0], job.summary)
    values[0] = append(values[0], job.owner)
    values[0] = append(values[0], job.backup_owner)
    values[0] = append(values[0], job.lead_approver)
    values[0] = append(values[0], job.diff_uri)

    rb := &sheets.ValueRange{
        Range: rangeData,
        Values: values,
    }

    valueInputOption := "USER_ENTERED"
    insertDataOption := "INSERT_ROWS"
    ctx := context.Background()
    resp, err := sheetsService.Spreadsheets.Values.Append(spreadsheetId, rangeData, rb).ValueInputOption(valueInputOption).InsertDataOption(insertDataOption).Context(ctx).Do()
    if err != nil {
        fmt.Printf("%v\n", err)
    }
    fmt.Printf("%v\n", resp)
}

func MarkExecCompleted(exec JobExecution) {
    b, err := ioutil.ReadFile("client_secret.json")
    if err != nil {
        log.Fatalf("Unable to read client secret file: %v", err)
    }

    config, _ := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")

    c := getClient(config)

    sheetsService, _ := sheets.New(c)

    spreadsheetId := "1B84DImukPyqhSMDJmpE_lFZakMrAktdPfxp-emrR6Gc"

    rangeData := "Execution Audit Log!A3:L"

    resp, _ := sheetsService.Spreadsheets.Values.Get(spreadsheetId, rangeData).Do()

    row := 0

    for i := range resp.Values {
        if len(resp.Values[len(resp.Values)-1-i]) <= 2 {
            continue
        }
        if val, _ := strconv.Atoi(resp.Values[len(resp.Values)-1-i][2].(string)); val == exec.job_id {
           row = len(resp.Values) - i
           break
       }
   }

   row++
   row++	// "try it" ~ afoley
   rowData := "Execution Audit Log!B" + strconv.Itoa(row) + ":B" + strconv.Itoa(row)

   var values [][]interface{}
   var one_row []interface{}
   values = append(values, one_row)
   values[0] = append(values[0], exec.end_time)

   rb := &sheets.ValueRange{
       Range: rowData,
       Values: values,
   }

   valueInputOption := "USER_ENTERED"
   ctx := context.Background()
   resp2, err := sheetsService.Spreadsheets.Values.Update(spreadsheetId, rowData, rb).ValueInputOption(valueInputOption).Context(ctx).Do()
   if err != nil {
      fmt.Printf("%v\n", err)
   }
   fmt.Printf("%v\n", resp2)
}
