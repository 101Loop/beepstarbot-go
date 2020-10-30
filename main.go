package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Telegram Constants
const (
	// APIEndpoint is the endpoint for all API methods,
	// with formatting for Sprintf.
	APIEndpoint = "https://api.telegram.org/bot%s/%s"
)

// APIResponse is a response from the Telegram API with the result
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

// webhookReqBody struct that mimics the webhook response body
// https://core.telegram.org/bots/api#update
type webhookReqBody struct {
	Message struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From struct {
			ID int64 `json:"id"`
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
		fmt.Println("Could not decode request body", err)
		return
	}

	// Check if the message contains the word `aww`
	// if not, return without doing anything
	if !strings.Contains(strings.ToLower(body.Message.Text), "aww") {
		return
	}

	// if the text contains `aww`, call kickUser function
	if _, err := kickUser(body.Message.Chat.ID, body.Message.From.ID); err != nil {
		fmt.Println("Error occurred", err)
		return
	}

	// log confirmation message
	fmt.Println("User Kicked!")
}

// Below code deals with kicking user & sending message to the group

// kickUserReqBody struct to confirm the JSON body
// of the send request
// https://core.telegram.org/bots/api#sendmessage
type kickUserReqBody struct {
	ChatID    int64 `json:"chat_id"`
	UserID    int64 `json:"user_id"`
	UntilDate int64 `json:"until_date"`
}

// kickUser takes a chatID and sends message to them
func kickUser(chatID int64, userID int64) (APIResponse, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	t := time.Now()
	untilDate := t.AddDate(0, 0, 1).Unix()

	// Create request body struct
	reqBody := &kickUserReqBody{
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
	server := &http.Server{
		Addr:    ":3000",
		Handler: http.HandlerFunc(Handler),
	}
	log.Println("Bot is up and running...")
	server.ListenAndServe()
}
