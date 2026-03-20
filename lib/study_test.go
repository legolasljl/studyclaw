package lib

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

type fakeStudyPage struct {
	gotoURL     string
	gotoOptions []playwright.PageGotoOptions
	gotoErr     error

	waitOptions []playwright.PageWaitForLoadStateOptions
	waitErr     error

	evaluateScripts []string
	evaluateResults []interface{}
	evaluateCall    int
	evaluateErr     error
	currentURL      string
}

func (f *fakeStudyPage) Goto(url string, options ...playwright.PageGotoOptions) (playwright.Response, error) {
	f.gotoURL = url
	f.gotoOptions = append(f.gotoOptions, options...)
	return nil, f.gotoErr
}

func (f *fakeStudyPage) WaitForLoadState(options ...playwright.PageWaitForLoadStateOptions) error {
	f.waitOptions = append(f.waitOptions, options...)
	return f.waitErr
}

func (f *fakeStudyPage) Evaluate(expression string, arg ...interface{}) (interface{}, error) {
	f.evaluateScripts = append(f.evaluateScripts, expression)
	if f.evaluateErr != nil {
		return nil, f.evaluateErr
	}
	if f.evaluateCall < len(f.evaluateResults) {
		result := f.evaluateResults[f.evaluateCall]
		f.evaluateCall++
		return result, nil
	}
	f.evaluateCall++
	return nil, nil
}

func (f *fakeStudyPage) URL() string {
	return f.currentURL
}

func TestNavigateStudyPageUsesStableLoadStrategy(t *testing.T) {
	page := &fakeStudyPage{currentURL: "https://www.xuexi.cn/article/1"}

	err := navigateStudyPage(page, "https://example.com/article", "https://example.com/ref")
	if err != nil {
		t.Fatalf("navigateStudyPage() error = %v", err)
	}

	if page.gotoURL != "https://example.com/article" {
		t.Fatalf("unexpected goto url: %s", page.gotoURL)
	}
	if len(page.gotoOptions) != 1 {
		t.Fatalf("expected 1 goto call, got %d", len(page.gotoOptions))
	}
	if page.gotoOptions[0].Referer == nil || *page.gotoOptions[0].Referer != "https://example.com/ref" {
		t.Fatalf("unexpected referer: %+v", page.gotoOptions[0].Referer)
	}
	if page.gotoOptions[0].WaitUntil == nil || *page.gotoOptions[0].WaitUntil != *playwright.WaitUntilStateLoad {
		t.Fatalf("unexpected goto waitUntil: %+v", page.gotoOptions[0].WaitUntil)
	}
	if page.gotoOptions[0].Timeout == nil || *page.gotoOptions[0].Timeout != studyNavigationTimeoutMs {
		t.Fatalf("unexpected goto timeout: %+v", page.gotoOptions[0].Timeout)
	}
	if len(page.waitOptions) != 1 {
		t.Fatalf("expected 1 waitForLoadState call, got %d", len(page.waitOptions))
	}
	if page.waitOptions[0].State == nil || *page.waitOptions[0].State != *playwright.LoadStateLoad {
		t.Fatalf("unexpected load state: %+v", page.waitOptions[0].State)
	}
	if page.waitOptions[0].Timeout == nil || *page.waitOptions[0].Timeout != studyLoadStateTimeoutMs {
		t.Fatalf("unexpected wait timeout: %+v", page.waitOptions[0].Timeout)
	}
	if len(page.evaluateScripts) != 1 {
		t.Fatalf("expected initial scroll reset, got %d evaluate calls", len(page.evaluateScripts))
	}
	if !strings.Contains(page.evaluateScripts[0], "window.scrollTo(0, 0)") {
		t.Fatalf("expected reset scroll script, got %s", page.evaluateScripts[0])
	}
}

func TestNavigateStudyPageStopsOnGotoError(t *testing.T) {
	page := &fakeStudyPage{
		gotoErr:    errors.New("goto timeout"),
		currentURL: "https://www.xuexi.cn/article/1",
	}

	err := navigateStudyPage(page, "https://example.com/article", "https://example.com/ref")
	if err == nil {
		t.Fatal("expected navigateStudyPage() to return error")
	}
	if len(page.waitOptions) != 0 {
		t.Fatalf("expected no waitForLoadState calls, got %d", len(page.waitOptions))
	}
	if len(page.evaluateScripts) != 0 {
		t.Fatalf("expected no evaluate calls, got %d", len(page.evaluateScripts))
	}
}

