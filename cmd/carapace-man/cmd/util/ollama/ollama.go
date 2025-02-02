package ollama

import (
	"context"
	"fmt"

	"github.com/jmorganca/ollama/api"
)

type Client struct {
	client *api.Client
	model  string
}

func NewClient(model string) (*Client, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}
	return &Client{
		client,
		model,
	}, nil
}

func (c Client) chat(content string) (string, error) {
	stream := false
	req := api.ChatRequest{
		Model:  c.model,
		Stream: &stream,
		Messages: []api.Message{
			{
				Role:    "user",
				Content: content,
			},
		},
	}
	var response string
	err := c.client.Chat(context.Background(), &req, func(cr api.ChatResponse) error {
		response = cr.Message.Content
		return nil
	})
	return response, err

}

func (c Client) ExplainCommand(command string) (string, error) {
	return c.chat(fmt.Sprintf(
		`Explain the command %#v. Be concise. Skip additional context information`,
		command,
	))
}

func (c Client) ExplainFlag(command, flag string) (string, error) {
	return c.chat(fmt.Sprintf(
		`Explain the flag %#v. Be concise. Skip additional context information`,
		command+" "+flag,
	))
}
