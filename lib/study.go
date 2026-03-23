package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	log "github.com/sirupsen/logrus"

	"github.com/legolasljl/studyclaw/conf"
	"github.com/legolasljl/studyclaw/model"
	"github.com/legolasljl/studyclaw/utils"
)

var (
	studyNavigationTimeoutMs = 30000.0
	studyLoadStateTimeoutMs  = 15000.0
	studyMediaStartAttempts  = 12
	studyMediaStartInterval  = time.Second

	article_url_list = []string{
		"https://www.xuexi.cn/lgdata/35il6fpn0ohq.json",
		"https://www.xuexi.cn/lgdata/45a3hac2bf1j.json",
		"https://www.xuexi.cn/lgdata/1ajhkle8l72.json",
		"https://www.xuexi.cn/lgdata/1ahjpjgb4n3.json",
		"https://www.xuexi.cn/lgdata/1je1objnh73.json",
		"https://www.xuexi.cn/lgdata/1kvrj9vvv73.json",
		"https://www.xuexi.cn/lgdata/17qonfb74n3.json",
		"https://www.xuexi.cn/lgdata/1i30sdhg0n3.json"}

	video_url_list = []string{
		"https://www.xuexi.cn/lgdata/3j2u3cttsii9.json",
		"https://www.xuexi.cn/lgdata/1novbsbi47k.json",
		"https://www.xuexi.cn/lgdata/31c9ca1tgfqb.json",
		"https://www.xuexi.cn/lgdata/1oajo2vt47l.json",
		"https://www.xuexi.cn/lgdata/18rkaul9h7l.json",
		"https://www.xuexi.cn/lgdata/2qfjjjrprmdh.json",
		"https://www.xuexi.cn/lgdata/3o3ufqgl8rsn.json",
		"https://www.xuexi.cn/lgdata/525pi8vcj24p.json",
		"https://www.xuexi.cn/lgdata/1742g60067k.json"}

	yp_url_list = []string{
		"https://www.xuexi.cn/lgdata/1ode6kjlu7m.json",
		"https://www.xuexi.cn/lgdata/1ggb81u8f7m.json",
		"https://www.xuexi.cn/lgdata/139993ri8nm.json",
		"https://www.xuexi.cn/lgdata/u07dubuq7m.json",
		"https://www.xuexi.cn/lgdata/spisr390nm.json",
		"https://www.xuexi.cn/lgdata/1elt18mm57m.json"}
)

type studyPage interface {
	Goto(url string, options ...playwright.PageGotoOptions) (playwright.Response, error)
	WaitForLoadState(options ...playwright.PageWaitForLoadStateOptions) error
	Evaluate(expression string, arg ...interface{}) (interface{}, error)
	URL() string
}

type studyMediaEvalTarget interface {
	Evaluate(expression string, arg ...interface{}) (interface{}, error)
	URL() string
}

type studyMediaState struct {
	Status      string
	CurrentTime float64
	Duration    float64
	ReadyState  int
	Clicked     int
	MediaCount  int
	RootCount   int
	TargetURL   string
}

func (s studyMediaState) IsPlaying() bool {
	return s.Status == "playing" || s.CurrentTime > 0 || (s.ReadyState >= 3 && s.Clicked > 0)
}

type studySettings struct {
	RecentDays            int
	FallbackToAll         bool
	NavigationRetryTimes  int
	RetryBackoffSeconds   int
	OperationIntervalSecs int
	ScoreRetryTimes       int
	ArticleDurationSecs   int
	ArticleDurationJitter int
	VideoDurationSecs     int
	VideoDurationJitter   int
	AudioDurationSecs     int
	AudioDurationJitter   int
	// 新增：登入失效相關配置
	AuthFailureThreshold   int
	FailureCooldownMinutes int
	MaxSliderFailures      int
	FastFailEnabled        bool
}

type filteredStudyLinks struct {
	Links        []Link
	OldCount     int
	InvalidCount int
}

const studyPublishTimeLayout = "2006-01-02 15:04:05"

func buildStudyScrollScript(step int) string {
	return fmt.Sprintf(`() => {
		const currentStep = %d;
		const root = document.scrollingElement || document.documentElement || document.body;
		if (!root) {
			return -1;
		}
		const totalHeight = Math.max(
			root.scrollHeight || 0,
			document.body ? document.body.scrollHeight : 0,
			document.documentElement ? document.documentElement.scrollHeight : 0,
			window.innerHeight || 0
		);
		const maxTop = Math.max(totalHeight - (window.innerHeight || 0), 0);
		const baseTop = Math.min(maxTop, Math.floor(totalHeight / 120 * currentStep));
		const jitter = Math.floor(Math.random() * 61) + 20;
		const sign = Math.random() < 0.5 ? -1 : 1;
		const targetTop = Math.max(0, Math.min(maxTop, baseTop + sign * jitter));
		if (typeof window.scrollTo === "function") {
			window.scrollTo(0, targetTop);
		}
		root.scrollTop = targetTop;
		if (document.body) {
			document.body.scrollTop = targetTop;
		}
		if (document.documentElement) {
			document.documentElement.scrollTop = targetTop;
		}
		return targetTop;
	}`, step)
}

func resetStudyScroll(page studyPage) error {
	_, err := page.Evaluate(`() => {
		const root = document.scrollingElement || document.documentElement || document.body;
		if (!root) {
			return -1;
		}
		if (typeof window.scrollTo === "function") {
			window.scrollTo(0, 0);
		}
		root.scrollTop = 0;
		if (document.body) {
			document.body.scrollTop = 0;
		}
		if (document.documentElement) {
			document.documentElement.scrollTop = 0;
		}
		return 0;
	}`)
	return err
}

