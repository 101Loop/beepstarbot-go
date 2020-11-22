package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// APIEndpoint is the endpoint for all API methods,
	// with formatting for Sprintf.
	APIEndpoint = "https://api.telegram.org/bot%s/%s"
)

// APIResponse is response from the Telegram API with the result
// stored raw.
type APIResponse struct {
	Ok          bool                `json:"ok"`
	Result      json.RawMessage     `json:"result"`
	ErrorCode   int                 `json:"error_code"`
	Description string              `json:"description"`
	Parameters  *ResponseParameters `json:"parameters"`
}

// ResponseParameters are various errors that can be returned in APIResponse.
type ResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id"` // optional
	RetryAfter      int   `json:"retry_after"`        // optional
}

// Error is an error containing extra information returned by the Telegram API.
type Error struct {
	Code    int
	Message string
	ResponseParameters
}

func (e Error) Error() string {
	return e.Message
}

// webhookReqBody struct that mimics the webhook response body from telegram
// https://core.telegram.org/bots/api#update
type webhookReqBody struct {
	Message struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
		} `json:"from"`
	} `json:"message"`
}

// decodeAPIResponse just decode http.Response.Body stream to APIResponse struct
// for efficient memory usage
func decodeAPIResponse(respBody io.Reader, res *APIResponse) (_ []byte, err error) {
	dec := json.NewDecoder(respBody)
	err = dec.Decode(res)
	return
}

// This Handler is called everytime telegram sends us a webhook event
func Handler(res http.ResponseWriter, req *http.Request) {
	// Decode the JSON response body
	body := &webhookReqBody{}
	if err := json.NewDecoder(req.Body).Decode(body); err != nil {
		sentry.CaptureException(err)
		fmt.Println("Could not decode request body", err)
		return
	}

	// Check if the message contains the word `aww`
	// if not, return without doing anything
	if !strings.Contains(strings.ToLower(body.Message.Text), "aww") {
		return
	}

	// if text contains `aww`, call kickUser function
	if _, err := kickChatMember(body.Message.Chat.ID, body.Message.From.ID); err != nil {
		// handle when message sent by owner of group
		if fmt.Sprint(err) == "Bad Request: can't remove chat owner" {
			if _, err := sendMessage(body.Message.Chat.ID, "Group Owner's can use forbidden words!"); err != nil {
				sentry.CaptureException(err)
				return
			}
		// handle when bot doesn't have enough permission to kick users
		} else if fmt.Sprint(err) == "Bad Request: not enough rights to restrict/unrestrict chat member" {
			text := "Forbidden Word used but I don't have enough permissions to kick members. Please make me an admin."
			if _, err := sendMessage(body.Message.Chat.ID, text); err != nil {
				sentry.CaptureException(err)
				return
			}
		// handle when message sent by group admin
		} else if fmt.Sprint(err) == "Bad Request: user is an administrator of the chat" {
			text := "Chat Admins can also use forbidden words!"
			if _, err := sendMessage(body.Message.Chat.ID, text); err != nil {
				sentry.CaptureException(err)
				return
			}
		// handle when message sent in private chat
		} else if fmt.Sprint(err) == "Bad Request: chat member status can't be changed in private chats" {
			text := "Sorry, This doesn't work in private chats!"
			if _, err := sendMessage(body.Message.Chat.ID, text); err != nil {
				sentry.CaptureException(err)
				return
			}
		// for any other errors, log errors to sentry
		} else {
			sentry.CaptureException(err)
			return
		}
	// then call sendMessage to send message in group
	} else {
		firstName := body.Message.From.FirstName
		text := fmt.Sprintf("%s have used a forbidden word and will be banned for a day from this group.", firstName)

		_, err := sendMessage(body.Message.Chat.ID, text)
		if err != nil {
			sentry.CaptureException(err)
			return
		}
	}
}

// Below code deals with kicking user & sending message to the group

// kickChatMemberReqBody struct to confirm the JSON body
// of the request
// https://core.telegram.org/bots/api#kickchatmember
type kickChatMemberReqBody struct {
	ChatID    int64 `json:"chat_id"`
	UserID    int64 `json:"user_id"`
	UntilDate int64 `json:"until_date"`
}

// sendMessageReqBody struct to confirm the JSON body
// of the request
// https://core.telegram.org/bots/api#sendmessage
type sendMessageReqBody struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

// kickChatMember takes chatID and userID to kick members from group
func kickChatMember(chatID int64, userID int64) (APIResponse, error) {
	t := time.Now()
	untilDate := t.AddDate(0, 0, 1).Unix()

	// Create request body struct
	reqBody := &kickChatMemberReqBody{
		ChatID:    chatID,
		UserID:    userID,
		UntilDate: untilDate,
	}

	// Create JSON body from the struct
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return APIResponse{}, err
	}

	endpoint := fmt.Sprintf(APIEndpoint, os.Getenv("BOT_TOKEN"), "kickChatMember")

	return handleAPIResponse(err, endpoint, reqJSON)
}

// sendMessage sends message to the telegram group when user is kicked
func sendMessage(chatID int64, text string) (APIResponse, error) {
	// Create request body struct
	reqBody := &sendMessageReqBody{
		ChatID: chatID,
		Text:   text,
	}

	// Create JSON body from the struct
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return APIResponse{}, err
	}

	endpoint := fmt.Sprintf(APIEndpoint, os.Getenv("BOT_TOKEN"), "sendMessage")

	return handleAPIResponse(err, endpoint, reqJSON)
}

// handleAPIResponse handles API call to telegram
func handleAPIResponse(err error, endpoint string, reqJSON []byte) (APIResponse, error) {
	// Send Post Request to Telegram API
	res, err := http.Post(endpoint, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return APIResponse{}, err
	}
	// defer will close the body at the end
	defer res.Body.Close()

	var apiResp APIResponse
	_, err = decodeAPIResponse(res.Body, &apiResp)
	if err != nil {
		return apiResp, err
	}

	if !apiResp.Ok {
		parameters := ResponseParameters{}
		if apiResp.Parameters != nil {
			parameters = *apiResp.Parameters
		}
		return apiResp, Error{Code: apiResp.ErrorCode, Message: apiResp.Description, ResponseParameters: parameters}
	}
	return apiResp, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		sentry.CaptureException(err)
	}

	err = sentry.Init(sentry.ClientOptions{
		Dsn: os.Getenv("SENTRY_DSN"),
	})

	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentry.Flush(2 * time.Second)

	server := &http.Server{
		Addr:    ":3000",
		Handler: http.HandlerFunc(Handler),
	}
	log.Println("Bot is up and running...")
	server.ListenAndServe()
}
