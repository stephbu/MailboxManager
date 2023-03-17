package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "secrets/token.json"
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

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
        err := f.Close()
        if err != nil {

        }
    }(f)
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) bool {

	var err error

	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer func(f *os.File) {
        err := f.Close()
        if err != nil {
        }
    }(f)
	err = json.NewEncoder(f).Encode(token)
    if err != nil {
        return false
    }
	return true
}

func main() {
	ctx := context.Background()
	b, err := ioutil.ReadFile("secrets/credentials.json")
	if err != nil {
		// secret file from google needed in ./secrets/credentials.json
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.MailGoogleComScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	user := "me"
	r, err := srv.Users.Labels.List(user).Do()
	if err != nil {
		log.Fatalf("Unable to list user labels: %v", err)
	}

	fmt.Println("Labels:")
	var keepLastLabels []*gmail.Label
	var keepDaysLabels []*gmail.Label

	// Messages to be deleted
	var deletionMessageIds []string

	// Extract Labels
	for _, l := range r.Labels {
		if strings.HasPrefix(l.Name, "ManagedLabels/Keep") {
			fmt.Printf("- %s\n", l.Name)
			if strings.HasPrefix(l.Name, "ManagedLabels/KeepLast") {
				keepLastLabels = append(keepLastLabels, l)
			}
			if strings.HasPrefix(l.Name, "ManagedLabels/KeepDays") {
				keepDaysLabels = append(keepDaysLabels, l)
			}
		}
	}

	for _, label := range keepLastLabels {

		var messageHeaders []*MessageHeader
		fmt.Printf("Label:%v\r\n", label.Name)

		instances, _ := strconv.Atoi(strings.TrimPrefix(label.Name, "ManagedLabels/KeepLast"))
		messageHeaders = GetLabelMessages(srv, user, label)

		// Sort Descending
		sort.SliceStable(messageHeaders, func(i, j int) bool {
			return messageHeaders[i].Time.After(messageHeaders[j].Time)
		})

		keepLastMap := map[string]int{}
		for _, mh := range messageHeaders {
			var ok bool
			var val int
			if val, ok = keepLastMap[mh.From]; ok {
				val = val + 1
			} else {
				val = 1
			}

			keepLastMap[mh.From] = val
			if val > instances {
				deletionMessageIds = append(deletionMessageIds, mh.Id)
				fmt.Printf("- %v\r\n", mh)
			}

		}
	}

	for _, label := range keepDaysLabels {

		fmt.Printf("Label:%v\r\n", label.Name)

		days, _ := strconv.Atoi(strings.TrimPrefix(label.Name, "ManagedLabels/KeepDays"))
		cutoffTime := time.Now().Add(time.Duration(-days) * time.Hour * 24)

		var messageHeaders []*MessageHeader

		fmt.Printf("- older_than:%v\r\n", cutoffTime)
		messageHeaders = GetLabelMessages(srv, user, label)

		for _, mh := range messageHeaders {

			if mh.Time.Before(cutoffTime) {
				fmt.Printf("- %v\r\n", mh)
				deletionMessageIds = append(deletionMessageIds, mh.Id)
			}
		}
	}
	fmt.Printf("%v\r\n", deletionMessageIds)

	for _, messageId := range deletionMessageIds {
		fmt.Printf("- deleting %v\r\n", messageId)
		err = srv.Users.Messages.Delete(user, messageId).Do()
		if err != nil {
			fmt.Println(err)
		}
	}

}

func GetLabelMessages(srv *gmail.Service, user string, l *gmail.Label) []*MessageHeader {

	var messageHeaders []*MessageHeader

	resp, err := srv.Users.Messages.List(user).LabelIds(l.Id).Do()
	if err != nil {
		log.Fatalf("Error listing labels: %v", err)
	}
	messageHeaders = append(messageHeaders, GetMessageHeaders(srv, user, resp.Messages)...)
	for resp.NextPageToken != "" {
		resp, err = srv.Users.Messages.List(user).LabelIds(l.Id).PageToken(resp.NextPageToken).Do()
		messageHeaders = append(messageHeaders, GetMessageHeaders(srv, user, resp.Messages)...)
		if err != nil {
			log.Fatalf("Error listing labels: %v", err)
		}
	}

	fmt.Printf("- %v messages\r\n", len(messageHeaders))

	return messageHeaders
}

func GetMessageHeaders(srv *gmail.Service, user string, messages []*gmail.Message) []*MessageHeader {
	var result []*MessageHeader

	for _, message := range messages {
		messageHeader, err := GetMessageHeader(srv, user, message.Id)

		if err != nil {
			// error getting message
			fmt.Printf("Error getting message %v", message.Id)
		} else {
			result = append(result, messageHeader)
		}
	}

	return result
}

func GetMessageHeader(srv *gmail.Service, user string, messageId string) (*MessageHeader, error) {
	resp, err := srv.Users.Messages.Get(user, messageId).Do()

	t := time.Unix(resp.InternalDate/1000, 0)

	const headerSubject = "Subject"
	const headerFrom = "From"
	mh := MessageHeader{
		Subject: GetHeaderValue(resp.Payload.Headers, headerSubject), // Subject: Inflation: Persistently Transitory | Investing Insights Weekly
		From:    GetHeaderValue(resp.Payload.Headers, headerFrom),
		Id:      messageId,
		Time:    t,
	}
	return &mh, err
}

func GetHeaderValue(headers []*gmail.MessagePartHeader, name string) string {
	for _, header := range headers {
		if header.Name == name {
			return header.Value
		}
	}
	return ""
}

type MessageHeader struct {
	Id        string
	Subject   string
	From      string
	Time      time.Time
	MessageId string
}

// converting the struct to String format.
func (mh MessageHeader) String() string {
	return fmt.Sprintf(mh.Id, mh.Time)
}