func navigateStudyPage(page studyPage, url string, referer string) error {
	resp, err := page.Goto(url, playwright.PageGotoOptions{
		Referer:   playwright.String(referer),
		Timeout:   playwright.Float(studyNavigationTimeoutMs),
		WaitUntil: playwright.WaitUntilStateLoad,
	})
	if err != nil {
		return fmt.Errorf("[导航] 页面打开失败: %w", err)
	}
	if resp != nil && resp.Status() >= 400 {
		return fmt.Errorf("[导航] 页面响应异常 status=%d url=%s", resp.Status(), resp.URL())
	}

	err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State:   playwright.LoadStateLoad,
		Timeout: playwright.Float(studyLoadStateTimeoutMs),
	})
	if err != nil {
		return fmt.Errorf("[导航] 页面加载未完成: %w", err)
	}
	currentURL := page.URL()
	if strings.Contains(currentURL, "login") || strings.Contains(currentURL, "secure_check") {
		return fmt.Errorf("[登录] 页面跳转到了登录流程，cookie 可能已失效: %s", currentURL)
	}

	return resetStudyScroll(page)
}

func scrollStudyPage(page studyPage, step int) error {
	_, err := page.Evaluate(buildStudyScrollScript(step))
	return err
}

func buildStudyMediaPlaybackScript(mediaKind string) string {
	return fmt.Sprintf(`async () => {
		const kind = %q;
		const preferredTag = kind === "audio" ? "AUDIO" : "VIDEO";
		const buttonSelectors = [
			".prism-big-play-btn",
			".prism-play-btn",
			".xgplayer-start",
			".xgplayer-play",
			".txp_btn_play",
			".play-btn",
			".play-button",
			".play-icon",
			".aliplayer-play-btn",
			"[aria-label*='播放']",
			"[title*='播放']"
		];
		const roots = [];
		const queue = [document];
		const seen = new Set();
		while (queue.length > 0 && roots.length < 20) {
			const root = queue.shift();
			if (!root || seen.has(root)) {
				continue;
			}
			seen.add(root);
			roots.push(root);
			if (!root.createTreeWalker && !root.querySelectorAll) {
				continue;
			}
			const scanRoot = root.documentElement || root;
			if (scanRoot.createTreeWalker) {
				const walker = scanRoot.createTreeWalker(scanRoot, NodeFilter.SHOW_ELEMENT);
				let node;
				while ((node = walker.nextNode())) {
					if (node.shadowRoot && !seen.has(node.shadowRoot)) {
						queue.push(node.shadowRoot);
					}
				}
			}
		}
		const collect = (selector) => {
			const elements = [];
			const dedupe = new Set();
			for (const root of roots) {
				if (!root || !root.querySelectorAll) continue;
				for (const el of root.querySelectorAll(selector)) {
					if (dedupe.has(el)) continue;
					dedupe.add(el);
					elements.push(el);
				}
			}
			return elements;
		};
		const isVisible = (el) => {
			if (!el) return false;
			const style = window.getComputedStyle(el);
			const rect = el.getBoundingClientRect();
			return style && style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
		};
		const clickPlayButtons = () => {
			let clicked = 0;
			for (const selector of buttonSelectors) {
				const elements = collect(selector);
				for (const el of elements) {
					if (!isVisible(el)) continue;
					try {
						const rect = el.getBoundingClientRect();
						const cx = rect.width > 0 ? rect.left + rect.width * 0.2 + Math.random() * rect.width * 0.6 : 0;
						const cy = rect.height > 0 ? rect.top + rect.height * 0.2 + Math.random() * rect.height * 0.6 : 0;
						el.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, view: window, clientX: cx, clientY: cy }));
						if (typeof el.click === "function") {
							el.click();
						}
						clicked += 1;
					} catch (e) {}
					if (clicked >= 3) {
						return clicked;
					}
				}
			}
			return clicked;
		};
		const mediaList = collect("video, audio");
		const media = mediaList.find((el) => el.tagName === preferredTag) || mediaList[0] || null;
		let clicked = clickPlayButtons();
		if (!media) {
			return {
				status: clicked > 0 ? "clicked" : "no_media",
				clicked,
				currentTime: 0,
				readyState: 0,
				mediaCount: mediaList.length,
				rootCount: roots.length,
			};
		}
		try {
			media.muted = true;
			media.autoplay = true;
			media.playsInline = true;
			media.volume = 0;
		} catch (e) {}
		try {
			await media.play();
		} catch (e) {}
		if (media.paused) {
			clicked += clickPlayButtons();
			try {
				await media.play();
			} catch (e) {}
		}
		await new Promise((resolve) => setTimeout(resolve, 300));
		return {
			status: media.paused ? "paused" : "playing",
			clicked,
			currentTime: Number(media.currentTime || 0),
			duration: Number(media.duration || 0),
			readyState: Number(media.readyState || 0),
			mediaCount: mediaList.length,
			rootCount: roots.length,
		};
	}`, mediaKind)
}

