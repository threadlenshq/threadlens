package pipeline

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

// TrustLookup loads project-scoped trust overrides.
type TrustLookup interface {
	ListTrustRecords(ctx context.Context, projectID string) ([]domain.TrustRecord, error)
}

// AIFilterer optionally consults an AI model for ambiguous candidates.
type AIFilterer interface {
	FilterCandidate(ctx context.Context, input FilterInput) (domain.FilterDecision, error)
}

// FilterInput is the normalized candidate fed to the classifier.
type FilterInput struct {
	FindingType    string
	Platform       string
	ID             string
	Title          string
	Body           string
	URL            string
	Author         string
	Domain         string
	SourceIdentity domain.SourceIdentity
}

// FilterClassifier applies conservative, explainable filtering.
type FilterClassifier struct {
	trust TrustLookup
	ai    AIFilterer
}

// NewFilterClassifier constructs a FilterClassifier. ai may be nil.
func NewFilterClassifier(trust TrustLookup, ai AIFilterer) *FilterClassifier {
	return &FilterClassifier{trust: trust, ai: ai}
}

var aiBoilerplateRe = regexp.MustCompile(
	`(?i)(in today's fast-paced|cutting-edge solutions|unlock your potential|streamline workflows|leverage .* solutions)`,
)
var disposableAuthorRe = regexp.MustCompile(`(?i)^(user|throwaway|anon)[-_]?[0-9]{4,}$`)

// Classify normalises the input, checks source trust overrides first, then
// applies conservative deterministic rules, checks signature-based trust
// overrides on the candidate decision, and finally optionally consults AI for
// ambiguous cases. It never discards findings; callers persist the decision.
func (c *FilterClassifier) Classify(ctx context.Context, projectID string, input FilterInput) (domain.FilterDecision, error) {
	input = normalizeFilterInput(input)

	var records []domain.TrustRecord
	if c != nil && c.trust != nil {
		var err error
		records, err = c.trust.ListTrustRecords(ctx, projectID)
		if err != nil {
			return domain.FilterDecision{}, err
		}
	}

	// 1. Source-identity trust overrides win before rules.
	if tr, ok := matchSourceTrust(input, records); ok {
		return domain.FilterDecision{
			State:          domain.FilterStateVisible,
			Reason:         domain.FilterReasonTrustedOverride,
			Reasons:        []string{domain.FilterReasonTrustedOverride},
			Explanation:    "Trusted override matched: " + tr.SourceKind + " " + tr.SourceKey,
			Source:         domain.FilterSourceTrustedOverride,
			Signature:      inputSignature(input, "trusted_override"),
			SourceIdentity: input.SourceIdentity,
		}, nil
	}

	// 2. Conservative deterministic rules.
	if decision, ok := deterministicFilter(input); ok {
		// 3. Signature-based trust overrides: owner may have trusted the exact
		//    decision signature produced for this candidate, restoring it.
		if matchSignatureTrust(decision.Signature, input.Platform, records) {
			return domain.FilterDecision{
				State:          domain.FilterStateVisible,
				Reason:         domain.FilterReasonTrustedOverride,
				Reasons:        []string{domain.FilterReasonTrustedOverride},
				Explanation:    "Signature trust override matched: " + decision.Signature,
				Source:         domain.FilterSourceTrustedOverride,
				Signature:      inputSignature(input, "trusted_override"),
				SourceIdentity: input.SourceIdentity,
			}, nil
		}
		return decision, nil
	}

	// 4. Default: visible with no filter applied.
	return domain.FilterDecision{
		State:          domain.FilterStateVisible,
		Source:         domain.FilterSourceNone,
		Reasons:        []string{},
		Explanation:    "No conservative filter matched.",
		Signature:      inputSignature(input, "visible"),
		SourceIdentity: input.SourceIdentity,
	}, nil
}

