package main

import (
	"context"
	"flag"
	"fmt"
	"os"

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
							},
							"required": []string{"command"},
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
				result, err := executeRead(toolCall.Function.Arguments)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error unmarshaling tool call arguments: %v\n", err)
					os.Exit(1)
				}
				message = append(message, resp.Choices[0].Message.ToParam())
				message = append(message, openai.ToolMessage(result, toolCall.ID))

			} else if toolCall.Function.Name == "Write" {
				err := executeWrite(toolCall.Function.Arguments)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error unmarshaling tool call arguments: %v\n", err)
					os.Exit(1)
				}
				message = append(message, resp.Choices[0].Message.ToParam())
				message = append(message, openai.ToolMessage(string("writing complete"), toolCall.ID))
			} else if toolCall.Function.Name == "Bash" {
				result, err := executeBash(toolCall.Function.Arguments)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error unmarshaling tool call arguments: %v\n", err)
					os.Exit(1)
				}
				message = append(message, resp.Choices[0].Message.ToParam())
				message = append(message, openai.ToolMessage(result, toolCall.ID))
			}

		} else {
			fmt.Print(resp.Choices[0].Message.Content)
			break
		}
	}
}
