package main

// This is the credential provisioner.
//
// It waits for actions on a pubsub queue, and calls out to scripts to
// create VPN or web certificates.  There are two pubsub queues: the
// request queue is for notifying this code to create certs.  When certs are
// created, the message is sent back down the response queue, so something
// like a web app could monitor the queue to find out when its responses have
// been actioned.

// FIXME: The response queue would benefit from status information, I guess.

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/pubsub/v1"
)

//
// Source: https://socketloop.com/tutorials/golang-validate-email-address-with-regular-expression
//
// Simple regex based email validity check
func validateEmail(email string) bool {
	Re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return Re.MatchString(email)
}

var (
	notifyTopic  = Getenv("PUBSUB_RESPONSE_TOPIC", "credential-response")
	requestTopic = Getenv("PUBSUB_REQUEST_TOPIC", "credential-request")
	subscription = Getenv("PUBSUB_SUBSCRIPTION", "credential-subscription")
)

// Structure for the JSON messages passed on the pub-sub.
type Message struct {

	// Request type
	Type string `json:"type,omitempty"`

	// Email address of user
	User string `json:"user,omitempty"`

	// Credential identity
	Identity string `json:"identity,omitempty"`

	// For probe credentials, a delivery endpoint
	Endpoint string `json:"endpoint,omitempty"`

	// For VPN service, a host to connect to.
	Host string `json:"host,omitempty"`

	// For VPN service, a probe credential string to use for delivery.
	ProbeCred string `json:"probecred,omitempty"`

	// For VPN service, a hostname providing the allocator service
	Allocator string `json:"allocator,omitempty"`
}

type MessageResponse struct {
	Message
	MessageId string `json:"id"`
	Success   bool   `json:"success"`
}

// Sign in to Google cloud pubsub.
func pubsubSignin(key []byte) (*pubsub.Service, error) {

	// Create JWT from key file
	config, err := google.JWTConfigFromJSON(key)
	if err != nil {
		return nil, errors.New("JWTConfigFromJSON: " + err.Error())
	}

	// Access scope
	config.Scopes = []string{pubsub.PubsubScope}

	// Create service client.
	client := config.Client(oauth2.NoContext)

	// Connect to Google Pubsub
	svc, err := pubsub.New(client)
	if err != nil {
		return nil, errors.New("Create client: " + err.Error())
	}

	return svc, nil

}

// Create a topic if it doesn't already exist.
func maybeCreateTopic(svc *pubsub.Service, pr, topic string) error {

	// Topic name.
	name := "projects/" + pr + "/topics/" + topic

	// Get the topic
	_, err := svc.Projects.Topics.Get(name).Do()
	if err == nil {

		// Already exists, then we're done.
		return nil

	}

	// Error... assume it doesn't exist.  If the error was because of
	// something else, that will become apparent shortly.

	// Create the topic.
	_, err = svc.Projects.Topics.Create(
		name,
		&pubsub.Topic{
			Name: name,
		}).Do()
	if err != nil {
		// Create failed.
		fmt.Println("Topic create failed.")
		return err
	}

	fmt.Println("Subscription created.")

	return nil

}

func sendResponse(svc *pubsub.Service, msg *Message, id string, success bool, notifName string) {
	msgResponse := MessageResponse{*msg, id, success}

	bin, err := json.Marshal(msgResponse)
	if err != nil {
		fmt.Println("ERROR Notify failed: could not format json for response "+
			"message, user may hang waiting for response. message: ", msgResponse)
		return
	}

	encoded := base64.StdEncoding.EncodeToString(bin)

	pr := &pubsub.PublishRequest{
		Messages: []*pubsub.PubsubMessage{
			&pubsub.PubsubMessage{
				Data: encoded,
			},
		},
	}

	_, err = svc.Projects.Topics.Publish(
		notifName,
		pr).Do()
	if err != nil {
		fmt.Println("ERROR Notify send failed: " + err.Error())
	}
}

