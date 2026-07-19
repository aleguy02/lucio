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
	logger.Printf("called GenerateContent: model=%s messages=%d", req.Model, len(req.Contents))
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

func makeOllamaMessages(contents []*genai.Content, size int) []ollama.Message {
	msgs := make([]ollama.Message, 0, size)

	for _, content := range contents {
		role := content.Role
		if role == "model" {
			role = "assistant" // ollama uses OpenAI API compatible format, which expects an "assistant" role
		}

		var (
			text            strings.Builder
			toolCalls       []ollama.ToolCall // agents can do multiple tool calls in one turn (i.e. parallel tools), so we make an array
			toolResponse    ollama.Message
			hasToolResponse bool
		)

		for _, part := range content.Parts {
			switch {
			case part.FunctionCall != nil:
				args := ollama.NewToolCallFunctionArguments()
				for k, v := range part.FunctionCall.Args {
					args.Set(k, v)
				}
				toolCalls = append(toolCalls, ollama.ToolCall{
					Function: ollama.ToolCallFunction{
						Name:      part.FunctionCall.Name,
						Arguments: args,
					},
				})

			case part.FunctionResponse != nil:
				content, _ := json.Marshal(part.FunctionResponse.Response)
				toolResponse = ollama.Message{
					Role:     "tool",
					Content:  string(content),
					ToolName: part.FunctionResponse.Name,
				}
				hasToolResponse = true

			case !part.Thought:
				text.WriteString(part.Text)

			case part.Thought:
				text.WriteString("<THINKING>")
				text.WriteString(part.Text)
				text.WriteString("<THINKING>")
			}
		}

		switch {
		case hasToolResponse:
			msgs = append(msgs, toolResponse)
		case len(toolCalls) > 0:
			msgs = append(msgs, ollama.Message{
				Role:      role,
				ToolCalls: toolCalls,
			})
		default:
			msgs = append(msgs, ollama.Message{
				Role:    role,
				Content: text.String(),
			})
		}
	}

	return msgs
}

// Model-layer function to generate an iterable stream of tokens. This function handles translation between ADK and Ollama.
func (m *ollamaLLM) generateStream(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		ch := make(chan callbackRes)

		// First half of this function is a translation layer between ADK <--> Ollama
		// ADK --> Ollama parsing happens here
		msgs := makeOllamaMessages(req.Contents, len(req.Contents))

		// reinject system prompt into every interaction. It isn't part of the messages
		if req.Config != nil && req.Config.SystemInstruction != nil && req.Config.SystemInstruction.Parts[0].Text != "" {
			msgs = append([]ollama.Message{{Role: "system", Content: req.Config.SystemInstruction.Parts[0].Text}}, msgs...)
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
		var accumulatedThinking strings.Builder

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
				accumulatedThinking.WriteString(resp.Message.Thinking)
				// Only forward non-empty tokens; skip thinking-only tokens.
				// TODO(improvement): forward a signal that the model is thinking, to surface in UI
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
			thought := accumulatedThinking.String()
			ch <- callbackRes{res: model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: thought, Thought: true}},
					Role:  "model",
				},
			}}
			ch <- callbackRes{res: model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: text, Thought: false}},
					Role:  "model",
				},
				TurnComplete: true,
			}}
			return nil
		}
		go func() {
			prettyJSON, err := json.MarshalIndent(oReq.Messages, "", "  ")
			if err != nil {
				logger.Fatalf("Failed to prettify JSON: %v", err)
			}
			logger.Printf("message history:\n%s", prettyJSON)
			err = m.client.Chat(ctx, oReq, respFunc)
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
		logger.Printf("using remote Ollama server: %v", u)
		c := &http.Client{}
		client = ollama.NewClient(u, c) // TODO: what happens if the url string parses correctly but is wrong?
	} else {
		logger.Printf("using local Ollama server")
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
