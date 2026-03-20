package lib

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/legolasljl/studyclaw/conf"
	"github.com/legolasljl/studyclaw/model"
)

type studyStep string

const (
	studyStepArticle    studyStep = "article"
	studyStepAudio      studyStep = "audio"
	studyStepDailyQuiz  studyStep = "daily_quiz"
)

func mediaScoreComplete(score Score) bool {
	video := score.Content["video"]
	videoTime := score.Content["video_time"]
	return video.MaxScore == 0 || (video.CurrentScore >= video.MaxScore && videoTime.CurrentScore >= videoTime.MaxScore)
}

func runAudioStudy(core2 *Core, u *model.User) {
	settings := loadStudySettings()
	score, err := getUserScoreWithRetry(u, settings.ScoreRetryTimes)
	if err != nil {
		logrus.Warningln(fmt.Sprintf("[音频学习] 学习前回查积分失败: %v", err))
		return
	}
	if mediaScoreComplete(score) {
		logrus.Infoln("[音频学习] 视听积分已完成，跳过音频流程")
		return
	}

	video := score.Content["video"]
	logrus.Infoln(fmt.Sprintf("[音频学习] 当前视听积分: %d/%d，开始执行音频流程", video.CurrentScore, video.MaxScore))
	core2.RadioStation(u)
}

func workflowSteps(model int) []studyStep {
	switch model {
	case 2:
		// 模式 2：文章 + 音頻 + 每日答題
		return []studyStep{studyStepArticle, studyStepAudio, studyStepDailyQuiz}
	default:
		// 模式 1（預設）：文章 + 音頻
		return []studyStep{studyStepArticle, studyStepAudio}
	}
}

func RunLearningWorkflow(core2 *Core, u *model.User) {
	for _, step := range workflowSteps(conf.GetConfig().Model) {
		switch step {
		case studyStepArticle:
			core2.LearnArticle(u)
		case studyStepAudio:
			runAudioStudy(core2, u)
		case studyStepDailyQuiz:
			logrus.Infoln("[每日答題] 開始執行每日答題流程")
			core2.RespondDaily(u, "daily")
		}
	}
}

func Study(core2 *Core, u *model.User) {
	defer func() {
		err := recover()
		if err != nil {
			logrus.Errorln("学习过程异常")
			logrus.Errorln(err)
		}
	}()
	startTime := time.Now()
	RunLearningWorkflow(core2, u)
	endTime := time.Now()
	score, err := GetUserScore(u.ToCookies())
	if err != nil {
		logrus.Errorln("获取成绩失败")
		logrus.Debugln(err.Error())
		return
	}

	message := FormatLearningCompletionMessage(u.Nick, endTime.Sub(startTime), score)
	core2.Push(u.PushId, "flush", message)
}