// normalizeFilterInput lowercases and trims fields for consistent matching.
// It returns a copy with a new SourceIdentity map so the caller's map is not
// mutated.
func normalizeFilterInput(input FilterInput) FilterInput {
	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	input.FindingType = strings.TrimSpace(input.FindingType)
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	input.Author = strings.ToLower(strings.TrimSpace(input.Author))
	input.URL = strings.TrimSpace(input.URL)
	if input.Domain == "" {
		input.Domain = domainFromURL(input.URL)
	}
	input.Domain = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(input.Domain), "www."))
	// Clone the map to avoid mutating the caller's SourceIdentity.
	cloned := make(domain.SourceIdentity, len(input.SourceIdentity))
	for k, v := range input.SourceIdentity {
		cloned[k] = strings.ToLower(strings.TrimSpace(v))
	}
	input.SourceIdentity = cloned
	return input
}

// deterministicFilter applies conservative heuristics in order of priority.
// Returns (decision, true) when a rule fires; (zero, false) when none match.
func deterministicFilter(input FilterInput) (domain.FilterDecision, bool) {
	text := strings.ToLower(input.Title + "\n" + input.Body)

	// Promotional / self-promotion spam
	if strings.Contains(text, "i built") ||
		strings.Contains(text, "i made") ||
		strings.Contains(text, "introducing") ||
		strings.Contains(text, "launching") ||
		strings.Contains(text, "feedback welcome") ||
		strings.Contains(text, "product hunt") ||
		strings.Contains(text, "check it out at") {
		conf := 0.92
		return filteredDecision(
			input,
			domain.FilterReasonSpam,
			[]string{"promotional_launch_language"},
			"Promotional launch or feedback-request language matched conservative spam rules.",
			conf,
		), true
	}

	// Disposable author with very low-content post
	if input.Author != "" && disposableAuthorRe.MatchString(input.Author) && len(strings.Fields(input.Body)) < 18 {
		conf := 0.86
		return filteredDecision(
			input,
			domain.FilterReasonLowQualityAccount,
			[]string{"disposable_author", "low_content"},
			"Disposable-looking author with very low-content text matched low-quality-account rules.",
			conf,
		), true
	}

	// AI-generated boilerplate
	if aiBoilerplateRe.MatchString(text) {
		conf := 0.88
		return filteredDecision(
			input,
			domain.FilterReasonAIGenerated,
			[]string{"ai_boilerplate_markers"},
			"High-confidence AI-written boilerplate markers matched conservative rules.",
			conf,
		), true
	}

	// Google search spam (casino/bonus pattern)
	if input.Platform == "google" && strings.Contains(text, "casino") && strings.Contains(text, "bonus") {
		conf := 0.94
		return filteredDecision(
			input,
			domain.FilterReasonSpam,
			[]string{"suspicious_domain_content"},
			"Suspicious search-result spam language matched conservative rules.",
			conf,
		), true
	}

	return domain.FilterDecision{}, false
}

// filteredDecision builds a FilterDecision for a matched rule.
func filteredDecision(input FilterInput, primary string, codes []string, explanation string, confidence float64) domain.FilterDecision {
	reasons := append([]string{primary}, codes...)
	return domain.FilterDecision{
		State:          domain.FilterStateFiltered,
		Reason:         primary,
		Reasons:        reasons,
		Explanation:    explanation,
		Confidence:     &confidence,
		Source:         domain.FilterSourceRules,
		Signature:      inputSignature(input, strings.Join(reasons, "+")),
		SourceIdentity: input.SourceIdentity,
	}
}

// matchSourceTrust returns the first source-identity trust record that matches
// the input, respecting platform scoping.
func matchSourceTrust(input FilterInput, records []domain.TrustRecord) (domain.TrustRecord, bool) {
	for _, tr := range records {
		if tr.TrustType != domain.TrustTypeSource {
			continue
		}
		if tr.Platform != "all" && tr.Platform != input.Platform {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(tr.SourceKey))
		if input.SourceIdentity[tr.SourceKind] == key {
			return tr, true
		}
	}
	return domain.TrustRecord{}, false
}

