package services

import "testing"

func TestValidPlatformsIncludesHackerNews(t *testing.T) {
	if !validPlatforms["hackernews"] {
		t.Fatal("validPlatforms must include hackernews")
	}
}