func intFromEvalValue(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func floatFromEvalValue(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}

func decodeStudyMediaState(result interface{}) studyMediaState {
	state := studyMediaState{}
	payload, ok := result.(map[string]interface{})
	if !ok {
		return state
	}
	if status, ok := payload["status"].(string); ok {
		state.Status = status
	}
	state.CurrentTime = floatFromEvalValue(payload["currentTime"])
	state.Duration = floatFromEvalValue(payload["duration"])
	state.ReadyState = intFromEvalValue(payload["readyState"])
	state.Clicked = intFromEvalValue(payload["clicked"])
	state.MediaCount = intFromEvalValue(payload["mediaCount"])
	state.RootCount = intFromEvalValue(payload["rootCount"])
	return state
}

func attemptStudyMediaPlaybackTarget(target studyMediaEvalTarget, mediaKind string) (studyMediaState, error) {
	result, err := target.Evaluate(buildStudyMediaPlaybackScript(mediaKind))
	if err != nil {
		return studyMediaState{}, err
	}
	state := decodeStudyMediaState(result)
	state.TargetURL = target.URL()
	return state, nil
}

func attemptStudyMediaPlayback(page studyPage, mediaKind string) (studyMediaState, error) {
	return attemptStudyMediaPlaybackTarget(page, mediaKind)
}

func collectStudyMediaTargets(page studyPage) []studyMediaEvalTarget {
	targets := []studyMediaEvalTarget{page}
	frameProvider, ok := page.(interface{ Frames() []playwright.Frame })
	if !ok {
		return targets
	}
	for _, frame := range frameProvider.Frames() {
		if frame == nil || frame.ParentFrame() == nil {
			continue
		}
		targets = append(targets, frame)
	}
	return targets
}

// getStudyMediaDuration 嘗試獲取頁面上媒體的時長（秒），返回 0 表示無法獲取
func getStudyMediaDuration(page studyPage) float64 {
	result, err := page.Evaluate(`() => {
		const media = document.querySelector("video, audio");
		if (!media || !isFinite(media.duration)) return 0;
		return media.duration;
	}`)
	if err != nil {
		return 0
	}
	return floatFromEvalValue(result)
}

// computeMediaLearnTime 根據媒體時長計算學習時間
// 規則：>= 60 秒的內容學習 70±10 秒即可得雙分；< 60 秒的內容需完整播放 + 小緩衝
func computeMediaLearnTime(mediaDurationSec float64, fallbackSecs int, fallbackJitter int) int {
	if mediaDurationSec <= 0 || mediaDurationSec != mediaDurationSec {
		// 無法獲取時長，使用配置的預設值
		return durationWithJitter(fallbackSecs, fallbackJitter)
	}
	if mediaDurationSec >= 60 {
		// 長內容：70 秒 + 0~10 秒隨機抖動
		return 70 + rand.Intn(11)
	}
	// 短內容：完整播放 + 5~15 秒緩衝
	return int(mediaDurationSec) + 5 + rand.Intn(11)
}

func waitForStudyMediaPlayback(page studyPage, mediaKind string, attempts int, interval time.Duration) error {
	if attempts < 1 {
		attempts = 1
	}
	if interval < 0 {
		interval = 0
	}
	var (
		lastErr   error
		lastState studyMediaState
	)
	for attempt := 1; attempt <= attempts; attempt++ {
		for _, target := range collectStudyMediaTargets(page) {
			state, err := attemptStudyMediaPlaybackTarget(target, mediaKind)
			if err == nil {
				lastState = state
				if state.IsPlaying() {
					return nil
				}
				continue
			}
			lastErr = err
		}
		if attempt < attempts && interval > 0 {
			time.Sleep(interval)
		}
	}
	if lastErr != nil {
		return fmt.Errorf("[媒体播放] %s 启动失败: %w", mediaKind, lastErr)
	}
	return fmt.Errorf("[媒体播放] %s 未进入播放状态 status=%s readyState=%d currentTime=%.2f clicked=%d media=%d roots=%d target=%s", mediaKind, lastState.Status, lastState.ReadyState, lastState.CurrentTime, lastState.Clicked, lastState.MediaCount, lastState.RootCount, lastState.TargetURL)
}

func loadStudySettings() studySettings {
	cfg := conf.GetConfig().Study
	settings := studySettings{
		RecentDays:            cfg.RecentDays,
		FallbackToAll:         cfg.FallbackToAll,
		NavigationRetryTimes:  cfg.NavigationRetryTimes,
		RetryBackoffSeconds:   cfg.RetryBackoffSeconds,
		OperationIntervalSecs: cfg.OperationIntervalSecs,
		ScoreRetryTimes:       cfg.ScoreRetryTimes,
		ArticleDurationSecs:   cfg.ArticleDurationSecs,
		ArticleDurationJitter: cfg.ArticleDurationJitter,
		VideoDurationSecs:     cfg.VideoDurationSecs,
		VideoDurationJitter:   cfg.VideoDurationJitter,
		AudioDurationSecs:     cfg.AudioDurationSecs,
		AudioDurationJitter:   cfg.AudioDurationJitter,
		// 新增配置
		AuthFailureThreshold:   cfg.AuthFailureThreshold,
		FailureCooldownMinutes: cfg.FailureCooldownMinutes,
		MaxSliderFailures:      cfg.MaxSliderFailures,
		FastFailEnabled:        cfg.FastFailEnabled,
	}
	if settings.NavigationRetryTimes < 1 {
		settings.NavigationRetryTimes = 4
	}
	if settings.RetryBackoffSeconds < 1 {
		settings.RetryBackoffSeconds = 5
	}
	if settings.OperationIntervalSecs < 1 {
		settings.OperationIntervalSecs = 7
	}
	if settings.ScoreRetryTimes < 0 {
		settings.ScoreRetryTimes = 3
	}
	if settings.ArticleDurationSecs < 1 {
		settings.ArticleDurationSecs = 90
	}
	if settings.VideoDurationSecs < 1 {
		settings.VideoDurationSecs = 80
	}
	if settings.AudioDurationSecs < 1 {
		settings.AudioDurationSecs = 80
	}
	if settings.ArticleDurationJitter < 0 {
		settings.ArticleDurationJitter = 20
	}
	if settings.VideoDurationJitter < 0 {
		settings.VideoDurationJitter = 15
	}
	if settings.AudioDurationJitter < 0 {
		settings.AudioDurationJitter = 15
	}
	// 新增配置的預設值
	if settings.AuthFailureThreshold < 1 {
		settings.AuthFailureThreshold = 5
	}
	if settings.FailureCooldownMinutes < 1 {
		settings.FailureCooldownMinutes = 5
	}
	if settings.MaxSliderFailures < 1 {
		settings.MaxSliderFailures = 5
	}
	return settings
}

func parseLinkPublishTime(publishTime string) (time.Time, error) {
	layouts := []string{
		studyPublishTimeLayout,
		"2006-01-02 15:04",
		"2006-01-02",
		time.RFC3339,
	}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, publishTime, time.Local)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析发布时间: %s", publishTime)
}

