package bridge

// NewCopilotRuntime returns a Runtime that invokes the GitHub Copilot CLI.
func NewCopilotRuntime() Runtime {
	return &CLIRuntime{
		id:           "copilot",
		binaryName:   "copilot",
		defaultModel: "gpt-5-mini",
	}
}
