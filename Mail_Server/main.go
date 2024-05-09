package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

var srv *gmail.Service

var toEmail = "thucuyen.truong@qlay.ai"

// var toEmail = "dangduchieudn99@gmail.com"

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
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

	authCode := "4/0AdLIrYd14rh_aR_V3OETfwYWLCS29kPs6j-gKFY4rJmbBpefsdkcqfIdUQ3Ef3fvQPFnEQ"
	// if _, err := fmt.Scan(&authCode); err != nil {
	// 	log.Fatalf("Unable to read authorization code: %v", err)
	// }

	tok, err := config.Exchange(context.Background(), authCode)
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
	var err error
	srv, err = initService()
	if err != nil {
		log.Fatal(err)
	}
	go receiveFromQueue()

	var forever chan struct{}
	<-forever
}

func initService() (*gmail.Service, error) {
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
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
	return srv, err
}

func send(userId, msg string) error {
	message, err := srv.Users.Messages.Send("me", &gmail.Message{
		Raw: base64.URLEncoding.EncodeToString(buildBody("dangduchieudn1999@gmail.com",
			userId, "Weather forecast", msg)),
	}).Do()
	fmt.Println(message)
	return err
}

func buildBody(from, to, subject, msg string) []byte {
	return []byte(
		"From: " + from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n\r\n" +
			msg)
}

// loop inf
func receiveFromQueue() error {
	print(os.Getenv("BROKER_USER"))
	conn, err := amqp091.Dial(
		"amqp://" +
			os.Getenv("BROKER_USER") +
			":" + os.Getenv("BROKER_PASSWORD") +
			"@" + os.Getenv("BROKER_HOST") +
			":" + os.Getenv("BROKER_PORT") + "/")
	if err != nil {
		log.Fatal(err)
	}

	ch, err := conn.Channel()

	if err != nil {
		log.Fatal(err)
	}

	queue, err := ch.QueueDeclare("MAIL", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	deliveries, err := ch.Consume(
		queue.Name, // name
		"",         // consumerTag,
		false,      // autoAck
		false,      // exclusive
		false,      // noLocal
		false,      // noWait
		nil,        // arguments
	)

	ch.QueueBind(queue.Name, "*", "amq."+amqp091.ExchangeDirect, false, nil)
	if err != nil {
		log.Fatal(err)
	}
	for deliveriy := range deliveries {
		body := deliveriy.Body
		err := send(toEmail, string(body))
		if err == nil {
			deliveriy.Ack(false)
		} else {
			retry := 0
			go func(deli amqp091.Delivery) {
				for {
					fmt.Print(err)
					time.Sleep(5 * time.Minute)
					retry++
					err := send(toEmail, string(body))
					if err == nil {
						return
					}
					log.Print(err)
					if retry >= 3 {
						deli.Ack(false)
						return
					}
				}
			}(deliveriy)
		}
	}

	return nil
}