func filterLinksByRecentDays(links []Link, recentDays int, now time.Time) filteredStudyLinks {
	if recentDays <= 0 {
		cloned := append([]Link(nil), links...)
		return filteredStudyLinks{Links: cloned}
	}
	cutoff := now.AddDate(0, 0, -recentDays)
	filtered := make([]Link, 0, len(links))
	oldCount := 0
	invalidCount := 0
	for _, link := range links {
		publishedAt, err := parseLinkPublishTime(link.PublishTime)
		if err != nil {
			invalidCount++
			continue
		}
		if publishedAt.Before(cutoff) {
			oldCount++
			continue
		}
		filtered = append(filtered, link)
	}
	sort.Slice(filtered, func(i, j int) bool {
		left, leftErr := parseLinkPublishTime(filtered[i].PublishTime)
		right, rightErr := parseLinkPublishTime(filtered[j].PublishTime)
		if leftErr != nil || rightErr != nil {
			return filtered[i].PublishTime > filtered[j].PublishTime
		}
		return left.After(right)
	})
	return filteredStudyLinks{
		Links:        filtered,
		OldCount:     oldCount,
		InvalidCount: invalidCount,
	}
}

func prepareStudyLinks(modelName string, settings studySettings, now time.Time) ([]Link, error) {
	links, err := getLinks(modelName)
	if err != nil {
		return nil, err
	}
	filtered := filterLinksByRecentDays(links, settings.RecentDays, now)
	if settings.RecentDays > 0 {
		log.Infof("[内容筛选] 模式=%s 原始=%d 保留=%d 超期=%d 时间异常=%d 条件=最近%d天", modelName, len(links), len(filtered.Links), filtered.OldCount, filtered.InvalidCount, settings.RecentDays)
	}
	if len(filtered.Links) == 0 {
		if settings.FallbackToAll {
			log.Warningf("[内容筛选] 模式=%s 最近%d天没有可用内容，回退到未筛选列表", modelName, settings.RecentDays)
			filtered.Links = append([]Link(nil), links...)
		} else {
			return nil, fmt.Errorf("[内容筛选] 模式=%s 最近%d天没有可用内容，请调整 study.recent_days", modelName, settings.RecentDays)
		}
	}
	rand.Shuffle(len(filtered.Links), func(i, j int) {
		filtered.Links[i], filtered.Links[j] = filtered.Links[j], filtered.Links[i]
	})
	return filtered.Links, nil
}

func shouldScrollAtStep(step int, interval int) bool {
	return step > 0 && interval > 0 && step%interval == 0
}

// maybeMouseDrift 僅在捲動節點後注入一次低成本可信滑鼠移動。
// 保留 isTrusted: true 的特徵，但避免在每秒學習循環中持續製造高密度事件。
func maybeMouseDrift(page studyPage) {
	if rand.Intn(4) != 0 {
		return
	}
	x := float64(100 + rand.Intn(600))
	y := float64(200 + rand.Intn(400))
	if p, ok := page.(playwright.Page); ok {
		_ = p.Mouse().Move(x, y)
	}
}

func durationWithJitter(baseSeconds int, jitterSeconds int) int {
	if jitterSeconds <= 0 {
		return baseSeconds
	}
	return baseSeconds + rand.Intn(jitterSeconds+1)
}

func getUserScoreWithRetry(user *model.User, retries int) (Score, error) {
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		score, err := GetUserScore(user.ToCookies())
		if err == nil {
			return score, nil
		}
		lastErr = err

		// 檢查是否為登入失效錯誤
		if CheckAuthError(err) {
			log.Errorf("[積分檢查] 檢測到登入失效: %v", err)
			return Score{}, NewAuthError("獲取積分時檢測到登入失效", err)
		}

		if attempt < retries {
			log.Warningf("[積分檢查] 第%d次獲取積分失敗: %v", attempt+1, err)
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}
	return Score{}, fmt.Errorf("[積分檢查] 連續獲取積分失敗，可能是網絡異常或登錄已失效: %w", lastErr)
}

func navigateStudyPageWithRetry(page studyPage, url string, referer string, settings studySettings) error {
	var lastErr error
	for attempt := 1; attempt <= settings.NavigationRetryTimes; attempt++ {
		lastErr = navigateStudyPage(page, url, referer)
		if lastErr == nil {
			if attempt > 1 {
				log.Infof("[導航] 第%d次嘗試成功 url=%s", attempt, url)
			}
			return nil
		}

		// 檢查是否為登入失效錯誤
		if CheckAuthError(lastErr) {
			log.Errorf("[導航] 檢測到登入失效 url=%s: %v", url, lastErr)
			return NewAuthError("頁面導航時檢測到登入失效", lastErr)
		}

		log.Warningf("[導航] 第%d/%d次嘗試失敗 url=%s err=%v", attempt, settings.NavigationRetryTimes, url, lastErr)
		if attempt < settings.NavigationRetryTimes {
			time.Sleep(time.Duration(settings.RetryBackoffSeconds) * time.Second)
		}
	}
	return lastErr
}

