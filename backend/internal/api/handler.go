// Package api implements HTTP handlers following the OpenAPI contract.
//
// M1: hand-written handlers aligned with contracts/openapi.yaml.
// M2: replace with oapi-codegen-generated server + handler stubs.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/chenlong/keystone/internal/agent"
	"github.com/chenlong/keystone/internal/model"
	"github.com/chenlong/keystone/internal/store"
	"github.com/chenlong/keystone/internal/tool"
	"github.com/chenlong/keystone/internal/types"
)

// Server holds all dependencies for the HTTP handlers.
type Server struct {
	store  *store.MemoryStore
	engine *agent.Engine
	tools  *tool.Registry
	models *model.Registry
}

// NewServer creates a new API server with all dependencies wired.
func NewServer(st *store.MemoryStore, eng *agent.Engine, tr *tool.Registry, mr *model.Registry) *Server {
	return &Server{
		store:  st,
		engine: eng,
		tools:  tr,
		models: mr,
	}
}

// Routes returns an http.ServeMux with all API routes registered.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /healthz", s.healthz)

	// Agents
	mux.HandleFunc("GET /api/v1/agents", s.listAgents)
	mux.HandleFunc("POST /api/v1/agents", s.createAgent)
	mux.HandleFunc("GET /api/v1/agents/{id}", s.getAgent)
	mux.HandleFunc("PUT /api/v1/agents/{id}", s.updateAgent)
	mux.HandleFunc("DELETE /api/v1/agents/{id}", s.deleteAgent)
	mux.HandleFunc("GET /api/v1/agents/{id}/versions", s.listAgentVersions)

	// Tools
	mux.HandleFunc("GET /api/v1/tools", s.listTools)
	mux.HandleFunc("POST /api/v1/tools", s.registerTool)
	mux.HandleFunc("GET /api/v1/tools/{id}", s.getTool)
	mux.HandleFunc("DELETE /api/v1/tools/{id}", s.deleteTool)
	mux.HandleFunc("POST /api/v1/tools/{id}/test", s.testTool)

	// Models
	mux.HandleFunc("GET /api/v1/models", s.listModels)

	// Sessions
	mux.HandleFunc("POST /api/v1/sessions", s.createSession)
	mux.HandleFunc("GET /api/v1/sessions/{id}", s.getSession)
	mux.HandleFunc("DELETE /api/v1/sessions/{id}", s.deleteSession)
	mux.HandleFunc("POST /api/v1/sessions/{id}/messages", s.sendMessage)

	// Runs / Trace
	mux.HandleFunc("GET /api/v1/runs/{id}", s.getRun)
	mux.HandleFunc("GET /api/v1/runs/{id}/trace", s.getRunTrace)

	// Observability
	mux.HandleFunc("GET /api/v1/observability/summary", s.getObservabilitySummary)

	// CORS + JSON middleware
	return s.withMiddleware(mux)
}

// ========== Middleware ==========

func (s *Server) withMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Request logging
		fmt.Printf("[%s] %s %s\n", r.Method, r.URL.Path, r.URL.Query().Encode())
		h.ServeHTTP(w, r)
	})
}

// ========== Helpers ==========

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, types.Error{Code: code, Message: msg})
}

func parsePage(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

// ========== Health ==========

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "keystone",
	})
}

