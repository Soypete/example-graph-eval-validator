package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxTurns = 5

// ToolCallLog captures one tool invocation and its result.
type ToolCallLog struct {
	Tool      string `json:"tool"`
	Input     string `json:"input"`
	Output    string `json:"output"`
	ParsedOK  bool   `json:"parsed_ok"`
	HasData   bool   `json:"has_data"`
}

// AgentResult captures the outcome of one agent run.
type AgentResult struct {
	ToolCalled       bool
	QueryParsed      bool
	DataReturned     bool
	QuestionAnswered bool
	FinalAnswer      string
	TurnsUsed        int
	QueriesAttempted int
	LatencyMS        int64
	Error            string
	ToolCalls        []ToolCallLog
}

// chatMessage represents a message in the OpenAI chat format.
type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// chatRequest is the OpenAI-compatible request body.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []any         `json:"tools,omitempty"`
}

// chatResponse is the OpenAI-compatible response body.
type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Agent runs a ReAct loop against the nemotron model.
type Agent struct {
	endpoint     string
	modelID      string
	httpClient   *http.Client
	sparqlTool   *SPARQLTool
	tboxContent  string // TBox + prefixes only (no ABox data)
}

// NewAgent creates a new agent. Extracts TBox content from the full TTL,
// cutting off at the ABox section so the model sees schema but not instance data.
func NewAgent(endpoint, modelID string, sparqlTool *SPARQLTool, ttlContent string) *Agent {
	tbox := extractTBox(ttlContent)
	return &Agent{
		endpoint: endpoint,
		modelID:  modelID,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		sparqlTool:  sparqlTool,
		tboxContent: tbox,
	}
}

// extractTBox returns everything up to (but not including) the ABox section.
func extractTBox(ttl string) string {
	marker := "# ABox: Instance Data"
	idx := strings.Index(ttl, marker)
	if idx > 0 {
		return strings.TrimRight(ttl[:idx], "\n ") + "\n\n# (ABox instance data omitted -- use sparql_query to query instances in the sng: namespace)\n"
	}
	return ttl
}

// readOntologyToolDef returns the tool definition for reading the TTL file.
func readOntologyToolDef() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "read_ontology",
			"description": "Read the music genre ontology schema (TBox) including namespace prefixes, class definitions, properties, and concept scheme. Does NOT include ABox instance data -- use sparql_query to find instances. Returns the TBox portion of the Turtle file.",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

const systemPrompt = `You are a knowledge graph query assistant. You have access to a music genre ontology loaded as an RDF graph.

Your job is to answer questions about music genres by querying the graph using SPARQL.

You have two tools:
1. read_ontology - Read the TTL file to understand the schema, classes, properties, and available data. Use this first if you're unsure about the structure.
2. sparql_query - Execute a SPARQL SELECT query against the loaded graph.

Strategy:
- If you're unsure about the ontology structure, call read_ontology first to inspect it.
- Write SPARQL queries using the prefixes and properties described in the tool.
- If a query returns no results, try adjusting your query (different property, broader pattern).
- If the data simply doesn't exist in the ontology, say so clearly.
- Give a concise final answer based on the query results.`

// Run executes one agent run for a question. Returns the result.
func (a *Agent) Run(question string) AgentResult {
	start := time.Now()

	tools := []any{
		a.sparqlTool.ToolDefinition(),
		readOntologyToolDef(),
	}

	messages := []chatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: question},
	}

	result := AgentResult{}

	for turn := 0; turn < maxTurns; turn++ {
		result.TurnsUsed = turn + 1

		resp, err := a.callModel(messages, tools)
		if err != nil {
			result.Error = fmt.Sprintf("model call error: %v", err)
			break
		}

		if len(resp.Choices) == 0 {
			result.Error = "empty response from model"
			break
		}

		choice := resp.Choices[0]
		assistantMsg := choice.Message

		// Add assistant message to history
		messages = append(messages, assistantMsg)

		// If no tool calls, we have the final answer
		if len(assistantMsg.ToolCalls) == 0 || choice.FinishReason == "stop" {
			result.FinalAnswer = assistantMsg.Content
			result.QuestionAnswered = isSubstantiveAnswer(assistantMsg.Content)
			break
		}

		// Process tool calls
		for _, tc := range assistantMsg.ToolCalls {
			result.ToolCalled = true

			var toolResult string

			switch tc.Function.Name {
			case "sparql_query":
				result.QueriesAttempted++
				var args struct {
					Query string `json:"query"`
				}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					toolResult = fmt.Sprintf(`{"error": "invalid arguments: %s"}`, err.Error())
					result.ToolCalls = append(result.ToolCalls, ToolCallLog{
						Tool: "sparql_query", Input: tc.Function.Arguments, Output: toolResult,
					})
				} else {
					tcLog := ToolCallLog{Tool: "sparql_query", Input: args.Query}
					out, err := a.sparqlTool.Execute(args.Query)
					if err != nil {
						toolResult = fmt.Sprintf(`{"error": "%s"}`, err.Error())
						tcLog.Output = toolResult
					} else {
						result.QueryParsed = true
						tcLog.ParsedOK = true
						if !strings.Contains(out, `"count": 0`) {
							result.DataReturned = true
							tcLog.HasData = true
						}
						toolResult = out
						tcLog.Output = out
					}
					result.ToolCalls = append(result.ToolCalls, tcLog)
				}

			case "read_ontology":
				toolResult = a.tboxContent
				result.ToolCalls = append(result.ToolCalls, ToolCallLog{
					Tool: "read_ontology", Input: "", Output: toolResult,
				})

			default:
				toolResult = fmt.Sprintf(`{"error": "unknown tool: %s"}`, tc.Function.Name)
			}

			// Add tool response to history
			messages = append(messages, chatMessage{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}

	result.LatencyMS = time.Since(start).Milliseconds()
	return result
}

// callModel makes a chat completion request to the OpenAI-compatible endpoint.
func (a *Agent) callModel(messages []chatMessage, tools []any) (*chatResponse, error) {
	reqBody := chatRequest{
		Model:    a.modelID,
		Messages: messages,
		Tools:    tools,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(a.endpoint, "/") + "/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	return &chatResp, nil
}

// isSubstantiveAnswer checks if the answer is non-trivial.
func isSubstantiveAnswer(answer string) bool {
	if len(strings.TrimSpace(answer)) < 10 {
		return false
	}
	lower := strings.ToLower(answer)
	noAnswerPhrases := []string{
		"i don't know",
		"i cannot",
		"no data",
		"not available",
		"unable to",
		"no information",
	}
	for _, phrase := range noAnswerPhrases {
		if strings.Contains(lower, phrase) {
			return false
		}
	}
	return true
}
