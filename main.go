package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
)

func main() {
	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	go handleExitSignal(exitSignal)

	fileName := os.Getenv("CHAT_FILE")
	if fileName == "" {
		fileName = "chat.txt"
	}

	file, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Interactive Chat with OpenAI (Type 'exit' to quit)")

	for {
		fmt.Print("\nYou: ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		_, err := file.WriteString("---\n\n" + userInput + "\n\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v\n", err)
			return
		}

		response, err := query(userInput)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		fmt.Printf("\nAI: %s\n", response)

		_, err = file.WriteString("---\n\n" + response + "\n\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v\n", err)
			return
		}
	}

}

func query(input string) (string, error) {
	prompt := struct {
		Input string `json:"input"`
	}{
		input,
	}

	model, ok := os.LookupEnv("OPENAI_MODEL")
	if !ok {
		model = "gpt-3.5-turbo"
	}

	llm, err := openai.NewChat(openai.WithModel(model))
	if err != nil {
		return "", err
	}

	chatmsgs := []schema.ChatMessage{
		schema.HumanChatMessage{Content: prompt.Input},
	}

	aimsg, err := llm.Call(context.Background(), chatmsgs)
	if err != nil {
		return "", err
	}

	response := aimsg.GetContent()
	return response, nil

}


func handleExitSignal(exitSignal chan os.Signal) {
	<-exitSignal
	fmt.Println("\nReceived termination signal. Shutting down...")
	os.Exit(0)
}
