package lib

import "testing"

func TestWorkflowStepsModelOneUsesArticleThenAudio(t *testing.T) {
	got := workflowSteps(1)
	want := []studyStep{studyStepArticle, studyStepAudio}
	assertWorkflowSteps(t, got, want)
}

func TestWorkflowStepsModelTwoUsesArticleAudioAndDailyQuiz(t *testing.T) {
	got := workflowSteps(2)
	want := []studyStep{studyStepArticle, studyStepAudio, studyStepDailyQuiz}
	assertWorkflowSteps(t, got, want)
}

func TestWorkflowStepsModelFourStillUsesArticleThenAudio(t *testing.T) {
	got := workflowSteps(4)
	want := []studyStep{studyStepArticle, studyStepAudio}
	assertWorkflowSteps(t, got, want)
}

func assertWorkflowSteps(t *testing.T, got []studyStep, want []studyStep) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("workflowSteps() len = %d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("workflowSteps()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
