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
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	ctx := context.Background()
	b, err := ioutil.ReadFile("secrets/credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
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

	var deletionMessageIds []string

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

	for _, l := range keepLastLabels {

		instances, err := strconv.Atoi(strings.TrimPrefix(l.Name, "ManagedLabels/KeepLast"))

		resp, err := srv.Users.Messages.List(user).LabelIds(l.Id).Do()

		if err != nil {
			log.Fatalf("Error listing labels: %v", err)
		}

		fmt.Printf("Label:%v", l)

		messageHeaders := GetMessageHeaders(srv, resp.Messages)

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
				fmt.Printf("%v", mh)
			}

		}

		fmt.Printf("%v", keepLastMap)
	}

	for _, l := range keepDaysLabels {

		days := strings.TrimPrefix(l.Name,"ManagedLabels/KeepDays")
		fmt.Printf("older_than:%sd", days)
		resp, err := srv.Users.Messages.List(user).LabelIds(l.Id).Q(fmt.Sprintf("older_than:%sd", days)).Do()
		if err != nil {
			log.Fatalf("Error listing labels: %v", err)
		}

		messageHeaders := GetMessageHeaders(srv, resp.Messages)
		for _, mh := range messageHeaders {
			fmt.Printf("- %v\r\n", mh )
		}
	}

}

func GetMessageHeaders(srv *gmail.Service, messages []*gmail.Message) []*MessageHeader {
	var result []*MessageHeader

	for _,message := range messages {
		messageHeader, err := GetMessageHeader(srv, message.Id)

		if err != nil {
			// error getting message
			fmt.Printf("Error getting message %v", message.Id)
		} else {
			result = append(result, messageHeader)
		}
	}

	return result
}

func GetMessageHeader(srv *gmail.Service, messageId string) (*MessageHeader,error) {
	resp, err := srv.Users.Messages.Get("me", messageId).Do()

	t := time.Unix(resp.InternalDate/1000, 0)

	mh := MessageHeader{
		Subject: GetHeaderValue(resp.Payload.Headers, "Subject"), // Subject: Inflation: Persistently Transitory | Investing Insights Weekly
		From: GetHeaderValue(resp.Payload.Headers, "From"),
		Id: messageId,
		Time: t,
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
	Id string
	Subject string
	From string
	Time time.Time
	MessageId string
}

