package lib

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/legolasljl/studyclaw/utils"
)

type Score struct {
	TotalScore int             `json:"total_score"`
	TodayScore int             `json:"today_score"`
	Content    map[string]Data `json:"content"`
}

type Data struct {
	CurrentScore int `json:"current_score"`
	MaxScore     int `json:"max_score"`
}

func addScoreData(dst *Data, current int, max int) {
	dst.CurrentScore += current
	dst.MaxScore += max
}

func normalizeTaskTitle(title string) string {
	replacer := strings.NewReplacer(
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
		"　", "",
		"（", "(",
		"）", ")",
	)
	return replacer.Replace(strings.TrimSpace(title))
}

func extractTaskTitle(item gjson.Result) string {
	for _, key := range []string{"title", "name", "taskName", "desc"} {
		if title := strings.TrimSpace(item.Get(key).String()); title != "" {
			return title
		}
	}
	return ""
}

func isDailyTaskTitle(title string) bool {
	return strings.Contains(title, "每日答题")
}

func isWeeklyTaskTitle(title string) bool {
	return strings.Contains(title, "每周答题")
}

func isSpecialTaskTitle(title string) bool {
	return strings.Contains(title, "专项答题")
}

func isLoginTaskTitle(title string) bool {
	return strings.Contains(title, "登录")
}

func isArticleTaskTitle(title string) bool {
	if isDailyTaskTitle(title) || isWeeklyTaskTitle(title) || isSpecialTaskTitle(title) || isLoginTaskTitle(title) {
		return false
	}
	if strings.Contains(title, "视听") || strings.Contains(title, "试听") || strings.Contains(title, "视频") || strings.Contains(title, "音频") {
		return false
	}
	return strings.Contains(title, "文章") || strings.Contains(title, "选读") || strings.Contains(title, "阅读")
}

func isVideoTaskTitle(title string) bool {
	return strings.Contains(title, "视听") || strings.Contains(title, "试听") || strings.Contains(title, "视频") || strings.Contains(title, "音频") || strings.Contains(title, "电台") || strings.Contains(title, "广播")
}

func isVideoTimeTaskTitle(title string) bool {
	if !isVideoTaskTitle(title) {
		return false
	}
	return strings.Contains(title, "时长") || strings.Contains(title, "累计") || strings.Contains(title, "分钟")
}

func parseTaskProgressContent(resp []byte) map[string]Data {
	datas := gjson.GetBytes(resp, "data.taskProgress").Array()
	content := map[string]Data{
		"article":    {},
		"video":      {},
		"video_time": {},
		"login":      {},
		"daily":      {},
		"weekly":     {},
		"special":    {},
	}
	if len(datas) == 0 {
		return content
	}

	videoTotal := Data{}
	videoTime := Data{}
	matched := 0

	for _, item := range datas {
		current := int(item.Get("currentScore").Int())
		max := int(item.Get("dayMaxScore").Int())
		title := normalizeTaskTitle(extractTaskTitle(item))

		switch {
		case title == "":
			continue
		case isArticleTaskTitle(title):
			data := content["article"]
			addScoreData(&data, current, max)
			content["article"] = data
			matched++
		case isVideoTaskTitle(title):
			addScoreData(&videoTotal, current, max)
			if isVideoTimeTaskTitle(title) {
				addScoreData(&videoTime, current, max)
			}
			matched++
		case isLoginTaskTitle(title):
			data := content["login"]
			addScoreData(&data, current, max)
			content["login"] = data
			matched++
		case isDailyTaskTitle(title):
			data := content["daily"]
			addScoreData(&data, current, max)
			content["daily"] = data
			matched++
		case isWeeklyTaskTitle(title):
			data := content["weekly"]
			addScoreData(&data, current, max)
			content["weekly"] = data
			matched++
		case isSpecialTaskTitle(title):
			data := content["special"]
			addScoreData(&data, current, max)
			content["special"] = data
			matched++
		}
	}

	if matched == 0 && len(datas) >= 4 {
		content["article"] = Data{
			CurrentScore: int(datas[0].Get("currentScore").Int()),
			MaxScore:     int(datas[0].Get("dayMaxScore").Int()),
		}
		content["video"] = Data{
			CurrentScore: int(datas[1].Get("currentScore").Int()),
			MaxScore:     int(datas[1].Get("dayMaxScore").Int()),
		}
		content["video_time"] = content["video"]
		content["login"] = Data{
			CurrentScore: int(datas[2].Get("currentScore").Int()),
			MaxScore:     int(datas[2].Get("dayMaxScore").Int()),
		}
		content["daily"] = Data{
			CurrentScore: int(datas[3].Get("currentScore").Int()),
			MaxScore:     int(datas[3].Get("dayMaxScore").Int()),
		}
		if len(datas) > 4 {
			content["special"] = Data{
				CurrentScore: int(datas[4].Get("currentScore").Int()),
				MaxScore:     int(datas[4].Get("dayMaxScore").Int()),
			}
		}
		return content
	}

	if videoTotal.MaxScore > 0 {
		content["video"] = videoTotal
	}
	if videoTime.MaxScore > 0 {
		content["video_time"] = videoTime
	} else if videoTotal.MaxScore > 0 {
		// 老接口会把视听次数和时长合并成一个任务，复用聚合值兼容现有流程。
		content["video_time"] = videoTotal
	}

	return content
}