type Link struct {
	Editor       string   `json:"editor"`
	PublishTime  string   `json:"publishTime"`
	ItemType     string   `json:"itemType"`
	Author       string   `json:"author"`
	CrossTime    int      `json:"crossTime"`
	Source       string   `json:"source"`
	NameB        string   `json:"nameB"`
	Title        string   `json:"title"`
	Type         string   `json:"type"`
	Url          string   `json:"url"`
	ShowSource   string   `json:"showSource"`
	ItemId       string   `json:"itemId"`
	ThumbImage   string   `json:"thumbImage"`
	AuditTime    string   `json:"auditTime"`
	ChannelNames []string `json:"channelNames"`
	Producer     string   `json:"producer"`
	ChannelIds   []string `json:"channelIds"`
	DataValid    bool     `json:"dataValid"`
}

// 获取学习链接列表
func getLinks(model string) ([]Link, error) {
	UID := rand.Intn(20000000) + 10000000
	learnUrl := ""
	if model == "article" {
		learnUrl = article_url_list[rand.Intn(len(article_url_list))]
	} else if model == "video" {
		learnUrl = video_url_list[rand.Intn(len(video_url_list))]
	} else if model == "yp" {
		learnUrl = yp_url_list[rand.Intn(len(yp_url_list))]
	} else {
		return nil, errors.New("model选择出现错误")
	}
	var (
		resp []byte
	)

	response, err := utils.GetClient().R().SetQueryParam("_st", strconv.Itoa(UID)).Get(learnUrl)
	if err != nil {
		log.Errorln("请求链接列表出现错误！" + err.Error())
		return nil, err
	}
	resp = response.Bytes()

	var links []Link
	err = json.Unmarshal(resp, &links)
	if err != nil {
		log.Errorln("解析列表出现错误" + err.Error())
		return nil, err
	}
	return links, err
}

// 文章学习
func (c *Core) LearnArticle(user *model.User) {
	defer func() {
		err := recover()
		if err != nil {
			log.Errorln("文章学习模块异常结束")
			log.Errorln(err)
		}
	}()
	if c.IsQuit() {
		return
	}
	settings := loadStudySettings()

	// 建立登入狀態檢測器
	authChecker := NewAuthChecker(AuthCheckerConfig{
		MaxFailures:       settings.AuthFailureThreshold,
		CooldownPeriod:    time.Duration(settings.FailureCooldownMinutes) * time.Minute,
		MaxSliderFailures: settings.MaxSliderFailures,
		ModuleName:        "文章學習",
	})

	score, err := getUserScoreWithRetry(user, settings.ScoreRetryTimes)
	if err != nil {
		// 檢查是否為登入失效
		if _, ok := err.(*AuthError); ok || CheckAuthError(err) {
			log.Errorln("[文章學習] 登入失效，停止學習: " + err.Error())
			c.Push(user.PushId, "text", "文章學習：登入已失效，請重新登入")
			return
		}
		log.Errorln(err.Error())
		return
	}
	links, err := prepareStudyLinks("article", settings, time.Now())
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	if score.Content["article"].CurrentScore < score.Content["article"].MaxScore {
		log.Infoln("开始加载文章学习模块")

		context, err := c.browser.NewContext(playwright.BrowserNewContextOptions{
			Viewport: &playwright.Size{
				Width:  int(1920),
				Height: int(1080),
			}})
		_ = context.AddInitScript(playwright.Script{
			Content: playwright.String("Object.defineProperties(navigator, {webdriver:{get:()=>undefined}});")})
		if err != nil {
			log.Errorln("创建实例对象错误" + err.Error())
			return
		}

		defer func(context playwright.BrowserContext) {
			err := context.Close()
			if err != nil {
				log.Errorln("错误的关闭了实例对象" + err.Error())
			}
		}(context)

		page, err := context.NewPage()
		if err != nil {
			return
		}
		defer func() {
			err := page.Close()
			if err != nil {
				log.Errorln("关闭页面失败")
				return
			}
		}()

		err = context.AddCookies(user.ToBrowserCookies())
		if err != nil {
			log.Errorln("添加cookie失败" + err.Error())
			return
		}

		tryCount := 0

		for {
			// 檢查是否應該提前停止
			if authChecker.ShouldStop() && settings.FastFailEnabled {
				log.Errorln("[文章學習] 連續失敗次數過多，提前停止學習")
				c.Push(user.PushId, "text", "文章學習：連續失敗次數過多，已提前停止")
				return
			}

			if tryCount < 20 {
				PrintScore(score)
				n := rand.Intn(len(links))
				referer := links[rand.Intn(len(links))].Url
				err := navigateStudyPageWithRetry(page, links[n].Url, referer, settings)
				if err != nil {
					// 檢查是否為登入失效
					if _, ok := err.(*AuthError); ok || CheckAuthError(err) {
						log.Errorf("[文章學習] 登入失效: %v", err)
						c.Push(user.PushId, "text", "文章學習：登入已失效，請重新登入")
						return
					}

					log.Errorf("[文章学习] 页面跳转失败 title=%s url=%s err=%v", links[n].Title, links[n].Url, err)

					// 記錄失敗
					if !authChecker.RecordFailure(err) {
						log.Errorln("[文章學習] 失敗次數達到上限，停止學習")
						return
					}
					tryCount++
					continue
				}

				// 成功後重置失敗計數
				authChecker.Reset()

				log.Infoln("正在学习文章：" + links[n].Title)
				c.Push(user.PushId, "text", "正在学习文章："+links[n].Title)
				log.Infoln("文章发布时间：" + links[n].PublishTime)
				log.Infoln("文章学习链接：" + links[n].Url)
				humanPause(1500, 2800)
				learnTime := durationWithJitter(settings.ArticleDurationSecs, settings.ArticleDurationJitter)
				for i := 0; i < learnTime; i++ {
					if c.IsQuit() {
						return
					}
					fmt.Printf("\r[%v] [INFO]: 正在进行阅读学习中，剩余%d篇，本篇剩余时间%d秒", time.Now().Format("2006-01-02 15:04:05"), score.Content["article"].MaxScore-score.Content["article"].CurrentScore, learnTime-i)

					if shouldScrollAtStep(i, settings.OperationIntervalSecs) {
						if err := scrollStudyPage(page, i); err != nil {
							log.Errorf("[文章学习] 页面滚动失败 title=%s step=%d err=%v", links[n].Title, i, err)
						}
						maybeMouseDrift(page)
					}
					humanPause(850, 1350)
				}
				fmt.Println()
				score, err = getUserScoreWithRetry(user, settings.ScoreRetryTimes)
				if err != nil {
					// 檢查是否為登入失效
					if _, ok := err.(*AuthError); ok || CheckAuthError(err) {
						log.Errorln("[文章學習] 獲取積分時檢測到登入失效: " + err.Error())
						c.Push(user.PushId, "text", "文章學習：登入已失效，請重新登入")
						return
					}
					log.Errorln(err.Error())
					return
				}
				if score.Content["article"].CurrentScore >= score.Content["article"].MaxScore {
					log.Infoln("检测到本次阅读学习分数已满，退出学习")
					break
				}

				tryCount++
			} else {
				log.Errorln("阅读学习出现异常，稍后可重新学习")
				return
			}
		}
	} else {
		log.Infoln("检测到文章学习已经完成")
	}
}