func main() {

	request := Getenv("REQUEST_TOPIC", requestTopic)
	notify := Getenv("NOTIFY_TOPIC", notifyTopic)
	project := Getenv("PUBSUB_PROJECT", "")

	// Get environment variables.
	keyfile := Getenv("KEY", "private.json")

	// Read the key file
	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		fmt.Printf("Couldn't read key file: %s\n",
			err.Error())
		return
	}

	// Sign in to pubsub.
	svc, err := pubsubSignin(key)
	if err != nil {
		fmt.Printf("Couldn't connect: %s\n",
			err.Error())
		return
	}

	fmt.Printf("Connected.\n")

	// Create the request topic.
	err = maybeCreateTopic(svc, project, request)
	if err != nil {
		fmt.Printf("Couldn't create topic %s %s: %s\n",
			project, request,
			err.Error())
		return
	}

	// Create the notify topic.
	err = maybeCreateTopic(svc, project, notify)
	if err != nil {
		fmt.Printf("Couldn't create topic %s %s: %s\n",
			project, request,
			err.Error())
		return
	}

	// Names for stuff for later.
	subsName := "projects/" + project + "/subscriptions/" + subscription
	notifName := "projects/" + project + "/topics/" + notifyTopic

	// Get the subscription.
	_, err = svc.Projects.Subscriptions.Get(subsName).Do()

	// If there's an error, assume it's because the sub doesn't exist so,
	// create it.
	if err != nil {

		// Create subscription object.
		s := &pubsub.Subscription{
			Name: subsName,
			Topic: "projects/" + project + "/topics/" +
				requestTopic,
		}

		// Implement
		_, err = svc.Projects.Subscriptions.Create(subsName, s).
			Do()
		if err != nil {
			fmt.Printf("Couldn't create subscription: %s\n",
				err.Error())
			return
		}
		fmt.Println("Subscription created.")
	}

	// Refresh CRL's at boot
	fmt.Println()
	fmt.Println("---- Create all CRLs at boot")

	cmdOut, err := exec.Command("./create-all-crls").
		Output()

	if err != nil {
		fmt.Println("Error: " + err.Error())
	}

	fmt.Printf("%s", cmdOut)

	fmt.Println()
	fmt.Println("---- Process Messages")

	// Endless loop...
	for {

		// Pull next message.
		resp, err := svc.Projects.Subscriptions.Pull(subsName,
			&pubsub.PullRequest{
				MaxMessages:       1,
				ReturnImmediately: false,
			}).Do()
		if err != nil {
			fmt.Printf("Couldn't pull: %s\n",
				err.Error())
			time.Sleep(time.Second * 10)
			continue
		}

		// Loop through all (1) messages...
		for _, m := range resp.ReceivedMessages {

			// Decode base64.
			var msg Message
			data, _ :=
				base64.StdEncoding.DecodeString(m.Message.Data)

			// Decode JSON.
			err = json.Unmarshal([]byte(data), &msg)
			if err != nil {
				fmt.Println("Couldn't make sense of message: " +
					string(m.Message.Data))
				fmt.Println("Ignored.")
			}

			// VPN case
			if msg.Type == "vpn" {

				if !validateEmail(msg.User) || msg.Identity == "" || len(msg.Identity) == 0 {
					fmt.Println()
					fmt.Println("---- Creating vpn key: parameter validation failed")

					sendResponse(svc, &msg, m.Message.MessageId, false, notifName)

				} else {

					fmt.Println()
					fmt.Println("---- Creating vpn key for " +
						msg.User + msg.Identity)

					out, err := exec.Command("./create-vpn-key",
						msg.User, msg.Identity).
						Output()

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					fmt.Printf("%s", out)
					sendResponse(svc, &msg, m.Message.MessageId, err == nil, notifName)
				}
			} else if msg.Type == "web" {

				// Web cert case.

				if !validateEmail(msg.User) || msg.Identity == "" || len(msg.Identity) == 0 {
					fmt.Println()
					fmt.Println("---- Creating web key: parameter validation failed")

					sendResponse(svc, &msg, m.Message.MessageId, false, notifName)

				} else {

					fmt.Println()
					fmt.Println("---- Creating web key for " +
						msg.User + msg.Identity)

					out, err := exec.Command("./create-web-key",
						msg.User, msg.Identity).
						Output()

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					fmt.Printf("%s", out)

					sendResponse(svc, &msg, m.Message.MessageId, err == nil, notifName)
				}
			} else if msg.Type == "probe" {

				// Probe cert case.

				if !validateEmail(msg.User) || msg.Identity == "" || len(msg.Identity) == 0 || msg.Endpoint == "" {
					fmt.Println()
					fmt.Println("---- Creating probe key: parameter validation failed")

					sendResponse(svc, &msg, m.Message.MessageId, false, notifName)

				} else {

					fmt.Println()
					fmt.Println("---- Creating probe key for " +
						msg.User + msg.Identity)

					out, err := exec.Command("./create-probe-key",
						msg.User, msg.Identity,
						msg.Endpoint).
						Output()

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					fmt.Printf("%s", out)

					sendResponse(svc, &msg, m.Message.MessageId, err == nil, notifName)
				}
			} else if msg.Type == "vpn-service" {

				// VPN service cert case.

				if !validateEmail(msg.User) || msg.Identity == "" || len(msg.Identity) == 0 || msg.Host == "" || msg.Allocator == "" || msg.ProbeCred == "" {
					fmt.Println()
					fmt.Println("---- Creating probe key: parameter validation failed")

					sendResponse(svc, &msg, m.Message.MessageId, false, notifName)

				} else {

					fmt.Println()
					fmt.Println("---- Creating VPN service key for " +
						msg.User + msg.Identity)

					out, err := exec.Command("./create-vpn-service-key",
						msg.User, msg.Identity,
						msg.Host, msg.Allocator,
						msg.ProbeCred).
						Output()

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					fmt.Printf("%s", out)

					sendResponse(svc, &msg, m.Message.MessageId, err == nil, notifName)
				}
			} else if msg.Type == "revoke-vpn" {

				if !validateEmail(msg.User) {
					fmt.Println()
					fmt.Println("---- Revoking vpn key: parameter validation failed")

					sendResponse(svc, &msg, m.Message.MessageId, false, notifName)

				} else {

					fmt.Println()
					fmt.Println("---- Revoking VPN key for " +
						msg.User + msg.Identity)

					out, err := exec.Command("./revoke-vpn-key",
						msg.User, msg.Identity).
						Output()

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					fmt.Printf("%s", out)

					sendResponse(svc, &msg, m.Message.MessageId, err == nil, notifName)

				}
			} else if msg.Type == "revoke-web" {

				// Revoke Web cert case.

				if !validateEmail(msg.User) {
					fmt.Println()
					fmt.Println("---- Revoking vpn key: parameter validation failed")

					sendResponse(svc, &msg, m.Message.MessageId, false, notifName)

				} else {
					fmt.Println()
					fmt.Println("---- Revoking web key for " +
						msg.User)

					out, err := exec.Command("./revoke-web-key",
						msg.User).
						Output()

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					fmt.Printf("%s", out)

					sendResponse(svc, &msg, m.Message.MessageId, err == nil, notifName)
				}
			} else if msg.Type == "revoke-probe" {

				// Revoke probe cert case.

				if !validateEmail(msg.User) {
					fmt.Println()
					fmt.Println("---- Revoking probe key: parameter validation failed")

					sendResponse(svc, &msg, m.Message.MessageId, false, notifName)

				} else {
					fmt.Println()
					fmt.Println("---- Revoking probe key for " +
						msg.User)

					out, err := exec.Command("./revoke-probe-key",
						msg.User).
						Output()

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					fmt.Printf("%s", out)

					sendResponse(svc, &msg, m.Message.MessageId, err == nil, notifName)
				}
			} else if msg.Type == "revoke-vpn-service" {

				// Revoke probe cert case.

				if !validateEmail(msg.User) {
					fmt.Println()
					fmt.Println("---- Revoking VPN service key: parameter validation failed")

					sendResponse(svc, &msg, m.Message.MessageId, false, notifName)

				} else {
					fmt.Println()
					fmt.Println("---- Revoking VPN service key for " +
						msg.User)

					out, err := exec.Command("./revoke-vpn-service-key",
						msg.User).
						Output()

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					fmt.Printf("%s", out)

					sendResponse(svc, &msg, m.Message.MessageId, err == nil, notifName)
				}
			} else if msg.Type == "revoke-all" {

				if !validateEmail(msg.User) {
					fmt.Println()
					fmt.Println("---- Revoking all key: parameter validation failed")

					sendResponse(svc, &msg, m.Message.MessageId, false, notifName)

				} else {

					// Revoke All cert case.

					fmt.Println()
					fmt.Println("---- Revoking all vpn and web key for " +
						msg.User)

					out, err := exec.Command("./revoke-all-key",
						msg.User).
						Output()

					if err != nil {
						fmt.Println("Error: " + err.Error())
					}

					fmt.Printf("%s", out)

					sendResponse(svc, &msg, m.Message.MessageId, err == nil, notifName)
				}
			} else if msg.Type == "create-crls" {

				// Create all CRLs

				fmt.Println()
				fmt.Println("---- Creating all CRLs requested")

				out, err := exec.Command("./create-all-crls").
					Output()

				if err != nil {
					fmt.Println("Error: " + err.Error())
				}

				fmt.Printf("%s", out)

				sendResponse(svc, &msg, m.Message.MessageId, err == nil, notifName)

			} else if msg.Type == "" {

				fmt.Printf("Request type (empty) - Ignored \n")
				sendResponse(svc, &msg, m.Message.MessageId, false, notifName)

			} else {

				fmt.Printf("Request for unknown type (%s)?\n",
					msg.Type)
				fmt.Println("Ignored.")
			}

			// Acknowledge the message
			_, err = svc.Projects.Subscriptions.Acknowledge(subsName,
				&pubsub.AcknowledgeRequest{
					AckIds: []string{
						m.AckId,
					},
				}).Do()
			if err != nil {
				fmt.Println("Ack: " + err.Error())
			}
		}
	}

}