// 获取用户总分
func GetUserScore(cookies []*http.Cookie) (Score, error) {
	var score Score
	var resp []byte

	header := map[string]string{
		"Cache-Control": "no-cache",
	}

	client := utils.GetClient()
	response, err := client.R().SetCookies(cookies...).SetHeaders(header).Get(userTotalscoreUrl)
	if err != nil {
		log.Errorln("获取用户总分错误" + err.Error())
		return Score{}, err
	}
	resp = response.Bytes()
	score.TotalScore = int(gjson.GetBytes(resp, "data.score").Int())

	response, err = client.R().SetCookies(cookies...).SetHeaders(header).Get(userTodaytotalscoreUrl)
	if err != nil {
		log.Errorln("获取用户今日得分错误" + err.Error())
		return Score{}, err
	}
	resp = response.Bytes()
	score.TodayScore = int(gjson.GetBytes(resp, "data.score").Int())

	response, err = client.R().SetCookies(cookies...).SetHeaders(header).Get(userRatescoreUrl)
	if err != nil {
		log.Errorln("获取用户详情得分错误" + err.Error())
		return Score{}, err
	}
	resp = response.Bytes()
	score.Content = parseTaskProgressContent(resp)

	return score, err
}

func formatLearningScoreLines(score Score) string {
	return fmt.Sprintf(
		"當前學習總積分：%d\n今日得分：%d\n文章學習：%d/%d\n視頻學習：%d/%d\n",
		score.TotalScore,
		score.TodayScore,
		score.Content["article"].CurrentScore,
		score.Content["article"].MaxScore,
		score.Content["video"].CurrentScore,
		score.Content["video"].MaxScore,
	)
}

// 输出总分
func PrintScore(score Score) string {
	result := formatLearningScoreLines(score)
	log.Infoln(result)
	return result
}

// 格式化总分
func FormatScore(score Score) string {
	return formatLearningScoreLines(score)
}

// 格式化短格式总分
func FormatScoreShort(score Score) string {
	return formatLearningScoreLines(score)
}

func FormatLearningCompletionMessage(nick string, duration time.Duration, score Score) string {
	return fmt.Sprintf(
		"%s帳號 已學習完成\n當前學習總積分：%d\n今日得分：%d\n本次用時：%.1f分鐘\n文章學習：%d/%d\n視頻學習：%d/%d\n",
		nick,
		score.TotalScore,
		score.TodayScore,
		duration.Minutes(),
		score.Content["article"].CurrentScore,
		score.Content["article"].MaxScore,
		score.Content["video"].CurrentScore,
		score.Content["video"].MaxScore,
	)
}
