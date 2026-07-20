package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func main() {
	// argument prompt
	var prompt string
	flag.StringVar(&prompt, "p", "", "Prompt to send to LLM")
	flag.Parse()
	if prompt == "" {
		panic("Prompt must not be empty")
	}

	// init
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	baseUrl := os.Getenv("OPENROUTER_BASE_URL")
	if baseUrl == "" {
		baseUrl = "https://openrouter.ai/api/v1"
	}
	if apiKey == "" {
		panic("Env variable OPENROUTER_API_KEY not found")
	}
	client := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseUrl))

	message := []openai.ChatCompletionMessageParamUnion{
		{
			OfUser: &openai.ChatCompletionUserMessageParam{
				Content: openai.ChatCompletionUserMessageParamContentUnion{
					OfString: openai.String(prompt),
				},
			},
		},
	}

	for {
		resp, err := client.Chat.Completions.New(context.Background(),
			openai.ChatCompletionNewParams{
				Model:    "anthropic/claude-haiku-4.5",
				Messages: message,
				Tools: []openai.ChatCompletionToolUnionParam{
					openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
						Name:        "Read",
						Description: openai.String("Read and return the contents of a file"),
						Parameters: openai.FunctionParameters{
							"type": "object",
							"properties": map[string]any{
								"file_path": map[string]any{
									"type":        "string",
									"description": "The path to the file to read",
								},
							},
							"required": []string{"file_path"},
						},
					}),
					openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
						Name:        "Write",
						Description: openai.String("Write content to a file"),
						Parameters: openai.FunctionParameters{
							"type": "object",
							"properties": map[string]any{
								"file_path": map[string]any{
									"type":        "string",
									"description": "The path of the file to write to",
								},
								"content": map[string]any{
									"type":        "string",
									"description": "The content to write to the file",
								},
							},
							"required": []string{"file_path", "content"},
						},
					}),
					openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
						Name:        "Bash",
						Description: openai.String("Excute a shell command"),
						Parameters: openai.FunctionParameters{
							"type": "object",
							"properties": map[string]any{
								"command": map[string]any{
									"type":        "string",
									"description": "The command to excute",
								},
								"required": []string{"command"},
							},
						},
					}),
				},
			},
		)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if len(resp.Choices) == 0 {
			panic("No choices in response")
		}

		if len(resp.Choices[0].Message.ToolCalls) > 0 {
			toolCall := resp.Choices[0].Message.ToolCalls[0]

			if toolCall.Function.Name == "Read" {
				var args struct {
					FilePath string `json:"file_path"`
				}
				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error unmarshaling tool call arguments: %v\n", err)
					os.Exit(1)
				}

				// Read the file contents
				fileContents, err := os.ReadFile(args.FilePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
					os.Exit(1)
				}

				// Add the file contents to the message history
				message = append(message, resp.Choices[0].Message.ToParam())
				message = append(message, openai.ToolMessage(string(fileContents), toolCall.ID))
			} else if toolCall.Function.Name == "Write" {
				var args struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content"`
				}
				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error unmarshaling tool call arguments: %v\n", err)
					os.Exit(1)
				}

				// Write the file
				err = os.WriteFile(args.FilePath, []byte(args.Content), 0666)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error writing file: %v\n", err)
					os.Exit(1)
				}
				// Add the file contents to the message history
				message = append(message, resp.Choices[0].Message.ToParam())
				message = append(message, openai.ToolMessage(string("writing complete"), toolCall.ID))
			} else if toolCall.Function.Name == "Bash" {
				var args struct {
					Command string `json:"command"`
				}
				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error unmarshaling tool call arguments: %v\n", err)
					os.Exit(1)
				}

				// Do bash
				cmd := exec.Command("bash", "-c", args.Command)
				out, err := cmd.Output()

				if err != nil {
					fmt.Fprintf(os.Stderr, "error bash command: %v\n", err)
					os.Exit(1)
				}

				message = append(message, resp.Choices[0].Message.ToParam())
				message = append(message, openai.ToolMessage(string(out), toolCall.ID))
			}

		} else {
			fmt.Print(resp.Choices[0].Message.Content)
			break
		}
	}
}
