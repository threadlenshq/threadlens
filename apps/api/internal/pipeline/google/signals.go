// Package google implements the Google search scout pipeline utilities.
package google

// PROBLEM_TERMS mirrors PROBLEM_TERMS from signals.js.
var PROBLEM_TERMS = []string{
	"pain", "painful", "frustrating", "frustration", "stuck", "blocked", "hard", "difficult",
	"lost context", "losing context", "re-onboarding", "relearning", "re-learning",
	"quit", "abandon", "overwhelmed", "struggle",
}

// AUDIENCE_TERMS mirrors AUDIENCE_TERMS from signals.js.
var AUDIENCE_TERMS = []string{
	"developer", "developers", "programmer", "programmers", "coding", "software engineer", "engineer", "side project",
}

// WORKFLOW_TERMS mirrors WORKFLOW_TERMS from signals.js.
var WORKFLOW_TERMS = []string{
	"workflow", "process", "resume coding project", "restart", "re-entry", "context switch", "momentum", "return to",
}

// ACTIONABILITY_TERMS mirrors ACTIONABILITY_TERMS from signals.js.
var ACTIONABILITY_TERMS = []string{
	"guide", "checklist", "template", "steps", "step-by-step", "tool", "how to", "lightweight way",
}
