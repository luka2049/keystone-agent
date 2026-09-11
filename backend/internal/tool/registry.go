// Package tool implements the tool registration center.
//
// All tools implement the Tool interface (see architecture.md §2.4):
//   - ID() string
//   - Name() string
//   - Description() string
//   - InputSchema() map[string]any
//   - Execute(ctx, params, ToolContext) (*ToolResult, error)
//
// M0: scaffold placeholder.
package tool
