package bridge

// NewClaudeRuntime returns a Runtime that invokes the Claude CLI.
func NewClaudeRuntime() Runtime {
	return &CLIRuntime{
		id:           "claude-cli",
		binaryName:   "claude",
		defaultModel: "haiku",
	}
}
