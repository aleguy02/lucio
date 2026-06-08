package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/adk/tool"
)

// TODO: create tool factory function here

var TAVILY_SEARCH_URL string = "https://api.tavily.com/search"

// TODO(current): Tavily tool
type TavilyWebSearchToolArgs struct {
	Query string `json:"query" jsonschema:"Natural language search query"`
}

type TavilyWebSearchToolResult struct {
	Result string `json:"result"`
	Success    bool   `json:"success"`
}

func TavilyWebSearchTool(ctx tool.Context, args TavilyWebSearchToolArgs) (TavilyWebSearchToolResult, error) {
	payload := strings.NewReader(fmt.Sprintf("{\n  \"query\": %q,\n  \"search_depth\": \"advanced\",\n  \"chunks_per_source\": 3,\n  \"max_results\": 5,\n  \"topic\": \"general\",\n  \"time_range\": null,\n  \"include_answer\": true,\n  \"include_raw_content\": false,\n  \"include_images\": false,\n  \"include_image_descriptions\": false,\n  \"include_favicon\": false,\n  \"include_domains\": [],\n  \"exclude_domains\": [\"spotify.com\"],\n  \"country\": null,\n  \"auto_parameters\": false,\n  \"exact_match\": false,\n  \"include_usage\": true,\n  \"safe_search\": false\n}", args.Query))

	req, err := http.NewRequest("POST", TAVILY_SEARCH_URL, payload)
	if err != nil {
		return TavilyWebSearchToolResult{Success: false}, err
	}

	authStr := fmt.Sprintf("Bearer %s", appConf.TavilyAPIKey)
	req.Header.Add("Authorization", authStr)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return TavilyWebSearchToolResult{Success: false}, err
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var responseMap map[string]interface{}
	if err := json.Unmarshal(body, &responseMap); err != nil {
		return TavilyWebSearchToolResult{Success: false}, err
	}

	queryVal, _ := responseMap["answer"].(string)
	return TavilyWebSearchToolResult{Result: queryVal, Success: true}, nil
}