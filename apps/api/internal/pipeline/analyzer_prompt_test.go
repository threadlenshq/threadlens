package pipeline

import (
	"strings"
	"testing"
)

// The final, user-facing assessment is produced by buildSystemPrompt (small
// datasets) and buildMergeSystemPrompt (chunked datasets). Both must instruct
// the model to format the assessment as structured markdown so ReportView's
// renderAssessment turns it into a scannable list instead of one dense block.
func TestFinalAssessmentPromptsRequestStructuredFormat(t *testing.T) {
	prompts := map[string]string{
		"buildSystemPrompt(false)": buildSystemPrompt(false),
		"buildSystemPrompt(true)":  buildSystemPrompt(true),
		"buildMergeSystemPrompt":   buildMergeSystemPrompt(),
	}

	for name, prompt := range prompts {
		t.Run(name, func(t *testing.T) {
			for _, want := range []string{
				"lead paragraph",
				"numbered list",
				`\n\n`,
				"**",
			} {
				if !strings.Contains(prompt, want) {
					t.Errorf("prompt %s missing assessment-format guidance %q", name, want)
				}
			}
		})
	}
}
