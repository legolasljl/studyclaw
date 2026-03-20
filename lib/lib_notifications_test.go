package lib

import (
	"strings"
	"testing"
	"time"
)

func TestFormatScheduledStudySkipMessage(t *testing.T) {
	score := Score{
		TodayScore: 15,
		Content: map[string]Data{
			"article": {CurrentScore: 12, MaxScore: 12},
			"video":   {CurrentScore: 12, MaxScore: 12},
			"daily":   {CurrentScore: 5, MaxScore: 5},
		},
	}
	got := formatScheduledStudySkipMessage("Alice", score)
	for _, expected := range []string{
		"Alice帳號 程式可學三項已滿分",
		"程式可學三項進度：29/29",
		"今日總分：15",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("formatScheduledStudySkipMessage() missing %q in %q", expected, got)
		}
	}
}

func TestFormatScheduledStudyCompletionMessageIncludesFullScoreStatus(t *testing.T) {
	score := Score{
		TotalScore: 88,
		TodayScore: 43,
		Content: map[string]Data{
			"article": {CurrentScore: 12, MaxScore: 12},
			"video":   {CurrentScore: 12, MaxScore: 12},
			"daily":   {CurrentScore: 5, MaxScore: 5},
		},
	}
	got := formatScheduledStudyCompletionMessage("Alice", 20*time.Minute, score)
	for _, expected := range []string{
		"Alice帳號 已學習完成",
		"程式可學三項：已滿 29/29",
		"單次學習用時：未超過 35 分鐘上限",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("formatScheduledStudyCompletionMessage() missing %q in %q", expected, got)
		}
	}
}

func TestProgramTaskProgressIgnoresOtherTodayScoreSources(t *testing.T) {
	score := Score{
		TodayScore: 40,
		Content: map[string]Data{
			"article": {CurrentScore: 12, MaxScore: 12},
			"video":   {CurrentScore: 8, MaxScore: 12},
			"daily":   {CurrentScore: 0, MaxScore: 5},
			"login":   {CurrentScore: 1, MaxScore: 1},
			"special": {CurrentScore: 10, MaxScore: 10},
		},
	}
	current, target := programTaskProgress(score)
	if current != 20 || target != 29 {
		t.Fatalf("programTaskProgress() = %d/%d, want 20/29", current, target)
	}
}
