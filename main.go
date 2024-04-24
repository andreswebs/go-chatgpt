package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/tools/serpapi"
)

var mem schema.Memory
var logger *slog.Logger

func init() {
	mem = memory.NewConversationBuffer()
	logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)
}

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
		err = errors.Wrap(err, "program failure")
		logger.Error("%+v", err)
		os.Exit(1)
	}
	defer file.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Interactive Chat with OpenAI (Type 'exit' to quit)")

	for {
		fmt.Print("\nYou: ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "exit" {
			fmt.Println("\nGoodbye!\n")
			break
		}

		_, err := file.WriteString("---\n\n" + userInput + "\n\n")
		if err != nil {
			err = errors.Wrap(err, "program failure")
			fmt.Printf("%+v %s", err, string(debug.Stack()))
			os.Exit(1)
		}

		response, err := query(userInput)
		if err != nil {
			err = errors.Wrap(err, "program failure")
			fmt.Printf("%+v %s", err, string(debug.Stack()))
			os.Exit(1)
		}

		fmt.Printf("\nAI: %s\n", response)

		_, err = file.WriteString("---\n\n" + response + "\n\n")
		if err != nil {
			err = errors.Wrap(err, "program failure")
			fmt.Printf("%+v %s", err, string(debug.Stack()))
			os.Exit(1)
		}
	}

}

func query(input string) (output string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	model, ok := os.LookupEnv("OPENAI_MODEL")
	if !ok {
		model = "gpt-4"
	}
	fmt.Printf("\nmodel: %s\n", model)

	llm, err := openai.NewChat(openai.WithModel(model))
	if err != nil {
		err = errors.Wrap(err, "llm error:")
		return
	}

	serp, err := serpapi.New()
	if err != nil {
		err = errors.Wrap(err, "tool error:")
		return
	}

	calculator := tools.Calculator{}

	a, err := agents.Initialize(
		llm,
		[]tools.Tool{serp, calculator},
		agents.ConversationalReactDescription,
		agents.WithMemory(mem),
	)
	if err != nil {
		err = errors.Wrap(err, "agent initialization error:")
		return
	}

	output, err = chains.Run(context.Background(), a, input)
	if err != nil {
		err = errors.Wrap(err, "chain error:")
	}
	return

}

func handleExitSignal(exitSignal chan os.Signal) {
	<-exitSignal
	fmt.Println("\nReceived termination signal. Shutting down...\n")
	os.Exit(0)
}
