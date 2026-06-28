package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strings"

	ollama "github.com/ollama/ollama/api"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// ollamaLLM is my implementation of ADK's LLM interface because ADK's Go port doesn't have a native connector for Ollama
// See:
//   - https://adk.dev/agents/models/ollama/
//   - https://pkg.go.dev/google.golang.org/adk@v1.1.0/agent/llmagent#Config
//   - https://pkg.go.dev/google.golang.org/adk@v1.1.0/model#LLM
type ollamaLLM struct {
	client          *ollama.Client
	modelStr        string
	name            string
	thinkingEnabled bool
	temperature     *float32
}

func (m *ollamaLLM) Name() string {
	return m.name
}

func (m *ollamaLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	// TODO
	if !stream {
		panic("whoopsy fucking daisy! only stream is supported")
	}
	return m.generateStream(ctx, req)
}

// Helper function to translate Google ADK function declaration (tool) to Ollama tool definition
func adkToolToOllamaTool(decl *genai.FunctionDeclaration) ollama.Tool {
	t := ollama.Tool{
		Type: "function",
		Function: ollama.ToolFunction{
			Name:        decl.Name,
			Description: decl.Description,
		},
	}

	// uncomment for deep debugging
	// if b, err := json.MarshalIndent(decl.Parameters, "", "  "); err == nil {
	// 	logger.Printf("decl.Parameters for %s: %s", decl.Name, b)
	// }
	// if b, err := json.MarshalIndent(decl.ParametersJsonSchema, "", "  "); err == nil {
	// 	logger.Printf("decl.ParametersJsonSchema for %s: %s", decl.Name, b)
	// }
	if decl.ParametersJsonSchema != nil {
		props := ollama.NewToolPropertiesMap()
		var required []string

		// serialize and deserialize ParametersJsonSchema through JSON to get a reliably typed map to enable subsequent ADK -> Ollama translation
		var schema map[string]any
		if b, err := json.Marshal(decl.ParametersJsonSchema); err == nil {
			_ = json.Unmarshal(b, &schema)
		}

		if propsMap, ok := schema["properties"].(map[string]any); ok {
			for name, propVal := range propsMap {
				if propDef, ok := propVal.(map[string]any); ok {
					prop := ollama.ToolProperty{}
					if typStr, ok := propDef["type"].(string); ok {
						prop.Type = ollama.PropertyType{typStr}
					}
					if desc, ok := propDef["description"].(string); ok {
						prop.Description = desc
					}
					props.Set(name, prop)
				}
			}
		}
		if req, ok := schema["required"].([]any); ok {
			for _, r := range req {
				if s, ok := r.(string); ok {
					required = append(required, s)
				}
			}
		}

		t.Function.Parameters = ollama.ToolFunctionParameters{
			Type:       "object",
			Required:   required,
			Properties: props,
		}
	}

	return t
}

