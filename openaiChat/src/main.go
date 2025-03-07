package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go"
)

/*
-----------------
<<< constants >>>
-----------------
*/

// Path to json with message history
const HISTORY_PATH = "./history.json"

/*
-------------------------
 <<< type definitions >>>
-------------------------
*/

// Simple message structure
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Simple conversation structure for saving history to json
type ConversationHistory struct {
	TimeStamp string         `json:"timeStamp"`
	Messages  []*ChatMessage `json:"messages"`
}

/*
-------------------------------
<<< Define global vars >>>
-------------------------------
*/

var openaiClient *openai.Client = nil
var historyMessages = []*ChatMessage{}                                // Track history
var conversationMessages = []openai.ChatCompletionMessageParamUnion{} // Track openai messages

/*
---------------------
<<< Aux functions >>>
---------------------
*/

// Save current message history to a Json at HISOTRY_PATH.
// Returns an error which is nil on success
func saveHistoryToJson() error {
	history := ConversationHistory{
		TimeStamp: time.Now().Format("2006-01-02T15:04:05"),
		Messages:  historyMessages,
	}

	jsonBytes, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	jsonFile, err := os.Create(HISTORY_PATH)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	_, err = jsonFile.Write(jsonBytes)
	if err != nil {
		return err
	}

	return nil
}

// Load message history from HISTORY_PATH
// Returns an error which is nil on success
func loadHistoryJson() error {
	jsonFile, err := os.Open(HISTORY_PATH)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	history := ConversationHistory{}
	json.NewDecoder(jsonFile).Decode(&history)
	fmt.Printf("Loading conversation from: %s\n", history.TimeStamp)

	historyMessages = history.Messages
	for _, message := range historyMessages {
		addConversationMessage(message)
	}

	return nil
}

// Initialize message history and openai messages with a simple system message
func initConversation() {
	fmt.Println("Initializing new conversation")
	content := "You are a useful assistant"
	historyMessages = []*ChatMessage{{Role: "system", Content: content}}
	conversationMessages = []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(content)}
}

// Load the conversation if any.
// If it fails to load or the loaded conversation doesn't have messages, initConversation is called.
// If restart is true, initConversation is forcefully called.
func loadConversation(restart bool) {
	if restart {
		fmt.Println("Forcefully started new conversation")
		initConversation()
	} else if err := loadHistoryJson(); err != nil || len(historyMessages) == 0 {
		fmt.Fprintln(os.Stderr, "Failed to load history")
		initConversation()
	}
}

// Add a message to tracked openai messages based on its role
func addConversationMessage(newMessage *ChatMessage) {
	switch newMessage.Role {
	case "system":
		conversationMessages = append(conversationMessages, openai.SystemMessage(newMessage.Content))
	case "assistant":
		conversationMessages = append(conversationMessages, openai.AssistantMessage(newMessage.Content))
	case "user":
		conversationMessages = append(conversationMessages, openai.UserMessage(newMessage.Content))
	default:
		fmt.Fprintf(os.Stderr, "Invalid message role: %s\n", newMessage.Role)
	}
}

// Update both the history messages and the openai messages tracked
func updateHistoryAndConversation(newMessage *ChatMessage) {
	historyMessages = append(historyMessages, newMessage)
	addConversationMessage(newMessage)
}

/*
-----------------------------
<<< main openai functions >>>
-----------------------------
*/

// Get globally defined openaiClient, initialize it if nil
func getOpenaiClient() *openai.Client {
	if openaiClient == nil {
		fmt.Println("Createing new OpenAI client")
		openaiClient = openai.NewClient()
	}
	return openaiClient
}

// Main openai chat completion, provide messages and a model, return a response
func openaiChatCompletion(messages []openai.ChatCompletionMessageParamUnion, model string) string {
	openaiClient = getOpenaiClient()

	chatCompletion, err := openaiClient.Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Messages: openai.F(messages),
			Model:    openai.F(model),
		},
	)

	if err != nil {
		panic(err)
	}

	return chatCompletion.Choices[0].Message.Content
}

// Openai chat loop. Starts chatcompletion with `question`, then ask user input on loop.
// Break the loop if user input is <exit>
func openaiChat(question string, model string) {
	inputBuffer := bufio.NewReader(os.Stdin)
	var err error

	fmt.Printf("user >> %s\n", question)
	for {
		userMessage := ChatMessage{Role: "user", Content: question}
		updateHistoryAndConversation(&userMessage)

		response := openaiChatCompletion(conversationMessages, model)
		fmt.Printf("assistant >> %s\n", response)

		assistantMessage := ChatMessage{Role: "assistant", Content: response}
		updateHistoryAndConversation(&assistantMessage)

		fmt.Printf("user >> ")
		question, err = inputBuffer.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "There was an issue parsing user input. Error: %s\n", err)
			break
		}

		question = strings.Trim(question, "\n ")
		if question == "<exit>" {
			fmt.Println("User requested exit")
			break
		}

	}
}

/*
----------
Entrypoint
----------
*/
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [question] [optional-restart-conversation(bool)]\n", os.Args[0])
		os.Exit(1)
	}

	restartConversation := false
	if len(os.Args) > 2 {
		restartConversation = strings.Contains(strings.ToLower(os.Args[2]), "true")
	}

	loadConversation(restartConversation)
	openaiChat(os.Args[1], openai.ChatModelGPT4oMini)
	saveHistoryToJson()
}
