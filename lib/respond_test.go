package lib

import (
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
)

func TestRandomDurationBetweenWithinRange(t *testing.T) {
	for i := 0; i < 20; i++ {
		d := randomDurationBetween(500, 900)
		if d < 500000000 || d > 900000000 {
			t.Fatalf("duration %v out of range", d)
		}
	}
}

func TestNormalizeAnswerButtonText(t *testing.T) {
	got := normalizeAnswerButtonText(" 提 交\n答 案 ")
	if got != "提交答案" {
		t.Fatalf("normalizeAnswerButtonText() = %q, want %q", got, "提交答案")
	}
}

func TestContainsAnswerSliderSpecificText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "specific slider instruction",
			text: "请按住滑块，拖动到最右边完成验证",
			want: true,
		},
		{
			name: "specific slide verification phrase",
			text: "系统提示：向右滑动验证后继续",
			want: true,
		},
		{
			name: "generic slider word should not match",
			text: "本题材料提到滑块轴承的机械结构",
			want: false,
		},
		{
			name: "generic verification text should not match",
			text: "请完成验证后继续答题",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsAnswerSliderSpecificText(tt.text); got != tt.want {
				t.Fatalf("containsAnswerSliderSpecificText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsAnswerSliderLooseText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "generic slider text",
			text: "系统即将弹出滑块，请拖动验证",
			want: true,
		},
		{
			name: "non slider text",
			text: "请完成阅读后继续答题",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsAnswerSliderLooseText(tt.text); got != tt.want {
				t.Fatalf("containsAnswerSliderLooseText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsAnswerFlowBlockedText(t *testing.T) {
	text := "请不要中途开启新的答题流程，不支持多端同时作答"
	if !containsAnswerFlowBlockedText(text) {
		t.Fatalf("containsAnswerFlowBlockedText() = false, want true")
	}
}

func TestContainsAnswerFlowBlockedTextIgnoresGenericFlowMention(t *testing.T) {
	text := "正在进入答题流程，请稍候"
	if containsAnswerFlowBlockedText(text) {
		t.Fatalf("containsAnswerFlowBlockedText() = true, want false")
	}
}

func TestContainsAnswerQuestionContextText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "question page summary",
			text: "单选题 5. 根据提示选择正确答案 上一题 确定",
			want: true,
		},
		{
			name: "fill blank page",
			text: "填空题 请根据提示作答 查看提示 提交",
			want: true,
		},
		{
			name: "result page",
			text: "本次答对题目数 5 正确率 100% 再来一组",
			want: false,
		},
		{
			name: "generic question help text",
			text: "答题前请先阅读题目提示并进入答题流程",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsAnswerQuestionContextText(tt.text); got != tt.want {
				t.Fatalf("containsAnswerQuestionContextText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAnswerCompletionText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "result summary",
			text: "本次答对题目数 5 正确率 100% 答错数 0 用时 00:00:32 再来一组",
			want: true,
		},
		{
			name: "question page",
			text: "单选题 5. 根据提示选择正确答案 上一题 确定",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAnswerCompletionText(tt.text); got != tt.want {
				t.Fatalf("isAnswerCompletionText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeSliderPosition(t *testing.T) {
	startX, startY, endX, endY, err := decodeSliderPosition(map[string]interface{}{
		"ok":     true,
		"startX": float64(10),
		"startY": float64(20),
		"endX":   float64(100),
		"endY":   float64(22),
	})
	if err != nil {
		t.Fatalf("decodeSliderPosition() error = %v", err)
	}
	if startX != 10 || startY != 20 || endX != 100 || endY != 22 {
		t.Fatalf("unexpected slider positions: %v %v %v %v", startX, startY, endX, endY)
	}
}

func TestBuildTipCandidates(t *testing.T) {
	got := buildTipCandidates([]string{"农村地区，革命老区、民族地区"})
	want := []string{"农村地区，革命老区、民族地区", "农村地区", "革命老区", "民族地区"}
	if len(got) != len(want) {
		t.Fatalf("buildTipCandidates() len = %d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("buildTipCandidates()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPickSelectableAnswersMatchesTips(t *testing.T) {
	options := []string{"A. 灭火器材", "B. 警示标志", "C. 防寒服"}
	got := pickSelectableAnswers(options, []string{"灭火器材"})
	if len(got) != 1 || got[0] != "A. 灭火器材" {
		t.Fatalf("pickSelectableAnswers() = %v, want [A. 灭火器材]", got)
	}
}

func TestPickSelectableAnswersFallsBackToExpectedCount(t *testing.T) {
	options := []string{"A. 公益性", "B. 基本性", "C. 均等性", "D. 便利性"}
	got := pickSelectableAnswers(options, []string{"公益性 基本性 均等性"})
	want := []string{"A. 公益性", "B. 基本性", "C. 均等性"}
	if len(got) != len(want) {
		t.Fatalf("pickSelectableAnswers() len = %d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pickSelectableAnswers()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPickSelectableAnswersAvoidsNegationFalsePositive(t *testing.T) {
	options := []string{"A. 具有", "B. 不具有"}
	got := pickSelectableAnswers(options, []string{"具有"})
	if len(got) != 1 || got[0] != "A. 具有" {
		t.Fatalf("pickSelectableAnswers() = %v, want [A. 具有]", got)
	}
}

func TestPickSelectableAnswersUsesJoinedTips(t *testing.T) {
	options := []string{"A. 设立", "B. 变更", "C. 终止"}
	got := pickSelectableAnswers(options, []string{"设立 变更 终 止"})
	want := []string{"A. 设立", "B. 变更", "C. 终止"}
	if len(got) != len(want) {
		t.Fatalf("pickSelectableAnswers() len = %d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pickSelectableAnswers()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSelectSingleChoiceAnswersHandlesReversePrompt(t *testing.T) {
	question := "关于烟花爆竹的选购，以下说法错误的是（）。"
	options := []string{
		"A. 切勿在无证摊点、流动商贩及网络等非正规渠道购买",
		"B. 选择标注有“个人燃放类”字样的产品，非个人燃放类产品不要购买",
		"C. 可以购买非绿色安全引火线的烟花爆竹产品",
	}
	got := selectSingleChoiceAnswers(question, options, []string{"不要购买非绿色安全引火线的烟花爆竹产品"})
	if len(got) != 1 || got[0] != options[2] {
		t.Fatalf("selectSingleChoiceAnswers() = %v, want [%s]", got, options[2])
	}
}

func TestSelectSingleChoiceAnswersHandlesReverseNegationPair(t *testing.T) {
	question := "以下说法错误的是（）。"
	options := []string{"A. 构成", "B. 不构成"}
	got := selectSingleChoiceAnswers(question, options, []string{"构成"})
	if len(got) != 1 || got[0] != options[1] {
		t.Fatalf("selectSingleChoiceAnswers() = %v, want [%s]", got, options[1])
	}
}

func TestSelectSingleChoiceAnswersMatchesExactOption(t *testing.T) {
	question := "在中华人民共和国境内销售无形资产，无须缴纳增值税。（）"
	options := []string{"A. 正确", "B. 错误"}
	got := selectSingleChoiceAnswers(question, options, []string{"错误"})
	if len(got) != 1 || got[0] != options[1] {
		t.Fatalf("selectSingleChoiceAnswers() = %v, want [%s]", got, options[1])
	}
}

func TestSegmentAnswerByOptionsUsesVisibleCandidates(t *testing.T) {
	got := segmentAnswerByOptions("正确投资理念", []string{"正", "确", "投", "资", "理", "念"}, 6)
	want := []string{"正", "确", "投", "资", "理", "念"}
	if len(got) != len(want) {
		t.Fatalf("segmentAnswerByOptions() len = %d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("segmentAnswerByOptions()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildClickBlankAnswersFallsBackToRunes(t *testing.T) {
	got := buildClickBlankAnswers([]string{"信用能源"}, nil, 4)
	want := []string{"信", "用", "能", "源"}
	if len(got) != len(want) {
		t.Fatalf("buildClickBlankAnswers() len = %d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("buildClickBlankAnswers()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildClickBlankAnswersPrefersOptionSegments(t *testing.T) {
	got := buildClickBlankAnswers([]string{"基本形成"}, []string{"基本", "形成"}, 4)
	want := []string{"基本", "形成"}
	if len(got) != len(want) {
		t.Fatalf("buildClickBlankAnswers() len = %d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("buildClickBlankAnswers()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildClickBlankAnswersPrefersWholeOptionWithSameRunes(t *testing.T) {
	// 當候選詞中有亂序的整詞選項時，應該返回提示的正確順序（拆分為單字）
	// 因為 blankCount = 4，需要 4 個單獨的答案
	got := buildClickBlankAnswers([]string{"应用场景"}, []string{"场景应用", "场", "景", "应", "用"}, 4)
	want := []string{"应", "用", "场", "景"} // 使用提示的正確順序，而非亂序的選項
	if len(got) != len(want) {
		t.Fatalf("buildClickBlankAnswers() len = %d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("buildClickBlankAnswers()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildClickBlankAnswersCorrectOrderWithScrambledOption(t *testing.T) {
	// 測試真實場景：提示「国药准字」，候選詞包含亂序的「字准药国」
	// 應該返回提示的正確順序，而非亂序選項
	got := buildClickBlankAnswers([]string{"国药准字"}, []string{"字准药国", "字", "准", "药", "国"}, 4)
	want := []string{"国", "药", "准", "字"}
	if len(got) != len(want) {
		t.Fatalf("buildClickBlankAnswers() len = %d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("buildClickBlankAnswers()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildClickBlankAnswersPrefersSegmentedWordsOverReorderedWholeOption(t *testing.T) {
	got := buildClickBlankAnswers([]string{"农村包围城市"}, []string{"城市包围农村", "城市", "包围", "农村"}, 6)
	want := []string{"农村", "包围", "城市"}
	if len(got) != len(want) {
		t.Fatalf("buildClickBlankAnswers() len = %d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("buildClickBlankAnswers()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExpandFillBlankAnswersPadsFallbacks(t *testing.T) {
	inputs := []playwright.ElementHandle{nil, nil}
	got := expandFillBlankAnswers([]string{"溯"}, inputs)
	if len(got) != 2 {
		t.Fatalf("expandFillBlankAnswers() len = %d, want 2, got=%v", len(got), got)
	}
	if got[0] != "溯" {
		t.Fatalf("expandFillBlankAnswers()[0] = %q, want %q", got[0], "溯")
	}
	if strings.TrimSpace(got[1]) == "" {
		t.Fatalf("expandFillBlankAnswers()[1] should not be empty, got=%q", got[1])
	}
}
