package insight

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/HerbHall/subnetree/pkg/analytics"
	"github.com/HerbHall/subnetree/pkg/llm"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/roles"
)

// intentParserSystemPrompt instructs the LLM to parse user queries into structured JSON intents.
const intentParserSystemPrompt = `You are a query parser for a network monitoring system called SubNetree. Parse the user's natural language question into a structured JSON intent.

Available intent types:
- "list_anomalies": Show detected anomalies. Optional "device_id" to filter by device, optional "limit" (default 50).
- "list_baselines": Show learned baselines for a device. Requires "device_id".
- "list_forecasts": Show capacity forecasts for a device. Requires "device_id".
- "list_correlations": Show active alert correlation groups. No parameters needed.
- "list_devices": Show all discovered network devices. No parameters needed.
- "device_status": Show comprehensive status for a specific device including anomalies, baselines, and forecasts. Requires "device_id".

Output format — return ONLY valid JSON, no explanation:
{"type":"<intent_type>","device_id":"<id_if_applicable>","limit":<number_if_applicable>}

Examples:
- "show me recent anomalies" → {"type":"list_anomalies","limit":10}
- "any anomalies on router-01?" → {"type":"list_anomalies","device_id":"router-01"}
- "what devices are on the network?" → {"type":"list_devices"}
- "status of web-server-01" → {"type":"device_status","device_id":"web-server-01"}
- "forecasts for db-primary" → {"type":"list_forecasts","device_id":"db-primary"}
- "are there correlated alerts?" → {"type":"list_correlations"}
- "baselines for switch-core" → {"type":"list_baselines","device_id":"switch-core"}`

// responseFormatterTemplate is used for the second LLM call that converts structured data
// into a natural language answer.
const responseFormatterTemplate = `You are answering a user's question about their network monitoring system (SubNetree).

User question: %s
Query type: %s
Data (JSON):
%s

Provide a clear, concise natural language answer based on the data above. If the data is empty or contains no items, say so politely. Focus on actionable insights rather than repeating raw numbers. Keep the response under 200 words.`

// nlQueryProcessor handles natural language query translation and execution.
type nlQueryProcessor struct {
	llmProvider llm.Provider
	store       *InsightStore
	plugins     plugin.PluginResolver
}

// newNLQueryProcessor creates a processor by resolving the LLM plugin.
// Returns nil if no LLM provider is available.
func newNLQueryProcessor(plugins plugin.PluginResolver, store *InsightStore) *nlQueryProcessor {
	if plugins == nil {
		return nil
	}

	providers := plugins.ResolveByRole(roles.RoleLLM)
	if len(providers) == 0 {
		return nil
	}

	llmPlugin, ok := providers[0].(roles.LLMProvider)
	if !ok {
		return nil
	}

	return &nlQueryProcessor{
		llmProvider: llmPlugin.Provider(),
		store:       store,
		plugins:     plugins,
	}
}

// Process executes a natural language query through a two-phase LLM pipeline:
// 1. Parse the user's question into a structured intent (low temperature).
// 2. Execute the intent against the store/plugins.
// 3. Format the results into a natural language answer (moderate temperature).
func (p *nlQueryProcessor) Process(ctx context.Context, query string) (*analytics.NLQueryResponse, error) {
	intent, model, err := p.parseIntent(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("parse intent: %w", err)
	}

	structured, err := intent.execute(ctx, p.store, p.plugins)
	if err != nil {
		return nil, fmt.Errorf("execute intent: %w", err)
	}

	answer, err := p.formatResponse(ctx, query, intent, structured)
	if err != nil {
		return nil, fmt.Errorf("format response: %w", err)
	}

	return &analytics.NLQueryResponse{
		Query:      query,
		Answer:     answer,
		Structured: structured,
		Model:      model,
	}, nil
}

// parseIntent sends the user query to the LLM with a system prompt that instructs it
// to return a JSON intent object.
func (p *nlQueryProcessor) parseIntent(ctx context.Context, query string) (*queryIntent, string, error) {
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: intentParserSystemPrompt},
		{Role: llm.RoleUser, Content: query},
	}

	resp, err := p.llmProvider.Chat(ctx, messages,
		llm.WithTemperature(0.1),
		llm.WithMaxTokens(256),
	)
	if err != nil {
		return nil, "", err
	}

	var intent queryIntent
	if err := json.Unmarshal([]byte(resp.Content), &intent); err != nil {
		return nil, "", fmt.Errorf("LLM returned invalid JSON: %w", err)
	}

	return &intent, resp.Model, nil
}

// formatResponse sends a second LLM call to convert structured query results
// into a natural language answer.
func (p *nlQueryProcessor) formatResponse(ctx context.Context, query string, intent *queryIntent, data any) (string, error) {
	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		dataJSON = []byte("[]")
	}

	prompt := fmt.Sprintf(responseFormatterTemplate, query, intent.Type, string(dataJSON))

	resp, err := p.llmProvider.Generate(ctx, prompt,
		llm.WithTemperature(0.7),
		llm.WithMaxTokens(1024),
	)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}