// 视频学习
func (c *Core) LearnVideo(user *model.User) {
	defer func() {
		err := recover()
		if err != nil {
			log.Errorln("视频学习模块异常结束")
			log.Errorln(err)
		}
	}()
	if c.IsQuit() {
		return
	}
	settings := loadStudySettings()

	// 建立登入狀態檢測器
	authChecker := NewAuthChecker(AuthCheckerConfig{
		MaxFailures:       settings.AuthFailureThreshold,
		CooldownPeriod:    time.Duration(settings.FailureCooldownMinutes) * time.Minute,
		MaxSliderFailures: settings.MaxSliderFailures,
		ModuleName:        "視頻學習",
	})

	score, err := getUserScoreWithRetry(user, settings.ScoreRetryTimes)
	if err != nil {
		// 檢查是否為登入失效
		if _, ok := err.(*AuthError); ok || CheckAuthError(err) {
			log.Errorln("[視頻學習] 登入失效，停止學習: " + err.Error())
			c.Push(user.PushId, "text", "視頻學習：登入已失效，請重新登入")
			return
		}
		log.Errorln(err.Error())
		return
	}
	links, err := prepareStudyLinks("video", settings, time.Now())
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	if !(score.Content["video"].CurrentScore >= score.Content["video"].MaxScore && score.Content["video_time"].CurrentScore >= score.Content["video_time"].MaxScore) {
		log.Infoln("开始加载视频学习模块")
		// core := Core{}
		// core.Init()

		context, err := c.browser.NewContext(playwright.BrowserNewContextOptions{
			Viewport: &playwright.Size{
				Width:  int(1920),
				Height: int(1080),
			}})
		_ = context.AddInitScript(playwright.Script{
			Content: playwright.String("Object.defineProperties(navigator, {webdriver:{get:()=>undefined}});")})
		if err != nil {
			log.Errorln("创建实例对象错误" + err.Error())
			return
		}
		defer func(context playwright.BrowserContext) {
			err := context.Close()
			if err != nil {
				log.Errorln("错误的关闭了实例对象" + err.Error())
			}
		}(context)

		page, err := context.NewPage()
		if err != nil {
			return
		}
		defer func() {
			page.Close()
		}()

		err = context.AddCookies(user.ToBrowserCookies())
		if err != nil {
			log.Errorln("添加cookie失败" + err.Error())
			return
		}
		tryCount := 0
		noGrowthCount := 0
		for {
			// 檢查是否應該提前停止
			if authChecker.ShouldStop() && settings.FastFailEnabled {
				log.Errorln("[視頻學習] 連續失敗次數過多，提前停止學習")
				c.Push(user.PushId, "text", "視頻學習：連續失敗次數過多，已提前停止")
				return
			}

			if tryCount < 20 {
				PrintScore(score)
				n := rand.Intn(len(links))
				referer := links[rand.Intn(len(links))].Url
				err := navigateStudyPageWithRetry(page, links[n].Url, referer, settings)
				if err != nil {
					// 檢查是否為登入失效
					if _, ok := err.(*AuthError); ok || CheckAuthError(err) {
						log.Errorf("[視頻學習] 登入失效: %v", err)
						c.Push(user.PushId, "text", "視頻學習：登入已失效，請重新登入")
						return
					}

					log.Errorf("[视频学习] 页面跳转失败 title=%s url=%s err=%v", links[n].Title, links[n].Url, err)

					// 記錄失敗
					if !authChecker.RecordFailure(err) {
						log.Errorln("[視頻學習] 失敗次數達到上限，停止學習")
						return
					}
					tryCount++
					continue
				}

				// 成功後重置失敗計數
				authChecker.Reset()

				playbackErr := waitForStudyMediaPlayback(page, "video", studyMediaStartAttempts, studyMediaStartInterval)
				if playbackErr != nil {
					log.Warningf("[视频学习] 未确认播放，改用页面停留兜底 title=%s url=%s err=%v", links[n].Title, links[n].Url, playbackErr)
				}

				log.Infoln("正在观看视频：" + links[n].Title)
				c.Push(user.PushId, "text", "正在观看视频："+links[n].Title)
				log.Infoln("视频发布时间：" + links[n].PublishTime)
				log.Infoln("视频学习链接：" + links[n].Url)
				humanPause(1800, 3200)
				beforeVideoScore := score.Content["video"]
				beforeVideoTimeScore := score.Content["video_time"]
				mediaDuration := getStudyMediaDuration(page)
				learnTime := computeMediaLearnTime(mediaDuration, settings.VideoDurationSecs, settings.VideoDurationJitter)
				if mediaDuration > 0 {
					log.Infof("[视频学习] 检测到媒体时长 %.0f 秒，计划学习 %d 秒", mediaDuration, learnTime)
				}
				for i := 0; i < learnTime; i++ {
					if c.IsQuit() {
						return
					}
					fmt.Printf("\r[%v] [INFO]: 正在进行视频学习中，剩余%d个，当前剩余时间%d秒", time.Now().Format("2006-01-02 15:04:05"), score.Content["video"].MaxScore-score.Content["video"].CurrentScore, learnTime-i)

					if shouldScrollAtStep(i, settings.OperationIntervalSecs) {
						if err := scrollStudyPage(page, i); err != nil {
							log.Errorf("[视频学习] 页面滚动失败 title=%s step=%d err=%v", links[n].Title, i, err)
						}
						maybeMouseDrift(page)
					}
					humanPause(850, 1350)
				}
				fmt.Println()
				score, err = getUserScoreWithRetry(user, settings.ScoreRetryTimes)
				if err != nil {
					// 檢查是否為登入失效
					if _, ok := err.(*AuthError); ok || CheckAuthError(err) {
						log.Errorln("[視頻學習] 獲取積分時檢測到登入失效: " + err.Error())
						c.Push(user.PushId, "text", "視頻學習：登入已失效，請重新登入")
						return
					}
					log.Errorln(err.Error())
					return
				}
				if score.Content["video"].CurrentScore <= beforeVideoScore.CurrentScore && score.Content["video_time"].CurrentScore <= beforeVideoTimeScore.CurrentScore {
					log.Warningf("[视频学习] 本次未检测到积分增长 title=%s video=%d/%d->%d/%d video_time=%d/%d->%d/%d",
						links[n].Title,
						beforeVideoScore.CurrentScore, beforeVideoScore.MaxScore,
						score.Content["video"].CurrentScore, score.Content["video"].MaxScore,
						beforeVideoTimeScore.CurrentScore, beforeVideoTimeScore.MaxScore,
						score.Content["video_time"].CurrentScore, score.Content["video_time"].MaxScore,
					)
					noGrowthCount++
					if playbackErr != nil && !authChecker.RecordFailure(playbackErr) {
						log.Errorln("[視頻學習] 失敗次數達到上限，停止學習")
						return
					}
					if playbackErr != nil && noGrowthCount >= 2 {
						log.Warningln("[视频学习] 连续未增长且播放器无法确认，提前切换到音频兜底")
						return
					}
				} else {
					noGrowthCount = 0
				}
				if score.Content["video"].CurrentScore >= score.Content["video"].MaxScore && score.Content["video_time"].CurrentScore >= score.Content["video_time"].MaxScore {
					log.Infoln("检测到本次视频学习分数已满，退出学习")
					break
				}

				tryCount++
			} else {
				log.Errorln("视频学习出现异常，稍后可重新学习")
				return
			}
		}
	} else {
		log.Infoln("检测到视频学习已经完成")
	}
}

