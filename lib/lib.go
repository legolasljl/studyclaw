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
	studyStepArticle            studyStep = "article"
	studyStepAudio              studyStep = "audio"
	studyStepDailyQuiz          studyStep = "daily_quiz"
	dailyFullScoreTarget                  = 29
	scheduledStudyDurationLimit           = 35 * time.Minute
)

func maxInt(value int, floor int) int {
	if value < floor {
		return floor
	}
	return value
}

func formatScheduledStudyStartMessage(nick string, score *Score) string {
	if score == nil {
		return fmt.Sprintf("%s帳號 定時學習開始", nick)
	}
	programCurrent, programTarget := programTaskProgress(*score)
	return fmt.Sprintf(
		"%s帳號 定時學習開始\n程式可學三項進度：%d/%d\n今日總分：%d\n文章學習：%d/%d\n視頻學習：%d/%d\n每日答題：%d/%d\n",
		nick,
		programCurrent,
		programTarget,
		score.TodayScore,
		score.Content["article"].CurrentScore,
		score.Content["article"].MaxScore,
		score.Content["video"].CurrentScore,
		score.Content["video"].MaxScore,
		score.Content["daily"].CurrentScore,
		score.Content["daily"].MaxScore,
	)
}

func formatScheduledStudySkipMessage(nick string, score Score) string {
	programCurrent, programTarget := programTaskProgress(score)
	return fmt.Sprintf(
		"%s帳號 程式可學三項已滿分，跳過本輪定時學習\n程式可學三項進度：%d/%d\n今日總分：%d\n",
		nick,
		programCurrent,
		programTarget,
		score.TodayScore,
	)
}

func formatScheduledStudyCompletionMessage(nick string, duration time.Duration, score Score) string {
	message := FormatLearningCompletionMessage(nick, duration, score)
	programCurrent, programTarget := programTaskProgress(score)
	if programCurrent >= programTarget {
		message += fmt.Sprintf("程式可學三項：已滿 %d/%d\n", programCurrent, programTarget)
	} else {
		remaining := maxInt(programTarget-programCurrent, 0)
		message += fmt.Sprintf("程式可學三項：%d/%d，距離滿分還差 %d 分\n", programCurrent, programTarget, remaining)
	}
	if duration > scheduledStudyDurationLimit {
		message += fmt.Sprintf("單次學習用時：已超過 %.0f 分鐘上限\n", scheduledStudyDurationLimit.Minutes())
	} else {
		message += fmt.Sprintf("單次學習用時：未超過 %.0f 分鐘上限\n", scheduledStudyDurationLimit.Minutes())
	}
	return message
}

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
	initialScore, initialErr := GetUserScore(u.ToCookies())
	if initialErr != nil {
		logrus.Warningln(fmt.Sprintf("[學習流程] 開始前回查積分失敗: %v", initialErr))
		core2.Push(u.PushId, "flush", formatScheduledStudyStartMessage(u.Nick, nil))
	} else {
		core2.Push(u.PushId, "flush", formatScheduledStudyStartMessage(u.Nick, &initialScore))
		programCurrent, programTarget := programTaskProgress(initialScore)
		if programCurrent >= programTarget {
			core2.Push(u.PushId, "flush", formatScheduledStudySkipMessage(u.Nick, initialScore))
			return
		}
	}
	timedOut := false
	workflowDone := make(chan struct{}, 1)
	go func() {
		defer func() {
			workflowDone <- struct{}{}
		}()
		RunLearningWorkflow(core2, u)
	}()
	select {
	case <-workflowDone:
	case <-time.After(scheduledStudyDurationLimit):
		timedOut = true
		core2.Push(u.PushId, "flush", fmt.Sprintf("%s帳號 單次學習已達 %.0f 分鐘上限，本輪自動停止", u.Nick, scheduledStudyDurationLimit.Minutes()))
		core2.Quit()
	}
	endTime := time.Now()
	score, err := GetUserScore(u.ToCookies())
	if err != nil {
		logrus.Errorln("获取成绩失败")
		logrus.Debugln(err.Error())
		core2.Push(u.PushId, "flush", fmt.Sprintf("%s帳號 本輪學習流程已結束，但成績查詢失敗：%v", u.Nick, err))
		return
	}

	message := formatScheduledStudyCompletionMessage(u.Nick, endTime.Sub(startTime), score)
	if timedOut {
		message += fmt.Sprintf("本輪狀態：已按 %.0f 分鐘上限自動停止\n", scheduledStudyDurationLimit.Minutes())
	}
	core2.Push(u.PushId, "flush", message)
}