// Model-layer function to generate an iterable stream of tokens. This function handles translation between ADK and Ollama.
func (m *ollamaLLM) generateStream(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		ch := make(chan callbackRes)
		logger.Printf("generateStream: model=%s messages=%d", req.Model, len(req.Contents))

		// First half of this function is a translation layer between ADK <--> Ollama
		// ADK --> Ollama parsing happens here
		msgs := make([]ollama.Message, 0, len(req.Contents))
		for _, c := range req.Contents {
			role := c.Role
			if role == "model" {
				role = "assistant"
			}

			var text string
			var toolCalls []ollama.ToolCall
			var toolResponses []ollama.Message

			for _, p := range c.Parts {
				switch {
				case p.FunctionCall != nil:
					args := ollama.NewToolCallFunctionArguments()
					for k, v := range p.FunctionCall.Args {
						args.Set(k, v)
					}
					toolCalls = append(toolCalls, ollama.ToolCall{
						// ID: p.FunctionCall.ID,  // This isn't showing up
						Function: ollama.ToolCallFunction{
							Name:      p.FunctionCall.Name,
							Arguments: args,
						},
					})

				case p.FunctionResponse != nil:
					content, _ := json.Marshal(p.FunctionResponse.Response)
					toolResponses = append(toolResponses, ollama.Message{
						Role:     "tool",
						Content:  string(content),
						ToolName: p.FunctionResponse.Name,
						// ToolCallID: p.FunctionResponse.ID,  // This isn't showing up
					})

				case !p.Thought:
					text += p.Text
				}
			}

			if len(toolResponses) > 0 {
				msgs = append(msgs, toolResponses...)
			} else if len(toolCalls) > 0 {
				msgs = append(msgs, ollama.Message{Role: role, ToolCalls: toolCalls})
			} else {
				msgs = append(msgs, ollama.Message{Role: role, Content: text})
			}
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
			logger.Printf("message history:\n%s", b)
		}

		var tools []ollama.Tool
		if req.Config != nil {
			for _, genaiTool := range req.Config.Tools {
				if genaiTool == nil {
					continue
				}
				for _, decl := range genaiTool.FunctionDeclarations {
					if decl != nil {
						tools = append(tools, adkToolToOllamaTool(decl))
					}
				}
			}
		}

		// uncomment for deep debugging
		// if b, err := json.MarshalIndent(tools, "", "  "); err == nil {
		// 	logger.Printf("tools sent to ollama:\n%s", b)
		// }

		// Second half of this function
		stream := true

		oReq := &ollama.ChatRequest{
			Model:    req.Model,
			Messages: msgs,
			Think: &ollama.ThinkValue{
				Value: m.thinkingEnabled,
			},
			Stream: &stream,
			Tools:  tools,
		}
		if m.temperature != nil {
			oReq.Options = map[string]any{"temperature": *m.temperature}
		}

		var accumulated strings.Builder
		var pendingToolCalls []ollama.ToolCall

		respFunc := func(resp ollama.ChatResponse) error {
			if appConf.VerboseLogging {
				logger.Printf("stream token: done=%v content=%q thinking=%q tool_calls=%d",
					resp.Done, resp.Message.Content, resp.Message.Thinking, len(resp.Message.ToolCalls))
			}

			// Ollama sends tool calls in a non-done streaming token, then a bare
			// done=true to close the stream. Collect them across all tokens.
			if len(resp.Message.ToolCalls) > 0 {
				pendingToolCalls = append(pendingToolCalls, resp.Message.ToolCalls...)
			}

			// This is where we stream tokens back to the user to display them in the TUI
			if !resp.Done {
				accumulated.WriteString(resp.Message.Content)
				// Only forward non-empty tokens; skip thinking-only tokens.
				if resp.Message.Content != "" {
					ch <- callbackRes{res: model.LLMResponse{
						Content: &genai.Content{
							Parts: []*genai.Part{{Text: resp.Message.Content}},
							Role:  "model",
						},
						Partial: true,
					}}
				}
				return nil
			}

			// Done — emit tool calls if any, otherwise emit the accumulated text.
			// Accumulated text is prepended when the model prefixes a tool call
			// with visible content so the full turn is preserved in session history.
			if len(pendingToolCalls) > 0 {
				var parts []*genai.Part
				if preamble := accumulated.String(); preamble != "" {
					parts = append(parts, &genai.Part{Text: preamble})
				}
				for _, tc := range pendingToolCalls {
					args := tc.Function.Arguments.ToMap()
					if b, err := json.Marshal(args); err == nil {
						logger.Printf("dispatching tool call: %s(%s)", tc.Function.Name, b)
					} else {
						logger.Printf("dispatching tool call: %s(%v)", tc.Function.Name, args)
					}
					parts = append(parts, genai.NewPartFromFunctionCall(tc.Function.Name, args))
				}
				ch <- callbackRes{res: model.LLMResponse{
					Content:      &genai.Content{Parts: parts, Role: "model"},
					TurnComplete: true,
				}}
				return nil
			}

			text := accumulated.String()
			// Some models route their entire response into Thinking and leave
			// Content empty. Fall back so the reply is never silently blank.
			// TODO(bug): I think this is dead code. I haven't observed it ever doing anything
			if text == "" && resp.Message.Thinking != "" {
				logger.Printf("content empty, falling back to thinking text (%d chars)", len(resp.Message.Thinking))
				text = "[WARNING]: content was empty, falling back to thinking text\n" + resp.Message.Thinking
			}
			ch <- callbackRes{res: model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: text}},
					Role:  "model",
				},
				TurnComplete: true,
			}}
			return nil
		}
		go func() {
			err := m.client.Chat(ctx, oReq, respFunc)
			if err != nil {
				ch <- callbackRes{err: err}
			} else {
				logger.Printf("chat completed successfully")
			}
			close(ch)
		}()

		for {
			res, ok := <-ch
			if !ok {
				break
			}
			if res.err != nil {
				logger.Printf("received error from channel: %s", res.err)
				yield(nil, res.err)
				return
			}
			if !yield(&res.res, res.err) {
				return
			}
		}
	}

}

func NewOllamaModel(c agentConf) (*ollamaLLM, error) {
	u, err := url.Parse(c.Model.URL)
	if err != nil {
		logger.Printf("failed to parse url: %s", err)
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	var client *ollama.Client
	if c.Model.URL != "" {
		c := &http.Client{}
		client = ollama.NewClient(u, c) // TODO: what happens if the url string parses correctly but is wrong?
	} else {
		client, err = ollama.ClientFromEnvironment()
		if err != nil {
			logger.Printf("failed create ollama client from environment: %s", err)
			return nil, fmt.Errorf("failed create ollama client from environment: %w", err)
		}
	}

	return &ollamaLLM{
		client:          client,
		modelStr:        c.Model.Name,
		name:            c.Model.Name,
		thinkingEnabled: c.Model.Settings.Thinking,
		temperature:     c.Model.Settings.Temperature,
	}, nil
}
