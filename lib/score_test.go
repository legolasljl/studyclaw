package lib

import (
	"strings"
	"testing"
	"time"
)

func TestParseTaskProgressContentAggregatesByTitle(t *testing.T) {
	resp := []byte(`{
		"data": {
			"taskProgress": [
				{"title": "文章选读", "currentScore": 6, "dayMaxScore": 6},
				{"title": "文章学习时长", "currentScore": 6, "dayMaxScore": 6},
				{"title": "视听学习", "currentScore": 4, "dayMaxScore": 6},
				{"title": "视听学习时长", "currentScore": 2, "dayMaxScore": 6},
				{"title": "每日登录", "currentScore": 1, "dayMaxScore": 1},
				{"title": "每日答题", "currentScore": 3, "dayMaxScore": 5},
				{"title": "每周答题", "currentScore": 2, "dayMaxScore": 5},
				{"title": "专项答题", "currentScore": 0, "dayMaxScore": 10}
			]
		}
	}`)

	content := parseTaskProgressContent(resp)
	if got := content["article"]; got.CurrentScore != 12 || got.MaxScore != 12 {
		t.Fatalf("article = %+v, want 12/12", got)
	}
	if got := content["video"]; got.CurrentScore != 6 || got.MaxScore != 12 {
		t.Fatalf("video = %+v, want 6/12", got)
	}
	if got := content["video_time"]; got.CurrentScore != 2 || got.MaxScore != 6 {
		t.Fatalf("video_time = %+v, want 2/6", got)
	}
	if got := content["login"]; got.CurrentScore != 1 || got.MaxScore != 1 {
		t.Fatalf("login = %+v, want 1/1", got)
	}
	if got := content["daily"]; got.CurrentScore != 3 || got.MaxScore != 5 {
		t.Fatalf("daily = %+v, want 3/5", got)
	}
	if got := content["weekly"]; got.CurrentScore != 2 || got.MaxScore != 5 {
		t.Fatalf("weekly = %+v, want 2/5", got)
	}
	if got := content["special"]; got.CurrentScore != 0 || got.MaxScore != 10 {
		t.Fatalf("special = %+v, want 0/10", got)
	}
}

func TestParseTaskProgressContentFallsBackToLegacyOrder(t *testing.T) {
	resp := []byte(`{
		"data": {
			"taskProgress": [
				{"currentScore": 12, "dayMaxScore": 12},
				{"currentScore": 6, "dayMaxScore": 12},
				{"currentScore": 1, "dayMaxScore": 1},
				{"currentScore": 3, "dayMaxScore": 5},
				{"currentScore": 0, "dayMaxScore": 10}
			]
		}
	}`)

	content := parseTaskProgressContent(resp)
	if got := content["article"]; got.CurrentScore != 12 || got.MaxScore != 12 {
		t.Fatalf("article = %+v, want 12/12", got)
	}
	if got := content["video"]; got.CurrentScore != 6 || got.MaxScore != 12 {
		t.Fatalf("video = %+v, want 6/12", got)
	}
	if got := content["video_time"]; got.CurrentScore != 6 || got.MaxScore != 12 {
		t.Fatalf("video_time = %+v, want 6/12", got)
	}
	if got := content["login"]; got.CurrentScore != 1 || got.MaxScore != 1 {
		t.Fatalf("login = %+v, want 1/1", got)
	}
	if got := content["daily"]; got.CurrentScore != 3 || got.MaxScore != 5 {
		t.Fatalf("daily = %+v, want 3/5", got)
	}
	if got := content["special"]; got.CurrentScore != 0 || got.MaxScore != 10 {
		t.Fatalf("special = %+v, want 0/10", got)
	}
}

func TestFormatScoreOnlyIncludesArticleAndVideo(t *testing.T) {
	score := Score{
		TotalScore: 42,
		TodayScore: 14,
		Content: map[string]Data{
			"article": {CurrentScore: 12, MaxScore: 12},
			"video":   {CurrentScore: 6, MaxScore: 12},
			"login":   {CurrentScore: 1, MaxScore: 1},
			"daily":   {CurrentScore: 5, MaxScore: 5},
			"special": {CurrentScore: 0, MaxScore: 10},
		},
	}

	got := FormatScore(score)
	if !strings.Contains(got, "文章學習：12/12") || !strings.Contains(got, "視頻學習：6/12") {
		t.Fatalf("FormatScore() missing learning lines: %q", got)
	}
	for _, removed := range []string{"登录：", "每日答题：", "专项答题：", "登入：", "每日答題：", "專項答題："} {
		if strings.Contains(got, removed) {
			t.Fatalf("FormatScore() should not include %q: %q", removed, got)
		}
	}
}

func TestFormatLearningCompletionMessageIncludesDuration(t *testing.T) {
	score := Score{
		TotalScore: 52,
		TodayScore: 18,
		Content: map[string]Data{
			"article": {CurrentScore: 12, MaxScore: 12},
			"video":   {CurrentScore: 9, MaxScore: 12},
		},
	}

	got := FormatLearningCompletionMessage("Dachi", 95*time.Second, score)
	for _, expected := range []string{
		"Dachi帳號 已學習完成",
		"當前學習總積分：52",
		"今日得分：18",
		"本次用時：1.6分鐘",
		"文章學習：12/12",
		"視頻學習：9/12",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("FormatLearningCompletionMessage() missing %q in %q", expected, got)
		}
	}
}
