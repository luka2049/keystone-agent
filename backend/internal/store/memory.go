// Package store implements persistence for Keystone.
//
// M1: in-memory implementation (no external dependencies).
// M2: PostgreSQL for durable storage + Redis for session cache.
package store

import (
	"sync"
	"time"

	"github.com/chenlong/keystone/internal/types"
)

// MemoryStore is a thread-safe in-memory store for all Keystone entities.
type MemoryStore struct {
	mu       sync.RWMutex
	agents   map[string]*agentEntry
	sessions map[string]*types.Session
	runs     map[string]*runEntry
	traces   map[string][]types.TraceEvent
}

type agentEntry struct {
	spec     types.AgentSpec
	versions []types.AgentVersion
}

type runEntry struct {
	result types.RunResult
}

// NewMemoryStore creates an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		agents:   make(map[string]*agentEntry),
		sessions: make(map[string]*types.Session),
		runs:     make(map[string]*runEntry),
		traces:   make(map[string][]types.TraceEvent),
	}
}

// ========== Agent ==========

// SaveAgent creates or updates an Agent, auto-incrementing version.
func (s *MemoryStore) SaveAgent(spec types.AgentSpec) types.AgentSpec {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := types.Now()
	if spec.ID == "" {
		spec.ID = types.NewUUID()
		spec.CreatedAt = now
		spec.Status = types.AgentStatusDraft
	} else {
		spec.CreatedAt = now // placeholder; M2 will preserve
	}
	spec.UpdatedAt = now

	if spec.Version == "" {
		spec.Version = "0.1.0"
	} else {
		spec.Version = bumpVersion(spec.Version)
	}

	if spec.Strategy.MaxIterations == 0 {
		spec.Strategy = types.DefaultStrategy()
	}

	entry, exists := s.agents[spec.ID]
	if exists {
		// Save snapshot of previous version
		entry.versions = append(entry.versions, types.AgentVersion{
			Version:   entry.spec.Version,
			Snapshot:  entry.spec,
			CreatedAt: now,
		})
		entry.spec = spec
	} else {
		s.agents[spec.ID] = &agentEntry{
			spec:     spec,
			versions: []types.AgentVersion{},
		}
	}
	return spec
}

// GetAgent retrieves an Agent by ID.
func (s *MemoryStore) GetAgent(id string) (types.AgentSpec, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.agents[id]
	if !ok {
		return types.AgentSpec{}, false
	}
	return entry.spec, true
}

// ListAgents returns a paginated list of agents.
func (s *MemoryStore) ListAgents(page, pageSize int, status types.AgentStatus) ([]types.AgentSpec, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []types.AgentSpec
	for _, e := range s.agents {
		if status != "" && e.spec.Status != status {
			continue
		}
		all = append(all, e.spec)
	}
	total := len(all)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []types.AgentSpec{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total
}

// DeleteAgent soft-deletes an agent (sets status to disabled).
func (s *MemoryStore) DeleteAgent(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.agents[id]
	if !ok {
		return false
	}
	entry.spec.Status = types.AgentStatusDisabled
	entry.spec.UpdatedAt = types.Now()
	return true
}

// GetAgentVersions returns version history for an agent.
func (s *MemoryStore) GetAgentVersions(id string) ([]types.AgentVersion, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.agents[id]
	if !ok {
		return nil, false
	}
	return entry.versions, true
}

// ========== Session ==========

// SaveSession creates or updates a session.
func (s *MemoryStore) SaveSession(session *types.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = types.Now()
	}
	session.UpdatedAt = types.Now()
	s.sessions[session.ID] = session
}

// GetSession retrieves a session by ID.
func (s *MemoryStore) GetSession(id string) (*types.Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

// DeleteSession removes a session.
func (s *MemoryStore) DeleteSession(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, existed := s.sessions[id]
	delete(s.sessions, id)
	return existed
}

// ListSessions returns all sessions (for observability stats).
func (s *MemoryStore) ListSessions() []*types.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*types.Session, 0, len(s.sessions))
	for _, s := range s.sessions {
		result = append(result, s)
	}
	return result
}

// ========== Run ==========

// SaveRun stores a run result.
func (s *MemoryStore) SaveRun(runID string, result types.RunResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[runID] = &runEntry{result: result}
}

// GetRun retrieves a run by ID.
func (s *MemoryStore) GetRun(id string) (types.RunResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.runs[id]
	if !ok {
		return types.RunResult{}, false
	}
	return entry.result, true
}

// ListRuns returns all runs (for observability stats).
func (s *MemoryStore) ListRuns() []types.RunResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]types.RunResult, 0, len(s.runs))
	for _, e := range s.runs {
		result = append(result, e.result)
	}
	return result
}

// ========== Trace ==========

// SaveTrace stores trace events for a run.
func (s *MemoryStore) SaveTrace(runID string, events []types.TraceEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.traces[runID] = events
}

// GetTrace retrieves trace events for a run.
func (s *MemoryStore) GetTrace(id string) ([]types.TraceEvent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events, ok := s.traces[id]
	return events, ok
}

// ========== Helpers ==========

// bumpVersion does a naive semantic version increment.
func bumpVersion(v string) string {
	// Parse "major.minor.patch" → increment patch
	parts := splitVersion(v)
	if len(parts) == 3 {
		return parts[0] + "." + parts[1] + "." + itoa(atoi(parts[2])+1)
	}
	return v + ".1"
}

func splitVersion(v string) []string {
	var parts []string
	current := ""
	for _, c := range v {
		if c == '.' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var s []byte
	for n > 0 {
		s = append([]byte{byte('0' + n%10)}, s...)
		n /= 10
	}
	return string(s)
}

// Ensure time import is used
var _ = time.Now
