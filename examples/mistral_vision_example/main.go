package main

import (
	"fmt"
	"log"
	"os"

	"github.com/agentizen/agent-sdk-go/pkg/agent"
	"github.com/agentizen/agent-sdk-go/pkg/model"
	"github.com/agentizen/agent-sdk-go/pkg/model/providers/mistral"
	"github.com/agentizen/agent-sdk-go/pkg/runner"
)

func loadImage(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("reading image: %w", err)
	}
	// Determine MIME type from extension.
	mimeType := "image/jpeg"
	if len(path) > 4 {
		switch path[len(path)-4:] {
		case ".png":
			mimeType = "image/png"
		case "webp":
			mimeType = "image/webp"
		case ".gif":
			mimeType = "image/gif"
		}
	}
	return data, mimeType, nil
}

func main() {
	apiKey := os.Getenv("MISTRAL_API_KEY")
	if apiKey == "" {
		log.Fatal("MISTRAL_API_KEY environment variable not set")
	}

	// Determine image path: use first CLI argument or fall back to a default.
	imagePath := "image.jpg"
	if len(os.Args) > 1 {
		imagePath = os.Args[1]
	}

	imageData, mimeType, err := loadImage(imagePath)
	if err != nil {
		log.Fatalf("Failed to load image %q: %v", imagePath, err)
	}
	fmt.Printf("Loaded image: %s (%s, %d bytes)\n", imagePath, mimeType, len(imageData))

	provider := mistral.NewProvider(apiKey)

	// mistral-small-2603 supports vision.
	assistant := agent.NewAgent("Mistral Vision Assistant")
	assistant.SetModelProvider(provider)
	assistant.WithModel("mistral-small-2603")
	assistant.SetSystemInstructions("You are a helpful visual assistant. Describe images in detail.")

	r := runner.NewRunner()
	r.WithDefaultProvider(provider)

	fmt.Println("\nSending image to Mistral vision model...")
	result, err := r.RunSync(assistant, &runner.RunOptions{
		Input: "What is in this image? Describe it in detail.",
		InputParts: []model.ContentPart{
			{
				Type:     model.ContentPartTypeImage,
				MimeType: mimeType,
				Data:     imageData,
				Name:     imagePath,
			},
		},
		MaxTurns: 1,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nAgent response:")
	fmt.Println(result.FinalOutput)

	if len(result.RawResponses) > 0 {
		last := result.RawResponses[len(result.RawResponses)-1]
		if last.Usage != nil {
			fmt.Printf("\nToken usage: %d total tokens\n", last.Usage.TotalTokens)
		}
	}
}
