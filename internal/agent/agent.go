package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"gopkg.in/yaml.v3"

	sp "aleguy02/spotify-tui/internal/spotify"
)

type agentConf struct {
	Model struct {
		Provider string `yaml:"provider"`
		APIKey   string `yaml:"api_key"`
		Name     string `yaml:"name"`
		URL      string `yaml:"url"`
		Settings struct {
			Thinking    bool     `yaml:"thinking"`
			Temperature *float32 `yaml:"temperature"`
		} `yaml:"settings"`
	} `yaml:"model"`
}

type callbackRes struct {
	res model.LLMResponse
	err error
}

var logger *log.Logger
var confPath = "conf.yaml"
var appConf struct {
	TavilyAPIKey   string `yaml:"tavily_api_key"`
	VerboseLogging bool   `yaml:"verbose_logging"`
}

func init() {
	f, err := os.OpenFile(filepath.Join("logs", "agent.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	logger = log.New(f, "", log.LstdFlags)

	data, err := os.ReadFile(confPath)
	if err != nil {
		logger.Printf("error reading file: %v", err)
	}
	if err := yaml.Unmarshal(data, &appConf); err != nil {
		logger.Printf("could not unmarshal yaml: %v", err)
	}
}

// NewRunner creates a ready-to-use ADK runner backed by Ollama with an in-memory session.
func NewRunner(client *sp.SpotifyClient) (session.Service, *runner.Runner, error) {
	data, err := os.ReadFile(confPath)
	if err != nil {
		logger.Printf("error reading file: %v", err)
		return nil, nil, fmt.Errorf("error reading file: %w", err)
	}

	c := agentConf{}
	if err := yaml.Unmarshal(data, &c); err != nil {
		logger.Printf("could not unmarshal yaml: %v", err)
		return nil, nil, fmt.Errorf("could not unmarshal yaml: %w", err)
	}

	if c.Model.Name == "" {
		logger.Printf("model must be set")
		return nil, nil, fmt.Errorf("model must be set")
	}

	if appConf.TavilyAPIKey == "" {
		logger.Printf("webSearch tool will not be injected: Tavily API key was not set")
	} else {
		logger.Printf("webSearch tool will be injected: Tavily API key was set")
	}

	llm, err := NewOllamaModel(c)
	if err != nil {
		logger.Printf("failed to create ollama model: %s", err)
		return nil, nil, fmt.Errorf("failed to create ollama model: %w", err)
	}
	// ctx := context.Background()
	// llm, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
	//     APIKey: "AQ.Ab8RN6KRiks08cLWeqC7kdiewrzWYqf-2GYZKipWQCVSYQWkfA",
	// })
	// if err != nil {
	//     log.Fatalf("Failed to create model: %v", err)
	// }

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

	tools, err := tools(client)
	if err != nil {
		logger.Printf("failed to create tools: %s", err)
		return nil, nil, err
	}

	// TODO(improvement, not planned):
	// 	- add compaction depending on yaml file. Why not planned: This is unecessary because the user should just be able to /clear
	ag, err := llmagent.New(llmagent.Config{
		Name:                "Lucio",
		Model:               llm,
		AfterModelCallbacks: []llmagent.AfterModelCallback{collapseNewlines},
		Instruction:         lucioInstruction,
		Tools:               tools,
	})
	if err != nil {
		logger.Printf("failed to create llm agent: %s", err)
		return nil, nil, fmt.Errorf("failed to create llm agent: %w", err)
	}

	sesh := session.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:           "spotify-tui",
		Agent:             ag,
		SessionService:    sesh,
		AutoCreateSession: true,
	})
	if err != nil {
		logger.Printf("failed to create runner: %s", err)
		return nil, nil, fmt.Errorf("failed to create runner: %w", err)
	}

	return sesh, r, nil
}
