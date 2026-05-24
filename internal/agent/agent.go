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
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
	"gopkg.in/yaml.v3"

	sp "aleguy02/spotify-tui/internal/spotify"

	ollama "github.com/ollama/ollama/api"
)

var logger *log.Logger
var confPath = "conf.yaml"

func init() {
	f, err := os.OpenFile(filepath.Join("logs", "agent.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	logger = log.New(f, "", log.LstdFlags)
}

type agentConf struct {
	ModelName      string `yaml:"model"`
	ModelURL       string `yaml:"model_url"`
	VerboseLogging bool   `yaml:"verbose_logging"`
	Thinking       bool   `yaml:"thinking"`
}

type myLLM struct {
	client          *ollama.Client
	modelStr        string
	name            string
	verboseLogging  bool
	thinkingEnabled bool
}

func NewOllamaModel(c agentConf) (*myLLM, error) {
	u, err := url.Parse(c.ModelURL)
	if err != nil {
		logger.Printf("failed to parse url: %s", err)
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	var client *ollama.Client
	if c.ModelURL != "" {
		c := &http.Client{}
		client = ollama.NewClient(u, c) // TODO: what happens if the url string parses correctly but is wrong?
	} else {
		client, err = ollama.ClientFromEnvironment()
		if err != nil {
			logger.Printf("failed create ollama client from environment: %s", err)
			return nil, fmt.Errorf("failed create ollama client from environment: %w", err)
		}
	}

	return &myLLM{
		client:          client,
		modelStr:        c.ModelName,
		name:            c.ModelName,
		verboseLogging:  c.VerboseLogging,
		thinkingEnabled: c.Thinking,
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

// func genaiSchemaToOllamaProperty(s *genai.Schema) ollama.ToolProperty {
// 	if s == nil {
// 		return ollama.ToolProperty{}
// 	}

// 	prop := ollama.ToolProperty{
// 		Type:        ollama.PropertyType{strings.ToLower(string(s.Type))},
// 		Description: s.Description,
// 	}

// 	if len(s.Enum) > 0 {
// 		prop.Enum = make([]any, len(s.Enum))
// 		for i, e := range s.Enum {
// 			prop.Enum[i] = e
// 		}
// 	}

// 	if len(s.AnyOf) > 0 {
// 		prop.AnyOf = make([]ollama.ToolProperty, len(s.AnyOf))
// 		for i, sub := range s.AnyOf {
// 			prop.AnyOf[i] = genaiSchemaToOllamaProperty(sub)
// 		}
// 	}

// 	if s.Items != nil {
// 		items := genaiSchemaToOllamaProperty(s.Items)
// 		prop.Items = items
// 	}

// 	if len(s.Properties) > 0 {
// 		pm := ollama.NewToolPropertiesMap()
// 		for name, sub := range s.Properties {
// 			pm.Set(name, genaiSchemaToOllamaProperty(sub))
// 		}
// 		prop.Properties = pm
// 		prop.Required = s.Required
// 	}

// 	return prop
// }

func genaiDeclToOllamaTool(decl *genai.FunctionDeclaration) ollama.Tool {
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
						ID: p.FunctionCall.ID,
						Function: ollama.ToolCallFunction{
							Name:      p.FunctionCall.Name,
							Arguments: args,
						},
					})
				case p.FunctionResponse != nil:
					content, _ := json.Marshal(p.FunctionResponse.Response)
					toolResponses = append(toolResponses, ollama.Message{
						Role:       "tool",
						Content:    string(content),
						ToolName:   p.FunctionResponse.Name,
						ToolCallID: p.FunctionResponse.ID,
					})
				case !p.Thought:
					text += p.Text
				}
			}

			if len(toolResponses) > 0 {
				msgs = append(msgs, toolResponses...)
			} else if len(toolCalls) > 0 {
				msgs = append(msgs, ollama.Message{Role: role, Content: text, ToolCalls: toolCalls})
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

		stream := true

		var tools []ollama.Tool
		if req.Config != nil {
			for _, genaiTool := range req.Config.Tools {
				if genaiTool == nil {
					continue
				}
				for _, decl := range genaiTool.FunctionDeclarations {
					if decl != nil {
						tools = append(tools, genaiDeclToOllamaTool(decl))
					}
				}
			}
		}

		// uncomment for deep debugging
		// if b, err := json.MarshalIndent(tools, "", "  "); err == nil {
		// 	logger.Printf("tools sent to ollama:\n%s", b)
		// }

		oReq := &ollama.ChatRequest{
			Model:    req.Model,
			Messages: msgs,
			Think: &ollama.ThinkValue{
				Value: m.thinkingEnabled,
			},
			Stream: &stream,
			Tools:  tools,
		}

		var accumulated strings.Builder
		var pendingToolCalls []ollama.ToolCall

		respFunc := func(resp ollama.ChatResponse) error {
			if m.verboseLogging {
				logger.Printf("stream token: done=%v content=%q thinking=%q tool_calls=%d",
					resp.Done, resp.Message.Content, resp.Message.Thinking, len(resp.Message.ToolCalls))
			}

			// Ollama sends tool calls in a non-done streaming token, then a bare
			// done=true to close the stream. Collect them across all tokens.
			if len(resp.Message.ToolCalls) > 0 {
				pendingToolCalls = append(pendingToolCalls, resp.Message.ToolCalls...)
			}

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
					logger.Printf("dispatching tool call: %s(%v)", tc.Function.Name, args)
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

// NewRunner creates a ready-to-use ADK runner backed by Ollama with an in-memory session.
func NewRunner(client *sp.SpotifyClient) (*runner.Runner, error) {
	data, err := os.ReadFile(confPath)
	if err != nil {
		logger.Printf("error reading file: %v", err)
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	c := agentConf{}
	if err := yaml.Unmarshal(data, &c); err != nil {
		logger.Printf("could not unmarshal yaml: %v", err)
		return nil, fmt.Errorf("could not unmarshal yaml: %w", err)
	}

	if c.ModelName == "" {
		logger.Printf("model must be set")
		return nil, fmt.Errorf("model must be set")
	}

	llm, err := NewOllamaModel(c)
	if err != nil {
		logger.Printf("failed to create ollama model: %s", err)
		return nil, fmt.Errorf("failed to create ollama model: %w", err)
	}

	// TODO(bug): sometimes newlines render extra tall sometimes not. It makes the agent response look messed up
	// this is a temporary fix to clamp extra newlines to mitigate the issue
	doubleNewline := regexp.MustCompile(`\n\n+`)
	// The ADK treats a non-nil return as "override the response"; return nil when
	// making no structural changes so the normal path is taken for tool-call events.
	collapseNewlines := func(_ agent.CallbackContext, resp *model.LLMResponse, respErr error) (*model.LLMResponse, error) {
		if respErr != nil || resp == nil || resp.Content == nil {
			return nil, respErr
		}
		modified := false
		for _, p := range resp.Content.Parts {
			collapsed := doubleNewline.ReplaceAllString(p.Text, "\n")
			if collapsed != p.Text {
				p.Text = collapsed
				modified = true
			}
		}
		if !modified {
			return nil, nil
		}
		return resp, nil
	}

	skipfTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifySkipTrack",
			Description: "Skip the current song/track playing in Spotify",
		}, client.SkipNextTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	skipbTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyPreviousTrack",
			Description: "Skip to the previous song/track playing in Spotify",
		}, client.SkipPreviousTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	userPlaylistsTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyGetUserPlaylists",
			Description: "Get a list of the current user's playlists on Spotify. If the user asks for all playlists, leverage the has_more field in the *return* value to determine if you need to make multiple calls to this tool.",
		}, client.GetUserPlaylistsTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	playItemTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyPlayItem",
			Description: "Play a track, album, artist, or playlist item on Spotify",
		}, client.PlayItemTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	searchSpotifyTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifySearch",
			Description: "Search for a track, album, artist, or playlist on Spotify",
		}, client.SearchSpotify)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	addToQueueTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyAddToQueue",
			Description: "Add a track to the queue on Spotify. *Only* supports tracks.",
		}, client.AddToQueue)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	// TODO(improvement, not planned):
	// 	- add compaction depending on yaml file. Why not planned: This is unecessary because the user should just be able to /clear
	ag, err := llmagent.New(llmagent.Config{
		Name:                "Lucio",
		Model:               llm,
		AfterModelCallbacks: []llmagent.AfterModelCallback{collapseNewlines},
		Instruction:         lucioInstruction,
		Tools: []tool.Tool{
			skipfTool,
			skipbTool,
			userPlaylistsTool,
			playItemTool,
			searchSpotifyTool,
			addToQueueTool,
		},
	})
	if err != nil {
		logger.Printf("failed to create llm agent: %s", err)
		return nil, fmt.Errorf("failed to create llm agent: %w", err)
	}

	r, err := runner.New(runner.Config{
		AppName:           "spotify-tui",
		Agent:             ag,
		SessionService:    session.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		logger.Printf("failed to create runner: %s", err)
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	return r, nil
}

// TODO(test): how do I test this shi
