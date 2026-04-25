package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	ollama "github.com/ollama/ollama/api"
)

var fileLog *log.Logger

func init() {
	f, err := os.OpenFile(filepath.Join("logs", "agent.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	fileLog = log.New(f, "", log.LstdFlags)
}

type myLLM struct {
	client   *ollama.Client
	modelStr string
	name     string
}

func NewOllamaModel(modelName string, urlStr string) (*myLLM, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	var client *ollama.Client
	if urlStr != "" {
		c := &http.Client{}
		client = ollama.NewClient(u, c) // TODO: what happens if the url string parses correctly but is wrong?
	} else {
		client, err = ollama.ClientFromEnvironment()
		if err != nil {
			return nil, fmt.Errorf("failed create ollama client from environment: %w", err)
		}
	}

	return &myLLM{
		client:   client,
		modelStr: modelName,
		name:     modelName,
	}, nil
}

func (m *myLLM) Name() string {
	return m.name
}

func (m *myLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	// TODO
	if !stream {
		panic("whoopsy fucking daisy! only stream is supported")
	}
	return m.generateStream(ctx, req)
}

type callbackRes struct {
	res model.LLMResponse
	err error
}

func (m *myLLM) generateStream(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		ch := make(chan callbackRes)

		msgs := make([]ollama.Message, 0, len(req.Contents))
		for _, c := range req.Contents {
			role := c.Role
			if role == "model" {
				role = "assistant"
			}
			var text string
			for _, p := range c.Parts {
				text += p.Text
			}
			msgs = append(msgs, ollama.Message{Role: role, Content: text})
		}

		// reinject system prompt into every interaction. It isn't part of the messages
		if req.Config != nil && req.Config.SystemInstruction != nil {
			var systemText string
			for _, p := range req.Config.SystemInstruction.Parts {
				systemText += p.Text
			}

			// TODO: it might be guaranteed that systemText exists since I'm coding this. I'll come back to it
			if systemText != "" {
				msgs = append([]ollama.Message{{Role: "system", Content: systemText}}, msgs...)
			}
		}

		// TODO(debug): remove in prod. Also there should be more logs for the agent I'll add those somewhere eventually
		if b, err := json.MarshalIndent(msgs, "", "  "); err == nil {
			fileLog.Printf("message history:\n%s", b)
		}

		stream := true
		oReq := &ollama.ChatRequest{
			Model:    req.Model,
			Messages: msgs,
			Think: &ollama.ThinkValue{
				Value: true,
			},
			Stream: &stream,
		}

		// Convert message content
		// Convert tool calls
		// Convert usage metadata
		// pass messages to a channel from inside the respFunc, which is called for every streamed response?
		var accumulated strings.Builder
		respFunc := func(resp ollama.ChatResponse) error {
			accumulated.WriteString(resp.Message.Content)
			text := resp.Message.Content

			if resp.Done {
				text = accumulated.String()
			}
			part := &genai.Part{Text: text}

			res := model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{part},
					Role:  "model",
				},
				Partial:      !resp.Done,
				TurnComplete: resp.Done,
			}

			ch <- callbackRes{
				res: res,
				err: nil,
			}

			return nil
		}
		go func() {
			if err := m.client.Chat(ctx, oReq, respFunc); err != nil {
				ch <- callbackRes{err: err}
			}
			close(ch)
		}()

		for {
			res, ok := <-ch
			if !ok {
				break
			}
			if res.err != nil {
				yield(nil, res.err)
				return
			}
			if !yield(&res.res, res.err) {
				return
			}
		}
	}

}

// NewRunner creates a ready-to-use ADK runner backed by Ollama with an in-memory session.
func NewRunner(modelName, urlStr string) (*runner.Runner, error) {
	llm, err := NewOllamaModel(modelName, urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama model: %w", err)
	}

	// TODO(bug): sometimes newlines render extra tall sometimes not. It makes the agent response look messed up
	// this is a temporary fix to clamp extra newlines to mitigate the issue
	doubleNewline := regexp.MustCompile(`\n\n+`)
	collapseNewlines := func(_ agent.CallbackContext, resp *model.LLMResponse, respErr error) (*model.LLMResponse, error) {
		if respErr != nil || resp == nil || resp.Content == nil {
			return resp, respErr
		}
		for _, p := range resp.Content.Parts {
			p.Text = doubleNewline.ReplaceAllString(p.Text, "\n")
		}
		return resp, nil
	}

	// TODO(improvement):
	// 	- add compaction depending on yaml file
	ag, err := llmagent.New(llmagent.Config{
		Name:                "spotify_agent",
		Model:               llm,
		AfterModelCallbacks: []llmagent.AfterModelCallback{collapseNewlines},
		Instruction: "Always respond in fewer than 200 words.",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create llm agent: %w", err)
	}

	r, err := runner.New(runner.Config{
		AppName:           "spotify-tui",
		Agent:             ag,
		SessionService:    session.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	return r, nil
}