// ========== Agents ==========

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePage(r)
	status := types.AgentStatus(r.URL.Query().Get("status"))
	if status != "" && status != types.AgentStatusDraft && status != types.AgentStatusActive && status != types.AgentStatusDisabled {
		status = ""
	}
	items, total := s.store.ListAgents(page, pageSize, status)
	writeJSON(w, http.StatusOK, types.AgentList{
		Items: items, Total: total, Page: page, PageSize: pageSize,
	})
}

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	var spec types.AgentSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON: "+err.Error())
		return
	}
	if spec.Name == "" || spec.SystemPrompt == "" || spec.ModelRef == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name, systemPrompt, modelRef are required")
		return
	}
	saved := s.store.SaveAgent(spec)
	writeJSON(w, http.StatusCreated, saved)
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	spec, ok := s.store.GetAgent(id)
	if !ok {
		writeError(w, http.StatusNotFound, "AGENT_NOT_FOUND", "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

func (s *Server) updateAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.store.GetAgent(id); !ok {
		writeError(w, http.StatusNotFound, "AGENT_NOT_FOUND", "agent not found")
		return
	}
	var spec types.AgentSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	spec.ID = id
	saved := s.store.SaveAgent(spec)
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) deleteAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.store.DeleteAgent(id) {
		writeError(w, http.StatusNotFound, "AGENT_NOT_FOUND", "agent not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listAgentVersions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	versions, ok := s.store.GetAgentVersions(id)
	if !ok {
		writeError(w, http.StatusNotFound, "AGENT_NOT_FOUND", "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, versions)
}

// ========== Tools ==========

func (s *Server) listTools(w http.ResponseWriter, r *http.Request) {
	specs := s.tools.ListSpecs()
	writeJSON(w, http.StatusOK, types.ToolList{
		Items: specs, Total: len(specs),
	})
}

func (s *Server) registerTool(w http.ResponseWriter, r *http.Request) {
	var spec types.ToolSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if spec.Name == "" || spec.Description == "" || spec.InputSchema == nil || spec.Execution.Type == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name, description, inputSchema, execution.type are required")
		return
	}
	if spec.ID == "" {
		spec.ID = "tool." + string(spec.Execution.Type) + "." + sanitizeName(spec.Name)
	}
	spec.Enabled = true
	spec.CreatedAt = types.Now()

	if err := s.tools.RegisterSpec(spec); err != nil {
		writeError(w, http.StatusBadRequest, "TOOL_REGISTER_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, spec)
}

func (s *Server) getTool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	spec, ok := s.tools.GetSpec(id)
	if !ok {
		writeError(w, http.StatusNotFound, "TOOL_NOT_FOUND", "tool not found")
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

func (s *Server) deleteTool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.tools.Delete(id) {
		writeError(w, http.StatusNotFound, "TOOL_NOT_FOUND", "tool not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) testTool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, ok := s.tools.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "TOOL_NOT_FOUND", "tool not found")
		return
	}
	var req types.ToolTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	traceID := req.TraceID
	if traceID == "" {
		traceID = types.NewUUID()
	}
	result, err := t.Execute(r.Context(), req.Params, tool.ToolContext{
		TraceID: traceID,
	})
	if err != nil {
		result = &types.ToolResult{
			Success: false,
			Error:   err.Error(),
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// ========== Models ==========

func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	providers := s.models.List()
	writeJSON(w, http.StatusOK, providers)
}

// ========== Sessions ==========

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var req types.CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.AgentID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "agentId is required")
		return
	}
	// Verify agent exists
	if _, ok := s.store.GetAgent(req.AgentID); !ok {
		writeError(w, http.StatusBadRequest, "AGENT_NOT_FOUND", "agent not found")
		return
	}
	session := &types.Session{
		ID:        types.NewUUID(),
		AgentID:   req.AgentID,
		Status:    types.SessionActive,
		Messages:  []types.Message{},
		Metadata:  req.Metadata,
		CreatedAt: types.Now(),
		UpdatedAt: types.Now(),
	}
	s.store.SaveSession(session)
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	session, ok := s.store.GetSession(id)
	if !ok {
		writeError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "session not found")
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.store.DeleteSession(id) {
		writeError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "session not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	session, ok := s.store.GetSession(sessionID)
	if !ok {
		writeError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "session not found")
		return
	}

	var req types.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "content is required")
		return
	}

	// Get the agent spec
	agentSpec, ok := s.store.GetAgent(session.AgentID)
	if !ok {
		writeError(w, http.StatusInternalServerError, "AGENT_NOT_FOUND", "agent spec missing")
		return
	}

	// Append user message to session
	userMsg := types.Message{
		ID:        types.NewUUID(),
		Role:      types.RoleUser,
		Content:   req.Content,
		Timestamp: types.Now(),
	}
	session.Messages = append(session.Messages, userMsg)

	// Run the agent engine
	runID := types.NewUUID()
	traceID := types.NewUUID()
	result, traceEvents := s.engine.Run(r.Context(), agent.RunRequest{
		Session:   session,
		AgentSpec: &agentSpec,
		UserMsg:   req.Content,
		TraceID:   traceID,
		RunID:     runID,
	})

	// Store run and trace
	s.store.SaveRun(runID, *result)
	s.store.SaveTrace(runID, traceEvents)

	// Append assistant answer to session
	if result.Answer != "" {
		session.Messages = append(session.Messages, types.Message{
			ID:        types.NewUUID(),
			Role:      types.RoleAssistant,
			Content:   result.Answer,
			Timestamp: types.Now(),
		})
	}
	s.store.SaveSession(session)

	writeJSON(w, http.StatusOK, result)
}

// ========== Runs / Trace ==========

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, ok := s.store.GetRun(id)
	if !ok {
		writeError(w, http.StatusNotFound, "RUN_NOT_FOUND", "run not found")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getRunTrace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	events, ok := s.store.GetTrace(id)
	if !ok {
		writeError(w, http.StatusNotFound, "RUN_NOT_FOUND", "run not found")
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// ========== Observability ==========

func (s *Server) getObservabilitySummary(w http.ResponseWriter, r *http.Request) {
	runs := s.store.ListRuns()
	sessions := s.store.ListSessions()

	totalRuns := len(runs)
	totalMessages := 0
	totalTokens := 0
	toolCalls := 0
	toolSuccess := 0
	var latencies []float64
	errorByType := map[string]int{}

	for _, r := range runs {
		totalTokens += r.Usage.TotalTokens
		if r.Usage.LatencyMs > 0 {
			latencies = append(latencies, float64(r.Usage.LatencyMs))
		}
		if r.Status == types.RunFailed || r.Status == types.RunTimeout {
			errorByType[string(r.Status)]++
		}
	}
	for _, sess := range sessions {
		totalMessages += len(sess.Messages)
	}

	// Count tool calls from traces
	for _, sess := range sessions {
		for _, msg := range sess.Messages {
			if msg.Role == types.RoleTool {
				toolCalls++
			}
		}
	}

	toolSuccessRate := 1.0
	if toolCalls > 0 {
		toolSuccessRate = float64(toolSuccess) / float64(toolCalls)
	}

	avgLatency := 0.0
	if len(latencies) > 0 {
		sum := 0.0
		for _, l := range latencies {
			sum += l
		}
		avgLatency = sum / float64(len(latencies))
	}

	writeJSON(w, http.StatusOK, types.ObservabilitySummary{
		TotalRuns:       totalRuns,
		TotalMessages:   totalMessages,
		TotalTokens:     totalTokens,
		ToolSuccessRate: toolSuccessRate,
		AvgLatencyMs:   avgLatency,
		ErrorByType:    errorByType,
	})
}

// ========== Helpers ==========

func sanitizeName(name string) string {
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "-")
	result = strings.ReplaceAll(result, "/", "-")
	return result
}