// matchSignatureTrust returns true when any filter-signature trust record
// matches the given signature string.
func matchSignatureTrust(signature, platform string, records []domain.TrustRecord) bool {
	sig := strings.ToLower(strings.TrimSpace(signature))
	for _, tr := range records {
		if tr.TrustType != domain.TrustTypeFilterSignature {
			continue
		}
		if tr.Platform != "all" && tr.Platform != platform {
			continue
		}
		if strings.ToLower(strings.TrimSpace(tr.SourceKey)) == sig {
			return true
		}
	}
	return false
}

// inputSignature produces a stable short hash for a candidate + reason key pair.
func inputSignature(input FilterInput, reasonKey string) string {
	parts := []string{
		input.Platform,
		input.FindingType,
		reasonKey,
		input.SourceIdentity["reddit_author"],
		input.SourceIdentity["bluesky_cid"],
		input.SourceIdentity["domain"],
		strings.ToLower(input.Title),
	}
	h := sha1.Sum([]byte(strings.Join(parts, "|")))
	return "filter:" + hex.EncodeToString(h[:])[:16]
}

// domainFromURL extracts the lowercase hostname from a URL, stripping "www.".
func domainFromURL(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(parsed.Hostname(), "www."))
}

// NormalizeFetchedPostForFiltering converts a FetchedPost from the ingestion
// pipeline into a FilterInput ready for classification.
func NormalizeFetchedPostForFiltering(platform, _ string, p FetchedPost) FilterInput {
	if platform == "reddit" {
		return FilterInput{
			FindingType: domain.FindingTypePost,
			Platform:    "reddit",
			ID:          p.ID,
			Title:       p.Title,
			Body:        p.Selftext,
			URL:         p.URL,
			Author:      p.Author,
			SourceIdentity: domain.SourceIdentity{
				"reddit_author": strings.ToLower(p.Author),
				"subreddit":     strings.ToLower(p.Subreddit),
				"domain":        domainFromURL(p.URL),
			},
		}
	}
	return FilterInput{
		FindingType: domain.FindingTypePost,
		Platform:    "bluesky",
		ID:          p.ID,
		Title:       p.Text,
		Body:        p.Text,
		URL:         p.PostURL,
		Author:      p.AuthorHandle,
		SourceIdentity: domain.SourceIdentity{
			"bluesky_handle": strings.ToLower(p.AuthorHandle),
			"bluesky_cid":    strings.ToLower(p.CID),
		},
	}
}

// TrustOptionsForDecision builds the list of actionable trust targets for a
// filtered finding so the owner can restore-and-trust in one click.
func TrustOptionsForDecision(input FilterInput, signature string) []domain.FilterTrustOption {
	input = normalizeFilterInput(input)
	var out []domain.FilterTrustOption
	for _, kind := range []string{"reddit_author", "subreddit", "bluesky_cid", "bluesky_handle", "domain", "canonical_url"} {
		if key := input.SourceIdentity[kind]; key != "" {
			out = append(out, domain.FilterTrustOption{
				Platform:   input.Platform,
				TrustType:  domain.TrustTypeSource,
				SourceKind: kind,
				SourceKey:  key,
				Label:      trustLabel(input.Platform, kind, key),
			})
		}
	}
	if signature != "" {
		out = append(out, domain.FilterTrustOption{
			Platform:   input.Platform,
			TrustType:  domain.TrustTypeFilterSignature,
			SourceKind: "filter_signature",
			SourceKey:  signature,
			Label:      "Trust exact pattern: " + signature,
		})
	}
	return out
}

func trustLabel(platform, kind, key string) string {
	switch kind {
	case "reddit_author":
		return fmt.Sprintf("Trust Reddit author u/%s", key)
	case "subreddit":
		return fmt.Sprintf("Trust subreddit r/%s", key)
	case "bluesky_cid":
		return fmt.Sprintf("Trust Bluesky CID %s", key)
	case "bluesky_handle":
		return fmt.Sprintf("Trust Bluesky handle %s", key)
	case "domain":
		return fmt.Sprintf("Trust domain %s", key)
	case "canonical_url":
		return fmt.Sprintf("Trust URL %s", key)
	default:
		return fmt.Sprintf("Trust %s %s", platform, key)
	}
}