func TestScrollStudyPageBuildsRobustScrollScript(t *testing.T) {
	page := &fakeStudyPage{}

	err := scrollStudyPage(page, 17)
	if err != nil {
		t.Fatalf("scrollStudyPage() error = %v", err)
	}
	if len(page.evaluateScripts) != 1 {
		t.Fatalf("expected 1 evaluate call, got %d", len(page.evaluateScripts))
	}
	if !strings.Contains(page.evaluateScripts[0], "const currentStep = 17;") {
		t.Fatalf("expected step value in script, got %s", page.evaluateScripts[0])
	}
	if !strings.Contains(page.evaluateScripts[0], "document.scrollingElement || document.documentElement || document.body") {
		t.Fatalf("expected fallback scroll root in script, got %s", page.evaluateScripts[0])
	}
}

func TestNavigateStudyPageDetectsLoginRedirect(t *testing.T) {
	page := &fakeStudyPage{currentURL: "https://login.xuexi.cn/somewhere"}

	err := navigateStudyPage(page, "https://example.com/article", "https://example.com/ref")
	if err == nil {
		t.Fatal("expected login redirect error")
	}
	if !strings.Contains(err.Error(), "cookie 可能已失效") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilterLinksByRecentDays(t *testing.T) {
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.Local)
	links := []Link{
		{Title: "newest", PublishTime: "2026-03-10 09:00:00"},
		{Title: "old", PublishTime: "2025-01-01 09:00:00"},
		{Title: "invalid", PublishTime: "bad-value"},
		{Title: "recent", PublishTime: "2025-12-20 09:00:00"},
	}

	filtered := filterLinksByRecentDays(links, 180, now)
	if len(filtered.Links) != 2 {
		t.Fatalf("expected 2 recent links, got %d", len(filtered.Links))
	}
	if filtered.OldCount != 1 {
		t.Fatalf("expected 1 old link, got %d", filtered.OldCount)
	}
	if filtered.InvalidCount != 1 {
		t.Fatalf("expected 1 invalid link, got %d", filtered.InvalidCount)
	}
	if filtered.Links[0].Title != "newest" {
		t.Fatalf("expected newest link first, got %s", filtered.Links[0].Title)
	}
}

func TestShouldScrollAtStep(t *testing.T) {
	if shouldScrollAtStep(0, 5) {
		t.Fatal("step 0 should not scroll")
	}
	if shouldScrollAtStep(4, 5) {
		t.Fatal("step 4 should not scroll with interval 5")
	}
	if !shouldScrollAtStep(5, 5) {
		t.Fatal("step 5 should scroll with interval 5")
	}
}

func TestAttemptStudyMediaPlaybackParsesEvaluateResult(t *testing.T) {
	page := &fakeStudyPage{
		evaluateResults: []interface{}{
			map[string]interface{}{
				"status":      "playing",
				"currentTime": float64(3),
				"readyState":  float64(4),
				"clicked":     float64(1),
			},
		},
	}

	state, err := attemptStudyMediaPlayback(page, "video")
	if err != nil {
		t.Fatalf("attemptStudyMediaPlayback() error = %v", err)
	}
	if !state.IsPlaying() {
		t.Fatal("expected media state to be playing")
	}
	if len(page.evaluateScripts) != 1 {
		t.Fatalf("expected 1 evaluate call, got %d", len(page.evaluateScripts))
	}
	if !strings.Contains(page.evaluateScripts[0], `const kind = "video";`) {
		t.Fatalf("expected video playback script, got %s", page.evaluateScripts[0])
	}
}

func TestWaitForStudyMediaPlaybackRetriesUntilPlaying(t *testing.T) {
	page := &fakeStudyPage{
		evaluateResults: []interface{}{
			map[string]interface{}{
				"status":      "paused",
				"currentTime": float64(0),
				"readyState":  float64(2),
				"clicked":     float64(1),
			},
			map[string]interface{}{
				"status":      "playing",
				"currentTime": float64(1),
				"readyState":  float64(4),
				"clicked":     float64(1),
			},
		},
	}

	err := waitForStudyMediaPlayback(page, "video", 2, 0)
	if err != nil {
		t.Fatalf("waitForStudyMediaPlayback() error = %v", err)
	}
	if len(page.evaluateScripts) != 2 {
		t.Fatalf("expected 2 evaluate calls, got %d", len(page.evaluateScripts))
	}
}
