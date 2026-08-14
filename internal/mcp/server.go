package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/Amaan-Khan14/project-ctx-graph"
	"io"
	"os"
	"time"
)

// Message envelope for JSON-RPC 2.0
type envelope struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *rpcError        `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Standard JSON-RPC error codes
const (
	parseError     = -32700
	methodNotFound = -32601
	invalidParams  = -32602
)

var clientName string // captured from initialize

// ServerInfoVersion is reported in the initialize handshake; stamped by
// release builds via ldflags.
var ServerInfoVersion = "0.1.0"

// handleMessage processes a single JSON-RPC request and returns the response (or nil for notifications)
func handleMessage(line []byte) []byte {
	var env envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return errorResponse(nil, parseError, "parse error")
	}

	// Notifications have no id → never respond
	if env.ID == nil {
		return nil
	}

	switch env.Method {
	case "initialize":
		return handleInitialize(env.ID, env.Params)
	case "tools/list":
		return handleToolsList(env.ID)
	case "tools/call":
		return handleToolsCall(env.ID, env.Params)
	default:
		return errorResponse(env.ID, methodNotFound, fmt.Sprintf("method not found: %s", env.Method))
	}
}

func handleInitialize(id *json.RawMessage, params json.RawMessage) []byte {
	var req struct {
		ProtocolVersion string `json:"protocolVersion"`
		ClientInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return errorResponse(id, invalidParams, "invalid initialize params")
	}

	clientName = req.ClientInfo.Name
	if clientName == "" {
		clientName = "unknown"
	}

	result := map[string]interface{}{
		"protocolVersion": req.ProtocolVersion,
		"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
		"serverInfo": map[string]string{
			"name":    "ctx",
			"version": ServerInfoVersion,
		},
	}

	return successResponse(id, result)
}

func handleToolsList(id *json.RawMessage) []byte {
	tools := []map[string]interface{}{
		{
			"name":        "ctx_explore",
			"description": "Retrieve project knowledge before acting: what this project already knows, decided, tried, or disputes. Pass paths you are about to modify to surface scoped constraints; query for topics; key for exact lookup; include_superseded reveals historical positions. Call before editing unfamiliar areas and before ctx_record.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Keyword match against key and statement",
					},
					"paths": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Files or directories you are about to touch",
					},
					"kind": map[string]interface{}{
						"type":        "string",
						"description": "Filter by kind",
						"enum":        []string{"decision", "constraint", "bug", "assumption", "rationale", "fact"},
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Exact key lookup (returns at most one result)",
					},
					"include_superseded": map[string]interface{}{
						"type":        "boolean",
						"description": "Include superseded entries to see historical context",
					},
				},
			},
		},
		{
			"name":        "ctx_record",
			"description": "Record project knowledge: decisions, constraints, bugs, assumptions, rationale, facts. WORKFLOW: first call ctx_explore with a related query to avoid duplicates; if the topic exists, record again with the same key to confirm/update it; if a decision replaces an older one, pass the old key in supersedes. Key rules: stable dot.case slug naming the TOPIC (one decision per key); statement = current position, 1-2 sentences; scope = paths this knowledge applies to ('.' for project-wide).",
			"inputSchema": map[string]interface{}{
				"type":     "object",
				"required": []string{"key", "kind", "statement", "scope"},
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Stable dot.case identifier",
					},
					"kind": map[string]interface{}{
						"type":        "string",
						"description": "Type of knowledge",
						"enum":        []string{"decision", "constraint", "bug", "assumption", "rationale", "fact"},
					},
					"statement": map[string]interface{}{
						"type":        "string",
						"description": "Current position, 1-2 sentences",
					},
					"scope": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Paths this knowledge applies to",
					},
					"supersedes": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Keys of older knowledge this replaces",
					},
					"note": map[string]interface{}{
						"type":        "string",
						"description": "Optional context about this evidence",
					},
					"session": map[string]interface{}{
						"type":        "string",
						"description": "Session identifier (defaults to client name)",
					},
				},
			},
		},
		{
			"name":        "ctx_dispute",
			"description": "Flag knowledge as contested when you believe an entry is wrong but cannot yet replace it; keeps it visible for humans. To CORRECT knowledge use ctx_record (same key, or new key + supersedes).",
			"inputSchema": map[string]interface{}{
				"type":     "object",
				"required": []string{"key"},
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Key of knowledge to dispute",
					},
					"note": map[string]interface{}{
						"type":        "string",
						"description": "Optional explanation of disagreement",
					},
					"session": map[string]interface{}{
						"type":        "string",
						"description": "Session identifier (defaults to client name)",
					},
				},
			},
		},
	}

	result := map[string]interface{}{"tools": tools}
	return successResponse(id, result)
}

func handleToolsCall(id *json.RawMessage, params json.RawMessage) []byte {
	var req struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return errorResponse(id, invalidParams, "invalid tools/call params")
	}

	// Validate the tool name before touching the filesystem.
	switch req.Name {
	case "ctx_explore", "ctx_record", "ctx_dispute":
	default:
		return errorResponse(id, invalidParams, fmt.Sprintf("unknown tool: %s", req.Name))
	}

	// Resolve the store fresh for EVERY call: the file is shared mutable
	// state and other sessions may write between our calls.
	_, knowledgePath, err := projectcontext.ResolveStore()
	if err != nil {
		return toolError(id, "no .ctx store found walking up from cwd; run `ctx init`")
	}

	switch req.Name {
	case "ctx_explore":
		return handleExplore(id, knowledgePath, req.Arguments)
	case "ctx_record":
		return handleRecord(id, knowledgePath, req.Arguments)
	default: // ctx_dispute
		return handleDispute(id, knowledgePath, req.Arguments)
	}
}

func handleExplore(id *json.RawMessage, knowledgePath string, args map[string]interface{}) []byte {
	store, err := projectcontext.Load(knowledgePath)
	if err != nil {
		return toolError(id, fmt.Sprintf("loading store: %v", err))
	}

	opts := projectcontext.QueryOpts{}

	if v, ok := args["query"].(string); ok {
		opts.Text = v
	}
	if v, ok := args["paths"].([]interface{}); ok {
		opts.Paths = toStringSlice(v)
	}
	if v, ok := args["kind"].(string); ok {
		opts.Kind = v
	}
	if v, ok := args["key"].(string); ok {
		opts.Key = v
	}
	if v, ok := args["include_superseded"].(bool); ok {
		opts.IncludeSuperseded = v
	}

	results := projectcontext.Query(store, opts)

	jsonBytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return toolError(id, fmt.Sprintf("marshaling results: %v", err))
	}

	return toolSuccess(id, string(jsonBytes))
}

func handleRecord(id *json.RawMessage, knowledgePath string, args map[string]interface{}) []byte {
	store, err := projectcontext.Load(knowledgePath)
	if err != nil {
		return toolError(id, fmt.Sprintf("loading store: %v", err))
	}

	key, _ := args["key"].(string)
	kind, _ := args["kind"].(string)
	statement, _ := args["statement"].(string)
	note, _ := args["note"].(string)
	session, _ := args["session"].(string)

	if session == "" {
		session = clientName
	}

	scope := toStringSlice(args["scope"])
	supersedes := toStringSlice(args["supersedes"])

	input := projectcontext.RecordInput{
		Key:        key,
		Kind:       kind,
		Statement:  statement,
		Scope:      scope,
		Supersedes: supersedes,
		Session:    session,
		Note:       note,
	}

	k, created, err := projectcontext.Record(store, input, time.Now())
	if err != nil {
		return toolError(id, err.Error())
	}

	if err := store.Save(knowledgePath); err != nil {
		return toolError(id, fmt.Sprintf("saving store: %v", err))
	}

	var msg string
	if created {
		msg = fmt.Sprintf("recorded %s", key)
	} else {
		msg = fmt.Sprintf("updated %s (evidence: %d)", key, len(k.Evidence))
	}

	return toolSuccess(id, msg)
}

func handleDispute(id *json.RawMessage, knowledgePath string, args map[string]interface{}) []byte {
	store, err := projectcontext.Load(knowledgePath)
	if err != nil {
		return toolError(id, fmt.Sprintf("loading store: %v", err))
	}

	key, _ := args["key"].(string)
	note, _ := args["note"].(string)
	session, _ := args["session"].(string)

	if session == "" {
		session = clientName
	}

	if _, err := projectcontext.Dispute(store, key, session, note, time.Now()); err != nil {
		return toolError(id, err.Error())
	}

	if err := store.Save(knowledgePath); err != nil {
		return toolError(id, fmt.Sprintf("saving store: %v", err))
	}

	return toolSuccess(id, fmt.Sprintf("disputed %s", key))
}

// Helpers

func toStringSlice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func successResponse(id *json.RawMessage, result interface{}) []byte {
	resultBytes, _ := json.Marshal(result)
	env := envelope{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultBytes,
	}
	resp, _ := json.Marshal(env)
	return resp
}

func errorResponse(id *json.RawMessage, code int, message string) []byte {
	env := envelope{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: message,
		},
	}
	resp, _ := json.Marshal(env)
	return resp
}

func toolSuccess(id *json.RawMessage, text string) []byte {
	result := map[string]interface{}{
		"content": []map[string]string{
			{"type": "text", "text": text},
		},
	}
	return successResponse(id, result)
}

func toolError(id *json.RawMessage, message string) []byte {
	result := map[string]interface{}{
		"content": []map[string]string{
			{"type": "text", "text": message},
		},
		"isError": true,
	}
	return successResponse(id, result)
}

// Serve runs the MCP stdio server
func Serve(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	buf := make([]byte, 1024*1024) // 1MB buffer for large explore results
	scanner.Buffer(buf, 1024*1024)

	writer := bufio.NewWriter(out)
	defer writer.Flush()

	for scanner.Scan() {
		line := scanner.Bytes()
		response := handleMessage(line)
		if response != nil {
			writer.Write(response)
			writer.WriteByte('\n')
			writer.Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "scanner error: %v\n", err)
		return err
	}

	return nil
}