// 音频学习
func (c *Core) RadioStation(user *model.User) {
	defer func() {
		err := recover()
		if err != nil {
			log.Errorln("电台学习模块异常结束")
			log.Errorln(err)
		}
	}()
	if c.IsQuit() {
		return
	}
	settings := loadStudySettings()

	// 建立登入狀態檢測器
	authChecker := NewAuthChecker(AuthCheckerConfig{
		MaxFailures:       settings.AuthFailureThreshold,
		CooldownPeriod:    time.Duration(settings.FailureCooldownMinutes) * time.Minute,
		MaxSliderFailures: settings.MaxSliderFailures,
		ModuleName:        "音頻學習",
	})

	score, err := getUserScoreWithRetry(user, settings.ScoreRetryTimes)
	if err != nil {
		// 檢查是否為登入失效
		if _, ok := err.(*AuthError); ok || CheckAuthError(err) {
			log.Errorln("[音頻學習] 登入失效，停止學習: " + err.Error())
			c.Push(user.PushId, "text", "音頻學習：登入已失效，請重新登入")
			return
		}
		log.Errorln(err.Error())
		return
	}
	links, err := prepareStudyLinks("yp", settings, time.Now())
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	if !(score.Content["video"].CurrentScore >= score.Content["video"].MaxScore && score.Content["video_time"].CurrentScore >= score.Content["video_time"].MaxScore) {
		log.Infoln("开始加载音频学习模块")
		context, err := c.browser.NewContext(playwright.BrowserNewContextOptions{
			Viewport: &playwright.Size{
				Width:  int(1920),
				Height: int(1080),
			}})
		_ = context.AddInitScript(playwright.Script{
			Content: playwright.String("Object.defineProperties(navigator, {webdriver:{get:()=>undefined}});")})
		if err != nil {
			log.Errorln("创建实例对象错误" + err.Error())
			return
		}
		defer func(context playwright.BrowserContext) {
			err := context.Close()
			if err != nil {
				log.Errorln("错误的关闭了实例对象" + err.Error())
			}
		}(context)

		page, err := context.NewPage()
		if err != nil {
			return
		}
		defer func() {
			page.Close()
		}()

		err = context.AddCookies(user.ToBrowserCookies())
		if err != nil {
			log.Errorln("添加cookie失败" + err.Error())
			return
		}
		tryCount := 0
		noGrowthCount := 0
		for {
			// 檢查是否應該提前停止
			if authChecker.ShouldStop() && settings.FastFailEnabled {
				log.Errorln("[音頻學習] 連續失敗次數過多，提前停止學習")
				c.Push(user.PushId, "text", "音頻學習：連續失敗次數過多，已提前停止")
				return
			}

			if tryCount < 20 {
				PrintScore(score)
				n := rand.Intn(len(links))
				referer := links[rand.Intn(len(links))].Url
				err := navigateStudyPageWithRetry(page, links[n].Url, referer, settings)
				if err != nil {
					// 檢查是否為登入失效
					if _, ok := err.(*AuthError); ok || CheckAuthError(err) {
						log.Errorf("[音頻學習] 登入失效: %v", err)
						c.Push(user.PushId, "text", "音頻學習：登入已失效，請重新登入")
						return
					}

					log.Errorf("[音频学习] 页面跳转失败 title=%s url=%s err=%v", links[n].Title, links[n].Url, err)

					// 記錄失敗
					if !authChecker.RecordFailure(err) {
						log.Errorln("[音頻學習] 失敗次數達到上限，停止學習")
						return
					}
					tryCount++
					continue
				}

				// 成功後重置失敗計數
				authChecker.Reset()

				playbackErr := waitForStudyMediaPlayback(page, "audio", studyMediaStartAttempts, studyMediaStartInterval)
				if playbackErr != nil {
					log.Warningf("[音频学习] 未确认播放，改用页面停留兜底 title=%s url=%s err=%v", links[n].Title, links[n].Url, playbackErr)
				}

				log.Infoln("正在收听：" + links[n].Title)
				c.Push(user.PushId, "text", "正在收听："+links[n].Title)
				log.Infoln("音频发布时间：" + links[n].PublishTime)
				log.Infoln("音频学习链接：" + links[n].Url)
				humanPause(1500, 2800)
				beforeVideoScore := score.Content["video"]
				beforeVideoTimeScore := score.Content["video_time"]
				mediaDuration := getStudyMediaDuration(page)
				learnTime := computeMediaLearnTime(mediaDuration, settings.AudioDurationSecs, settings.AudioDurationJitter)
				if mediaDuration > 0 {
					log.Infof("[音频学习] 检测到媒体时长 %.0f 秒，计划学习 %d 秒", mediaDuration, learnTime)
				}
				for i := 0; i < learnTime; i++ {
					if c.IsQuit() {
						return
					}
					fmt.Printf("\r[%v] [INFO]: 正在进行音频学习中，剩余%d个，当前剩余时间%d秒", time.Now().Format("2006-01-02 15:04:05"), score.Content["video"].MaxScore-score.Content["video"].CurrentScore, learnTime-i)

					if shouldScrollAtStep(i, settings.OperationIntervalSecs) {
						if err := scrollStudyPage(page, i); err != nil {
							log.Errorf("[音频学习] 页面滚动失败 title=%s step=%d err=%v", links[n].Title, i, err)
						}
						maybeMouseDrift(page)
					}
					humanPause(850, 1350)
				}
				fmt.Println()
				score, err = getUserScoreWithRetry(user, settings.ScoreRetryTimes)
				if err != nil {
					// 檢查是否為登入失效
					if _, ok := err.(*AuthError); ok || CheckAuthError(err) {
						log.Errorln("[音頻學習] 獲取積分時檢測到登入失效: " + err.Error())
						c.Push(user.PushId, "text", "音頻學習：登入已失效，請重新登入")
						return
					}
					log.Errorln(err.Error())
					return
				}
				if score.Content["video"].CurrentScore <= beforeVideoScore.CurrentScore && score.Content["video_time"].CurrentScore <= beforeVideoTimeScore.CurrentScore {
					log.Warningf("[音频学习] 本次未检测到积分增长 title=%s video=%d/%d->%d/%d video_time=%d/%d->%d/%d",
						links[n].Title,
						beforeVideoScore.CurrentScore, beforeVideoScore.MaxScore,
						score.Content["video"].CurrentScore, score.Content["video"].MaxScore,
						beforeVideoTimeScore.CurrentScore, beforeVideoTimeScore.MaxScore,
						score.Content["video_time"].CurrentScore, score.Content["video_time"].MaxScore,
					)
					noGrowthCount++
				}
				if playbackErr != nil && score.Content["video"].CurrentScore <= beforeVideoScore.CurrentScore && score.Content["video_time"].CurrentScore <= beforeVideoTimeScore.CurrentScore {
					if !authChecker.RecordFailure(playbackErr) {
						log.Errorln("[音頻學習] 失敗次數達到上限，停止學習")
						return
					}
					if noGrowthCount >= 2 {
						log.Warningln("[音频学习] 连续未增长且播放器无法确认，提前结束音频兜底")
						return
					}
				} else {
					noGrowthCount = 0
				}
				if score.Content["video"].CurrentScore >= score.Content["video"].MaxScore && score.Content["video_time"].CurrentScore >= score.Content["video_time"].MaxScore {
					log.Infoln("检测到本次音频学习分数已满，退出学习")
					break
				}

				tryCount++
			} else {
				log.Errorln("音频学习出现异常，稍后可重新学习")
				return
			}
		}
	} else {
		log.Infoln("检测到音频学习已经完成")
	}
}
