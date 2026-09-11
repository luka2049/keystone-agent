// Package agent implements the Agent runtime: execution engine,
// scheduler, session state, and multi-agent orchestration.
//
// M0: scaffold placeholder. M1 will implement:
//   - Engine: LLM↔tool loop with maxIterations guard
//   - Scheduler: goroutine + semaphore for concurrent tool calls
//   - Session: context truncation / snapshot
package agent
