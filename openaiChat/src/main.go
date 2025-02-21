package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go"
)

type ChatMessage struct {
	role    string
	content string
}

var OPENAI_CLIENT *openai.Client = nil

func getOpenaiClient() *openai.Client {
	if OPENAI_CLIENT == nil {
		OPENAI_CLIENT = openai.NewClient()
	}
	return OPENAI_CLIENT
}

func openaiChatCompletion(messages []*ChatMessage, model string) (string, error) {
	openaiClient := getOpenaiClient()
	var openaiChatMessages []openai.ChatCompletionMessageParamUnion

	for _, message := range messages {
		if message.role == "system" {
			openaiChatMessages = append(openaiChatMessages, openai.AssistantMessage(message.content))
			continue
		}

		if message.role == "user" {
			openaiChatMessages = append(openaiChatMessages, openai.UserMessage(message.content))
			continue
		}

		fmt.Fprintf(os.Stderr, "Invalid message role '%s'. Skipped\n", message.role)
	}

	chatCompletion, err := openaiClient.Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Messages: openai.F(openaiChatMessages),
			Model:    openai.F(model),
		},
	)

	if err != nil {
		return "", err
	}

	return chatCompletion.Choices[0].Message.Content, nil
}

func openaiChat(question string, model string) {
	inputBuffer := bufio.NewReader(os.Stdin)
	messages := []*ChatMessage{{role: "system", content: "You are a useful general purpose assisstant"}}

	fmt.Printf("user >> %s\n", question)

	for {
		userMessage := ChatMessage{role: "user", content: question}
		messages = append(messages, &userMessage)

		response, err := openaiChatCompletion(messages, model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed openai interaction. Error: %s\n", err)
			panic(err)
		}

		fmt.Printf("assistant >> %s\n", response)

		systemMessage := ChatMessage{role: "system", content: response}
		messages = append(messages, &systemMessage)

		fmt.Printf("user >> ")
		question, err = inputBuffer.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read user input")
			break
		}

		question = strings.Trim(question, "\n ")
		if question == "<exit>" {
			fmt.Println("Requested exit")
			break
		}
	}

}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [question]\n", os.Args[0])
		return
	}
	openaiChat(os.Args[1], openai.ChatModelGPT4oMini)
}
