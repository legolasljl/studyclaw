package lib

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	rand2 "math/rand"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/imroc/req/v3"
	"github.com/playwright-community/playwright-go"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/legolasljl/studyclaw/conf"
	"github.com/legolasljl/studyclaw/model"
	"github.com/legolasljl/studyclaw/utils"
)

const (
	MyPointsUri      = "https://pc.xuexi.cn/points/my-points.html"
	DailyPracticeUri = "https://pc.xuexi.cn/points/exam-practice.html"

	DailyBUTTON   = `#app > div > div.layout-body > div > div.my-points-section > div.my-points-content > div:nth-child(4) > div.my-points-card-footer > div.buttonbox > div`
	WEEKEND       = `#app > div > div.layout-body > div > div.my-points-section > div.my-points-content > div:nth-child(7) > div.my-points-card-footer > div.buttonbox > div`
	SPECIALBUTTON = `#app > div > div.layout-body > div > div.my-points-section > div.my-points-content > div:nth-child(6) > div.my-points-card-footer > div.buttonbox > div`
)

var (
	answerWorkspaceSelectors = []string{
		`#app .detail-body`,
		`.detail-body`,
		`#app .layout-body`,
	}
	answerQuestionSelectors = []string{
		`#app .detail-body .question .q-header`,
		`.detail-body .question .q-header`,
		`.question .q-header`,
	}
	answerSliderExactSelectors = []string{
		`#nc_mask > div`,
		`#nc_1_wrapper`,
		`.nc-container`,
		`[id^="nc_"][id$="_wrapper"]`,
		`.nc_wrapper`,
		`.nc_mask`,
		`#JS_aliyun-captcha-slider_bind_element`,
		`.JS_aliyun-captcha-slider_bind_element`,
		`[id*="aliyun-captcha"]`,
		`[class*="aliyun-captcha"]`,
		`[class*="captcha"][class*="slider"]`,
		`[class*="verify"][class*="slider"]`,
		`[id*="captcha"][id*="slider"]`,
	}
	answerSliderFallbackSelectors = []string{
		`[class*="drag-verify"]`,
		`[class*="slide-verify"]`,
		`.slidercontainer`,
		`.slide-verify`,
		`.drag_verify`,
		`[class*="captcha-slider"]`,
	}
)

const answerSliderManualWait = 90 * time.Second

func randomDurationBetween(minMs int, maxMs int) time.Duration {
	if minMs < 0 {
		minMs = 0
	}
	if maxMs < minMs {
		maxMs = minMs
	}
	if maxMs == minMs {
		return time.Duration(minMs) * time.Millisecond
	}
	return time.Duration(minMs+rand2.Intn(maxMs-minMs+1)) * time.Millisecond
}

func humanPause(minMs int, maxMs int) {
	time.Sleep(randomDurationBetween(minMs, maxMs))
}

// simulateHumanBehavior 模擬人類在頁面上的自然行為（鼠標移動、滾動等）
func simulateHumanBehavior(page playwright.Page) {
	// 隨機頁面滾動
	scrollDirection := rand2.Intn(3)
	scrollAmount := rand2.Intn(150) + 50
	var scrollY int
	switch scrollDirection {
	case 0: // 向下滾動
		scrollY = scrollAmount
	case 1: // 向上滾動
		scrollY = -scrollAmount
	default: // 不滾動
		scrollY = 0
	}
	if scrollY != 0 {
		_, _ = page.Evaluate(fmt.Sprintf(`() => window.scrollBy(0, %d)`, scrollY))
		humanPause(300, 800)
	}

	// 隨機鼠標移動到頁面不同位置（使用 Playwright 原生方法，產生 isTrusted: true 事件）
	mouseX := float64(rand2.Intn(800) + 100)
	mouseY := float64(rand2.Intn(500) + 100)
	steps := rand2.Intn(5) + 3
	page.Mouse().Move(mouseX, mouseY, playwright.MouseMoveOptions{
		Steps: &steps,
	})

	humanPause(200, 500)
}

func hasVisibleSelector(page playwright.Page, selectors []string) bool {
	for _, selector := range selectors {
		handle, err := page.QuerySelector(selector)
		if err != nil || handle == nil {
			continue
		}
		ok, err := handle.IsVisible()
		if err == nil && ok {
			return true
		}
	}
	return false
}

func waitForVisibleSelector(page playwright.Page, selectors []string, attempts int, minPauseMs int, maxPauseMs int) bool {
	if attempts < 1 {
		attempts = 1
	}
	for i := 0; i < attempts; i++ {
		if hasVisibleSelector(page, selectors) {
			return true
		}
		if i < attempts-1 {
			humanPause(minPauseMs, maxPauseMs)
		}
	}
	return false
}

func hasAnswerQuestion(page playwright.Page) bool {
	return hasVisibleSelector(page, answerQuestionSelectors)
}

func hasAnswerSliderExactPrompt(page playwright.Page) bool {
	return hasVisibleSelector(page, answerSliderExactSelectors)
}

func hasPotentialAnswerSliderElement(page playwright.Page) bool {
	result, err := page.Evaluate(`() => {
		const targetSelector = '[id*="captcha"], [id*="slider"], [id*="nc_"], [id*="drag"], [id*="slide"], [class*="captcha"], [class*="slider"], [class*="nc_"], [class*="slide"], [class*="drag"]';
		const elements = document.querySelectorAll(targetSelector);
		for (const el of elements) {
			const rect = el.getBoundingClientRect();
			if (rect.width <= 0 || rect.height <= 0) continue;
			const style = window.getComputedStyle(el);
			if (style.display === "none" || style.visibility === "hidden" || style.opacity === "0") continue;
			return true;
		}
		return false;
	}`)
	if err != nil {
		return false
	}
	ok, _ := result.(bool)
	return ok
}

func containsAnswerSliderSpecificText(text string) bool {
	normalized := normalizeAnswerButtonText(text)
	if normalized == "" {
		return false
	}
	phrases := []string{
		"请按住滑块",
		"拖动到最右边",
		"向右滑动验证",
		"滑块验证",
		"滑块校验",
	}
	for _, phrase := range phrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func containsAnswerSliderLooseText(text string) bool {
	normalized := normalizeAnswerButtonText(text)
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "滑块") ||
		strings.Contains(normalized, "滑动验证") ||
		strings.Contains(normalized, "拖动验证") ||
		strings.Contains(normalized, "请拖动") ||
		strings.Contains(normalized, "按住滑动")
}

func containsAnswerSliderContextText(text string) bool {
	normalized := normalizeAnswerButtonText(text)
	if normalized == "" {
		return false
	}
	contextKeywords := []string{
		"请按住",
		"拖动到最右边",
		"向右滑动",
		"安全验证",
		"完成验证",
		"验证码",
		"校验",
	}
	for _, keyword := range contextKeywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

func containsAnswerFlowBlockedText(text string) bool {
	normalized := normalizeAnswerButtonText(text)
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "请不要中途开启新的答题流程") ||
		strings.Contains(normalized, "不支持多端同时作答") ||
		strings.Contains(normalized, "答题流程")
}

func getAnswerPageText(page playwright.Page) string {
	result, err := page.Evaluate(`() => document.body ? document.body.innerText || "" : ""`)
	if err != nil {
		return ""
	}
	text, ok := result.(string)
	if !ok {
		return ""
	}
	return text
}

func hasAnswerSliderDeepPrompt(page playwright.Page) bool {
	if hasAnswerSliderExactPrompt(page) {
		log.Infoln("[答題] 通過精確選擇器檢測到滑塊驗證")
		return true
	}

	text := getAnswerPageText(page)
	if containsAnswerSliderSpecificText(text) {
		log.Infoln("[答題] 通過滑塊文本檢測到驗證提示")
		return true
	}

	if containsAnswerSliderContextText(text) && hasVisibleSelector(page, answerSliderFallbackSelectors) {
		log.Infoln("[答題] 通過備援選擇器檢測到滑塊驗證")
		return true
	}

	if containsAnswerSliderLooseText(text) && hasPotentialAnswerSliderElement(page) {
		log.Infoln("[答題] 通過深度檢查檢測到滑塊驗證")
		return true
	}

	return false
}

func hasAnswerSliderPrompt(page playwright.Page) bool {
	return hasAnswerSliderDeepPrompt(page)
}

// detectAndLogSliderElements 檢測並打印頁面上所有可能的滑塊元素（用於調試）
func detectAndLogSliderElements(page playwright.Page) {
	result, _ := page.Evaluate(`() => {
		const targetSelector = '[id*="captcha"], [id*="slider"], [id*="nc_"], [id*="drag"], [id*="slide"], [class*="captcha"], [class*="slider"], [class*="nc_"], [class*="slide"], [class*="drag"]';
		const elements = document.querySelectorAll(targetSelector);
		const found = [];
		for (const el of elements) {
			const rect = el.getBoundingClientRect();
			if (rect.width > 0 && rect.height > 0) {
				found.push({
					tag: el.tagName,
					className: (el.className || '').toString(),
					id: el.id || '',
					text: (el.innerText || el.textContent || '').toString().trim().substring(0, 30),
					width: rect.width,
					height: rect.height
				});
			}
		}
		return found.slice(0, 20);
	}`)

	if elements, ok := result.([]interface{}); ok && len(elements) > 0 {
		log.Infoln("[答題調試] 檢測到可能的滑塊元素：", len(elements), "個")
		for i, el := range elements {
			if m, ok := el.(map[string]interface{}); ok {
				log.Infoln("[答題調試] 元素", i, ": tag=", m["tag"], "class=", m["className"], "id=", m["id"], "text=", m["text"])
			}
		}
	} else {
		log.Infoln("[答題調試] 未檢測到滑塊相關元素")
	}
}

func isAnswerCompletionText(text string) bool {
	normalized := normalizeAnswerButtonText(text)
	if normalized == "" {
		return false
	}

	// 檢測答題完成的常見關鍵詞
	completionKeywords := []string{
		"本次答对题目数",
		"再来一组",
		"答题完成",
		"已结束",
		"答题结束",
		"恭喜完成",
		"满分",
	}
	for _, keyword := range completionKeywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}

	// 檢測積分獲得相關文字
	if strings.Contains(normalized, "积分") && strings.Contains(normalized, "获得") {
		return true
	}

	// 檢測結果頁的三要素
	return strings.Contains(normalized, "正确率") &&
		strings.Contains(normalized, "答错数") &&
		strings.Contains(normalized, "用时")
}

func isAnswerRoundComplete(page playwright.Page) bool {
	result, err := page.Evaluate(`() => document.body ? document.body.innerText || "" : ""`)
	if err != nil {
		return false
	}
	text, ok := result.(string)
	if !ok {
		return false
	}
	return isAnswerCompletionText(text)
}

func captureAnswerPageScreenshotBase64(page playwright.Page) (string, error) {
	bytes, err := page.Screenshot(playwright.PageScreenshotOptions{
		Type:       playwright.ScreenshotTypePng,
		FullPage:   playwright.Bool(true),
		Animations: playwright.ScreenshotAnimationsDisabled,
		Timeout:    playwright.Float(5000),
	})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

func waitForAnswerSliderDismiss(page playwright.Page, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if !hasAnswerSliderPrompt(page) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		humanPause(500, 1000)
	}
}

func (c *Core) handleAnswerSliderChallenge(page playwright.Page, user *model.User, authChecker *AuthChecker, stage string) bool {
	if !authChecker.RecordSliderFailure() {
		log.Errorln("[答題] 滑塊驗證失敗次數過多，停止答題")
		c.Push(user.PushId, "flush", "答題模組：多次遇到滑塊驗證，已停止本輪")
		return false
	}

	log.Infoln("[答題] 檢測到滑塊驗證，嘗試自動滑動...")

	// 先等待滑塊元素穩定
	humanPause(500, 1000)

	// 嘗試多次滑動（阿里雲滑塊有隨機性，多試幾次提高成功率）
	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Infoln("[答題] 滑塊驗證嘗試第", attempt, "次...")

		if err := solveAnswerSlider(page); err != nil {
			log.Warningln("[答題] 自動滑動失敗（嘗試", attempt, "）：", err.Error())
			// 等待後重試
			humanPause(1000, 2000)
			continue
		}

		// 滑動後等待結果
		humanPause(2000, 3500)

		if !hasAnswerSliderPrompt(page) {
			log.Infoln("[答題] 自動滑動成功，滑塊驗證已通過")
			authChecker.Reset()
			humanPause(800, 1500)
			return true
		}

		log.Warningln("[答題] 滑塊驗證未通過（嘗試", attempt, "），等待滑塊重置後重試...")
		// 失敗後阿里雲滑塊會重置動畫，等待重置完成
		humanPause(600, 1000)
	}

	log.Warningln("[答題] 多次自動滑動未通過，等待手動驗證 (90秒)...")
	if waitForAnswerSliderDismiss(page, 90*time.Second) {
		log.Infoln("[答題] 手動滑塊驗證已通過")
		authChecker.Reset()
		humanPause(600, 1000)
		return true
	}

	log.Warningln("[答題] 未在超時時間內完成滑塊驗證")
	return false
}

func findAnswerEntryButton(page playwright.Page, sectionKeywords []string) playwright.ElementHandle {
	cardSelectors := []string{
		`#app .my-points-content > div`,
		`.my-points-content > div`,
		`.my-points-card`,
	}
	buttonSelectors := []string{
		`button`,
		`.buttonbox > div`,
		`.buttonbox button`,
		`[role="button"]`,
		`div`,
	}

	for _, cardSelector := range cardSelectors {
		cards, err := page.QuerySelectorAll(cardSelector)
		if err != nil || len(cards) == 0 {
			continue
		}
		for _, card := range cards {
			text, err := card.TextContent()
			if err != nil {
				continue
			}
			normalizedText := normalizeAnswerButtonText(text)
			matched := false
			for _, keyword := range sectionKeywords {
				if strings.Contains(normalizedText, normalizeAnswerButtonText(keyword)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}

			for _, buttonSelector := range buttonSelectors {
				buttons, err := card.QuerySelectorAll(buttonSelector)
				if err != nil || len(buttons) == 0 {
					continue
				}
				btn := pickAnswerActionButton(buttons, []string{"去答题", "开始答题", "继续答题", "前往答题", "答题"})
				if btn != nil {
					return btn
				}
			}
		}
	}
	return nil
}

func openAnswerSection(page playwright.Page, modelName string) error {
	sectionName := ""
	sectionKeywords := []string{}
	fallbackSelector := ""

	switch modelName {
	case "daily":
		sectionName = "每日答题"
		sectionKeywords = []string{"每日答题"}
		fallbackSelector = DailyBUTTON
	case "weekly":
		sectionName = "每周答题"
		sectionKeywords = []string{"每周答题"}
		fallbackSelector = WEEKEND
	case "special":
		sectionName = "专项答题"
		sectionKeywords = []string{"专项答题"}
		fallbackSelector = SPECIALBUTTON
	default:
		return fmt.Errorf("未知答题模式: %s", modelName)
	}

	if btn := findAnswerEntryButton(page, sectionKeywords); btn != nil {
		humanPause(700, 1400)
		return clickAnswerActionHandle(btn)
	}

	if fallbackSelector != "" {
		handle, err := page.QuerySelector(fallbackSelector)
		if err == nil && handle != nil {
			humanPause(700, 1400)
			return clickAnswerActionHandle(handle)
		}
	}

	return fmt.Errorf("未找到%s入口", sectionName)
}

func ensureAnswerQuestionReady(page playwright.Page) error {
	if hasAnswerQuestion(page) {
		return nil
	}
	if waitForVisibleSelector(page, answerQuestionSelectors, 8, 500, 900) {
		return nil
	}
	return errors.New("答题页面未进入可作答状态")
}

func clickPreQuestionAction(page playwright.Page) error {
	if hasAnswerQuestion(page) {
		return nil
	}

	buttonSelectors := []string{
		`#app .detail-body .action-row button`,
		`#app .detail-body .action-row [role="button"]`,
		`.detail-body .action-row button`,
		`.detail-body .action-row [role="button"]`,
	}
	keywords := []string{"开始答题", "继续答题", "重新开始", "再来一组", "确定", "提交"}

	for _, selector := range buttonSelectors {
		btns, err := page.QuerySelectorAll(selector)
		if err != nil || len(btns) == 0 {
			continue
		}
		btn := pickAnswerActionButton(btns, keywords)
		if btn == nil {
			continue
		}
		humanPause(500, 1200)
		return clickAnswerActionHandle(btn)
	}

	return nil
}

func decodeSliderPosition(result interface{}) (float64, float64, float64, float64, error) {
	payload, ok := result.(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, errors.New("滑块位置数据格式异常")
	}
	if okValue, exists := payload["ok"].(bool); exists && !okValue {
		if reason, ok := payload["reason"].(string); ok && reason != "" {
			return 0, 0, 0, 0, errors.New(reason)
		}
		return 0, 0, 0, 0, errors.New("未找到滑块元素")
	}
	return floatFromEvalValue(payload["startX"]),
		floatFromEvalValue(payload["startY"]),
		floatFromEvalValue(payload["endX"]),
		floatFromEvalValue(payload["endY"]),
		nil
}

func findSliderInDocument(page playwright.Page) (interface{}, error) {
	return page.Evaluate(`() => {
		const isVisible = (el) => {
			if (!el) return false;
			const rect = el.getBoundingClientRect();
			if (rect.width <= 0 || rect.height <= 0) return false;
			const style = window.getComputedStyle(el);
			return style.display !== "none" && style.visibility !== "hidden" && style.opacity !== "0";
		};

		// ===== 阿里雲驗證碼 2.0（新版 aliyunCaptcha）=====
		// 先 dump 完整的滑塊 DOM 樹結構
		const v2Bind = document.querySelector("#JS_aliyun-captcha-slider_bind_element, .JS_aliyun-captcha-slider_bind_element");
		if (v2Bind && isVisible(v2Bind)) {
			// 遍歷所有子孫元素，找到真正的拖拽手柄和軌道
			const debugLog = [];
			const dumpChildren = (el, depth) => {
				const info = [];
				for (const child of el.children) {
					const r = child.getBoundingClientRect();
					const cls = (typeof child.className === 'string') ? child.className : '';
					const id = child.id || '';
					info.push({
						tag: child.tagName, id, cls: cls.substring(0, 80),
						w: Math.round(r.width), h: Math.round(r.height),
						l: Math.round(r.left), t: Math.round(r.top),
						kids: child.children.length
					});
					if (depth < 3 && child.children.length > 0) {
						info.push(...dumpChildren(child, depth + 1));
					}
				}
				return info;
			};

			const bindRect = v2Bind.getBoundingClientRect();
			debugLog.push("bind_element: w=" + Math.round(bindRect.width) + " h=" + Math.round(bindRect.height) + " l=" + Math.round(bindRect.left));

			const dump = dumpChildren(v2Bind, 0);
			for (const d of dump.slice(0, 30)) {
				debugLog.push(d.tag + " id=" + d.id + " cls=" + d.cls + " " + d.w + "x" + d.h + " @(" + d.l + "," + d.t + ") kids=" + d.kids);
			}

			// 策略：在 bind_element 內找最窄的可見元素作為拖拽手柄（>> 箭頭）
			// 軌道就是 bind_element 本身
			let handleEl = null;
			let handleRect = null;

			// 方法1：找 class 含 slider/btn/handle/icon 的子元素
			const handleCandidates = v2Bind.querySelectorAll("[class*='slider-btn'], [class*='slider-icon'], [class*='slide-btn'], [class*='icon'], [class*='handle']");
			for (const el of handleCandidates) {
				if (isVisible(el)) {
					const r = el.getBoundingClientRect();
					if (r.width > 0 && r.width < bindRect.width * 0.3) {
						handleEl = el;
						handleRect = r;
						debugLog.push("方法1找到手柄: cls=" + el.className + " w=" + Math.round(r.width));
						break;
					}
				}
			}

			// 方法2：找 aliyunCaptcha-sliding-slider 的子元素
			if (!handleEl) {
				const slidingSlider = v2Bind.querySelector("#aliyunCaptcha-sliding-slider, .aliyunCaptcha-sliding-slider");
				if (slidingSlider && isVisible(slidingSlider)) {
					// 看 slidingSlider 是手柄還是軌道 — 如果寬度 < bind 的一半就是手柄
					const ssRect = slidingSlider.getBoundingClientRect();
					if (ssRect.width < bindRect.width * 0.5) {
						handleEl = slidingSlider;
						handleRect = ssRect;
						debugLog.push("方法2: slidingSlider 是手柄 w=" + Math.round(ssRect.width));
					} else {
						// slidingSlider 是軌道，找其內部最左邊最小的子元素
						debugLog.push("方法2: slidingSlider 是軌道 w=" + Math.round(ssRect.width) + " 繼續找子元素");
						for (const child of slidingSlider.querySelectorAll("*")) {
							if (isVisible(child)) {
								const cr = child.getBoundingClientRect();
								if (cr.width > 5 && cr.width < ssRect.width * 0.3 && cr.left < ssRect.left + ssRect.width * 0.3) {
									handleEl = child;
									handleRect = cr;
									debugLog.push("方法2b: 找到軌道內手柄 tag=" + child.tagName + " cls=" + child.className + " w=" + Math.round(cr.width));
									break;
								}
							}
						}
					}
				}
			}

			// 方法3：暴力搜索 — bind_element 內最左最窄的可見子元素
			if (!handleEl) {
				let bestEl = null;
				let bestLeft = Infinity;
				for (const child of v2Bind.querySelectorAll("*")) {
					if (!isVisible(child)) continue;
					const cr = child.getBoundingClientRect();
					if (cr.width > 5 && cr.width < bindRect.width * 0.25 && cr.height > 10) {
						if (cr.left < bestLeft) {
							bestLeft = cr.left;
							bestEl = child;
						}
					}
				}
				if (bestEl) {
					handleEl = bestEl;
					handleRect = bestEl.getBoundingClientRect();
					debugLog.push("方法3暴力: tag=" + bestEl.tagName + " cls=" + bestEl.className + " w=" + Math.round(handleRect.width) + " l=" + Math.round(handleRect.left));
				}
			}

			if (handleEl && handleRect) {
				const startX = handleRect.left + handleRect.width / 2;
				const startY = handleRect.top + handleRect.height / 2;
				const endX = bindRect.left + bindRect.width - handleRect.width / 2 - 5;
				debugLog.push("最終: 起點(" + Math.round(startX) + "," + Math.round(startY) + ") 終點(" + Math.round(endX) + "," + Math.round(startY) + ") 距離=" + Math.round(endX - startX));
				return { ok: true, startX, startY, endX, endY: startY, debugLog };
			}

			// 所有方法都失敗，從 bind_element 左端拖到右端
			debugLog.push("所有方法均未找到手柄，使用 bind_element 左端");
			const startX = bindRect.left + 25;
			const startY = bindRect.top + bindRect.height / 2;
			const endX = bindRect.left + bindRect.width - 10;
			debugLog.push("備用: 起點(" + Math.round(startX) + "," + Math.round(startY) + ") 終點(" + Math.round(endX) + "," + Math.round(startY) + ") 距離=" + Math.round(endX - startX));
			return { ok: true, startX, startY, endX, endY: startY, debugLog };
		}

		// ===== 阿里雲 NoCaptcha（舊版）=====
		const ncHandleSelectors = [
			"#nc_1_n1z",
			"[id^='nc_'][id$='_n1z']",
			".nc_iconfont.btn_slide",
			".btn_slide",
			"[class*='btn_slide']",
		];
		const ncTrackSelectors = [
			"#nc_1_n1t",
			"[id^='nc_'][id$='_n1t']",
			".nc_scale",
			"[class*='nc_scale']",
		];
		let handleEl = null, handleRect = null;
		for (const sel of ncHandleSelectors) {
			for (const el of document.querySelectorAll(sel)) {
				if (isVisible(el)) {
					handleEl = el;
					handleRect = el.getBoundingClientRect();
					console.log("[滑塊調試] 找到 NC 滑塊按鈕: selector=" + sel + " left=" + handleRect.left + " width=" + handleRect.width);
					break;
				}
			}
			if (handleEl) break;
		}
		if (handleEl && handleRect) {
			let trackRect = null;
			for (const sel of ncTrackSelectors) {
				for (const el of document.querySelectorAll(sel)) {
					if (isVisible(el)) {
						trackRect = el.getBoundingClientRect();
						break;
					}
				}
				if (trackRect) break;
			}
			const startX = handleRect.left + handleRect.width / 2;
			const startY = handleRect.top + handleRect.height / 2;
			let endX;
			if (trackRect && trackRect.width > handleRect.width) {
				endX = trackRect.left + trackRect.width - handleRect.width / 2 - 5;
			} else {
				endX = startX + 260;
			}
			return { ok: true, startX, startY, endX, endY: startY };
		}

		// ===== 都沒找到，輸出調試信息 =====
		const debugSelector = '[id*="nc"], [id*="slide"], [id*="captcha"], [id*="aliyun"], [class*="nc"], [class*="slide"], [class*="captcha"], [class*="aliyun"], [class*="btn"]';
		const allEls = [];
		for (const el of document.querySelectorAll(debugSelector)) {
			const r = el.getBoundingClientRect();
			if (r.width > 0 && r.height > 0) {
				const id = el.id || '';
				const cls = (typeof el.className === 'string') ? el.className : '';
				allEls.push(el.tagName + '#' + id + '.' + cls.substring(0, 80));
			}
		}
		return { ok: false, reason: "未找到滑塊按鈕", debugElements: allEls.slice(0, 20).join(' | ') };
	}`)
}

func getAnswerSliderPosition(page playwright.Page) (float64, float64, float64, float64, error) {
	// 第一步：嘗試在主頁面查找滑塊
	log.Infoln("[答題] 嘗試在主頁面查找滑塊按鈕...")
	result, err := findSliderInDocument(page)
	if err == nil {
		// 輸出 JS 端的 debugLog 到 Go 日誌
		if m, ok := result.(map[string]interface{}); ok {
			if dl, ok := m["debugLog"]; ok {
				if arr, ok := dl.([]interface{}); ok {
					for _, line := range arr {
						log.Infoln("[滑塊DOM] ", line)
					}
				}
			}
		}
		x1, y1, x2, y2, decErr := decodeSliderPosition(result)
		if decErr == nil {
			return x1, y1, x2, y2, nil
		}
		// 主頁面找不到，記錄調試信息
		if m, ok := result.(map[string]interface{}); ok {
			if debugEls, ok := m["debugElements"]; ok {
				log.Infoln("[答題] 主頁面滑塊相關元素: ", debugEls)
			}
		}
	}

	// 第二步：遍歷所有 iframe 查找滑塊（阿里雲 NC 滑塊通常在 iframe 內）
	log.Infoln("[答題] 主頁面未找到，嘗試在 iframe 中查找滑塊...")
	frames := page.Frames()
	for i, frame := range frames {
		if frame == page.MainFrame() {
			continue
		}
		frameURL := frame.URL()
		log.Infoln("[答題] 檢查 iframe[", i, "]: ", frameURL)

		// 在 iframe 中執行查找
		frameResult, frameErr := frame.Evaluate(`() => {
			const handleSelectors = [
				"#JS_aliyun-captcha-slider_bind_element",
				".JS_aliyun-captcha-slider_bind_element",
				"[class*='aliyunCaptcha-sliding-slider']",
				"#aliyunCaptcha-sliding-slider",
				"#nc_1_n1z",
				"[id^='nc_'][id$='_n1z']",
				".nc_iconfont.btn_slide",
				".btn_slide",
				"[class*='btn_slide']",
				".nc-lang-cnt .btn_slide",
			];
			const trackSelectors = [
				"#aliyunCaptcha-sliding-slider",
				".aliyunCaptcha-sliding-slider",
				"[class*='aliyunCaptcha-sliding']",
				"#nc_1_n1t",
				"[id^='nc_'][id$='_n1t']",
				".nc_scale",
				".scale_text",
				"[class*='nc_scale']",
				".nc-lang-cnt",
			];
			const isVisible = (el) => {
				if (!el) return false;
				const rect = el.getBoundingClientRect();
				if (rect.width <= 0 || rect.height <= 0) return false;
				const style = window.getComputedStyle(el);
				return style.display !== "none" && style.visibility !== "hidden" && style.opacity !== "0";
			};
			let handleEl = null, handleRect = null;
			for (const sel of handleSelectors) {
				for (const el of document.querySelectorAll(sel)) {
					if (isVisible(el)) {
						handleEl = el;
						handleRect = el.getBoundingClientRect();
						break;
					}
				}
				if (handleEl) break;
			}
			if (!handleEl || !handleRect) {
				return { ok: false, reason: "iframe內未找到滑塊按鈕" };
			}
			let trackRect = null;
			for (const sel of trackSelectors) {
				for (const el of document.querySelectorAll(sel)) {
					if (isVisible(el)) {
						trackRect = el.getBoundingClientRect();
						break;
					}
				}
				if (trackRect) break;
			}
			return {
				ok: true,
				handleRect: { left: handleRect.left, top: handleRect.top, width: handleRect.width, height: handleRect.height },
				trackRect: trackRect ? { left: trackRect.left, top: trackRect.top, width: trackRect.width, height: trackRect.height } : null,
			};
		}`)
		if frameErr != nil {
			continue
		}

		frameMap, ok := frameResult.(map[string]interface{})
		if !ok {
			continue
		}
		okVal, _ := frameMap["ok"].(bool)
		if !okVal {
			continue
		}

		log.Infoln("[答題] 在 iframe[", i, "] 中找到滑塊按鈕！")

		// 獲取 iframe 在主頁面中的位置偏移
		iframeOffset, offsetErr := page.Evaluate(`(frameIndex) => {
			const iframes = document.querySelectorAll('iframe');
			let idx = 0;
			for (const iframe of iframes) {
				const rect = iframe.getBoundingClientRect();
				if (rect.width > 0 && rect.height > 0) {
					if (idx === frameIndex) {
						return { left: rect.left, top: rect.top };
					}
					idx++;
				}
			}
			// 備用：遍歷所有 iframe 找到匹配的
			for (const iframe of iframes) {
				const rect = iframe.getBoundingClientRect();
				if (rect.width > 0 && rect.height > 0) {
					return { left: rect.left, top: rect.top };
				}
			}
			return { left: 0, top: 0 };
		}`, i-1) // -1 因為跳過了 mainFrame
		if offsetErr != nil {
			log.Warningln("[答題] 獲取 iframe 偏移失敗，使用 0 偏移")
			iframeOffset = map[string]interface{}{"left": float64(0), "top": float64(0)}
		}

		offsetMap, _ := iframeOffset.(map[string]interface{})
		offsetLeft, _ := offsetMap["left"].(float64)
		offsetTop, _ := offsetMap["top"].(float64)
		log.Infoln("[答題] iframe 偏移: left=", offsetLeft, " top=", offsetTop)

		// 解析滑塊位置（iframe 內部座標）
		handleRectMap, _ := frameMap["handleRect"].(map[string]interface{})
		hLeft, _ := handleRectMap["left"].(float64)
		hTop, _ := handleRectMap["top"].(float64)
		hWidth, _ := handleRectMap["width"].(float64)
		hHeight, _ := handleRectMap["height"].(float64)

		// 轉換為主頁面座標 = iframe偏移 + iframe內部座標
		startX := offsetLeft + hLeft + hWidth/2
		startY := offsetTop + hTop + hHeight/2

		var endX, endY float64
		if trackRectRaw, ok := frameMap["trackRect"]; ok && trackRectRaw != nil {
			if trackRectMap, ok := trackRectRaw.(map[string]interface{}); ok {
				tLeft, _ := trackRectMap["left"].(float64)
				tWidth, _ := trackRectMap["width"].(float64)
				if tWidth > hWidth {
					endX = offsetLeft + tLeft + tWidth - hWidth/2 - 5
					endY = startY
					log.Infoln("[答題] iframe 內軌道寬度: ", tWidth)
				}
			}
		}
		if endX == 0 {
			endX = startX + 260
			endY = startY
		}

		log.Infoln("[答題] iframe 滑塊位置：起點(", startX, ",", startY, ") 終點(", endX, ",", endY, ")")
		return startX, startY, endX, endY, nil
	}

	return 0, 0, 0, 0, errors.New("未找到滑塊按鈕（主頁面和所有iframe均未找到）")
}

func easeOutCubic(t float64) float64 {
	t = t - 1
	return t*t*t + 1
}

func dragAnswerSlider(page playwright.Page, startX float64, startY float64, endX float64, endY float64) {
	distanceX := endX - startX

	// 1) 移動到滑塊起點附近
	humanPause(100, 200)
	page.Mouse().Move(startX, startY)

	// 2) 按下
	page.Mouse().Down()
	humanPause(50, 100)

	// 3) 簡化軌跡：5 個中間點（原版 20+ 個，保留必要的移動事件）
	type point struct {
		x, y  float64
		delay int
	}
	trail := []point{
		{startX + distanceX*0.15, startY + float64(rand2.Intn(3)-1), 15 + rand2.Intn(20)},
		{startX + distanceX*0.35, startY + float64(rand2.Intn(3)-1), 15 + rand2.Intn(20)},
		{startX + distanceX*0.55, startY + float64(rand2.Intn(5)-2), 20 + rand2.Intn(30)},
		{startX + distanceX*0.75, startY + float64(rand2.Intn(3)-1), 15 + rand2.Intn(20)},
		{endX, startY + float64(rand2.Intn(3)-1), 10 + rand2.Intn(15)},
	}

	// 4) 執行軌跡（必須有 Mouse.Move，否則滑塊不會移動）
	for _, p := range trail {
		page.Mouse().Move(p.x, p.y)
		time.Sleep(time.Duration(p.delay) * time.Millisecond)
	}

	// 5) 鬆手
	humanPause(50, 150)
	page.Mouse().Up()
}

func solveAnswerSlider(page playwright.Page) error {
	startX, startY, endX, endY, err := getAnswerSliderPosition(page)
	if err != nil {
		log.Errorln("[答題] 獲取滑塊位置失敗：", err.Error())
		return err
	}

	log.Infoln("[答題] 滑塊位置：起點(", startX, ",", startY, ") 終點(", endX, ",", endY, ")")
	log.Infoln("[答題] 開始拖動滑塊...")

	dragAnswerSlider(page, startX, startY, endX, endY)
	humanPause(600, 1200)
	return nil
}

// checkDailyScoreAndContinue 檢查每日答題積分，決定是否繼續答題
// 返回 true 表示需要繼續答題，false 表示可以退出
func (c *Core) checkDailyScoreAndContinue(page playwright.Page, user *model.User, score *Score, scoreRetryTimes int) bool {
	targetScore := 5 // 每日答題目標5分

	// 等待積分同步和答題流程冷卻（短等待足以避免「多端同時作答」限制）
	log.Infoln("[答題] 等待積分同步和答題流程冷卻...")
	humanPause(2000, 4000) // 優化：2-4秒足以，減少CPU/RAM占用

	// 獲取最新積分
	latestScore, scoreErr := getUserScoreWithRetry(user, scoreRetryTimes)
	if scoreErr != nil {
		log.Warningln("[答題] 獲取積分失敗，嘗試繼續答題: " + scoreErr.Error())
		return true
	}
	*score = latestScore

	currentScore := score.Content["daily"].CurrentScore
	maxScore := score.Content["daily"].MaxScore
	log.Infoln("[答題] 當前每日答題積分: ", currentScore, "/", maxScore)

	// 檢查是否已滿分
	if currentScore >= maxScore || currentScore >= targetScore {
		log.Infoln("[答題] 每日答題積分已滿，結束答題")
		return false
	}

	// 積分未滿，先返回積分頁面
	log.Infoln("[答題] 積分未滿，返回積分頁面準備下一輪答題")

	// 跳轉到積分頁面
	_, err := page.Goto(MyPointsUri, playwright.PageGotoOptions{
		Referer:   playwright.String("https://www.xuexi.cn/"),
		Timeout:   playwright.Float(15000),
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		log.Errorln("[答題] 跳轉積分頁面失敗: " + err.Error())
		return false
	}

	waitForVisibleSelector(page, []string{`#app .my-points-content`, `.my-points-content`, `#app .layout-body`}, 8, 300, 700)
	humanPause(3000, 5000) // 在積分頁面等待

	// 直接跳轉到每日答題頁面 (exam-practice.html)
	log.Infoln("[答題] 進入每日答題頁面...")
	_, err = page.Goto(DailyPracticeUri, playwright.PageGotoOptions{
		Referer:   playwright.String(MyPointsUri),
		Timeout:   playwright.Float(15000),
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		log.Errorln("[答題] 跳轉答題頁面失敗: " + err.Error())
		return false
	}

	humanPause(3000, 5000)

	// 檢測是否有"不要中途開啟新的答題流程"的提示
	pageText, _ := page.Evaluate(`() => document.body ? document.body.innerText || "" : ""`)
	if text, ok := pageText.(string); ok {
		normalizedText := strings.ReplaceAll(text, " ", "")
		normalizedText = strings.ReplaceAll(normalizedText, "\n", "")
		if strings.Contains(normalizedText, "请不要中途开启") ||
			strings.Contains(normalizedText, "不支持多端同时作答") ||
			strings.Contains(normalizedText, "答题流程") {
			log.Warningln("[答題] 檢測到「多端同時作答」限制提示，等待後重試")
			humanPause(3000, 5000) // 優化：3-5秒足以，避免長期goroutine堆積

			// 刷新頁面重試
			page.Reload(playwright.PageReloadOptions{
				Timeout:   playwright.Float(15000),
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			humanPause(3000, 5000)
		}
	}

	// 等待答題頁面加載
	waitForVisibleSelector(page, answerWorkspaceSelectors, 8, 500, 900)
	humanPause(2000, 3500)

	// 再次檢測是否有題目
	if err := ensureAnswerQuestionReady(page); err != nil {
		if isAnswerRoundComplete(page) {
			log.Infoln("[答題] 檢測到結果頁，可能上一輪未完成")
			return true // 繼續嘗試
		}
		log.Warningln("[答題] 答題頁面未就緒: " + err.Error())
	}

	log.Infoln("[答題] 已進入新一輪每日答題")
	return true
}

// 每日答题
// 新积分规则：只有每日答题，只需拿满5分
// 策略：完成一轮答题后退出，检查积分，未满5分则继续
func (c *Core) RespondDaily(user *model.User, modelName string) bool {

	var title string
	retryTimes := 0
	var id int

	// 专项答题已取消，直接返回
	if modelName == "special" {
		log.Infoln("[答題] 专项答题已取消，跳过")
		return true
	}

	// 捕获所有异常，防止程序崩溃
	defer func() {
		err := recover()
		if err != nil {
			log.Errorln("答题模块异常结束或答题已完成")
			c.Push(user.PushId, "text", "答题模块异常退出或答题已完成")
			log.Errorln(err)
		}
	}()
	// 判断浏览器是否被退出
	if c.IsQuit() {
		return false
	}

	// 建立登入狀態檢測器
	cfg := conf.GetConfig().Study
	authChecker := NewAuthChecker(AuthCheckerConfig{
		MaxFailures:       cfg.AuthFailureThreshold,
		CooldownPeriod:    time.Duration(cfg.FailureCooldownMinutes) * time.Minute,
		MaxSliderFailures: cfg.MaxSliderFailures,
		ModuleName:        "答題模組",
	})

	// 获取用户成绩
	score, err := getUserScoreWithRetry(user, cfg.ScoreRetryTimes)
	if err != nil {
		// 檢查是否為登入失效
		if _, ok := err.(*AuthError); ok || CheckAuthError(err) {
			log.Errorln("[答題] 獲取積分時檢測到登入失效: " + err.Error())
			c.Push(user.PushId, "text", "答題模組：登入已失效，請重新登入")
			return false
		}
		log.Errorln("获取分数失败，停止每日答题", err.Error())
		return false
	}

	// 每日答题目标分数（新规则只需5分）
	targetScore := 5
	if modelName == "daily" {
		// 记录当前得分，但不再提前退出，让答题流程完整执行
		log.Infoln("[答題] 每日答题当前得分:", score.Content["daily"].CurrentScore, "/ 目标:", targetScore)
	}

	// 创建浏览器上下文对象
	context, err := c.browser.NewContext()
	// 添加一个script,防止被检测
	_ = context.AddInitScript(playwright.Script{
		Content: playwright.String("Object.defineProperties(navigator, {webdriver:{get:()=>undefined}});")})
	if err != nil {
		log.Errorln("创建实例对象错误" + err.Error())
		return false
	}
	// 在退出方法时关闭对象
	defer func(context playwright.BrowserContext) {
		err := context.Close()
		if err != nil {
			log.Errorln("错误的关闭了实例对象" + err.Error())
		}
	}(context)
	// 创建一个页面
	page, err := context.NewPage()
	if err != nil {
		log.Errorln("创建页面失败" + err.Error())
		return false
	}
	// 退出时关闭页面
	defer func() {
		page.Close()
	}()
	// 添加用户的cookie
	err = context.AddCookies(user.ToBrowserCookies())
	if err != nil {
		log.Errorln("添加cookie失败" + err.Error())
		return false
	}
	// 跳转到积分页面
	_, err = page.Goto(MyPointsUri, playwright.PageGotoOptions{
		Referer:   playwright.String(MyPointsUri),
		Timeout:   playwright.Float(10000),
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		log.Errorln("跳转页面失败" + err.Error())
		return false
	}
	waitForVisibleSelector(page, []string{`#app .my-points-content`, `.my-points-content`, `#app .layout-body`}, 8, 300, 700)
	humanPause(400, 800)
	log.Infoln("已加载答题模块")
	// 判断答题类型，然后相应处理
	switch modelName {
	case "daily":
		{
			// 检测是否已经完成
			if score.Content["daily"].CurrentScore >= score.Content["daily"].MaxScore {
				log.Infoln("检测到每日答题已经完成，即将退出答题")

				return true
			}
			// 点击每日答题的按钮
			err = openAnswerSection(page, modelName)
			if err != nil {
				log.Errorln("跳转到积分页面错误" + err.Error())

				return false
			}
			c.Push(user.PushId, "text", "已加载每日答题模块")
		}
	case "weekly":
		{
			// 检测是否已经完成
			if score.Content["weekly"].CurrentScore >= score.Content["weekly"].MaxScore {
				log.Infoln("检测到每周答题已经完成，即将退出答题")

				return true
			}
			// err = page.Click(WEEKEND)
			// if err != nil {
			//	log.Errorln("跳转到积分页面错误")
			//	return
			//}

			// 获取每周答题的ID
			id, err = getweekID(user.ToCookies())
			if err != nil {
				return false
			}
			// 跳转到每周答题界面
			_, err = page.Goto(fmt.Sprintf("https://pc.xuexi.cn/points/exam-weekly-detail.html?id=%d", id), playwright.PageGotoOptions{
				Referer:   playwright.String(MyPointsUri),
				Timeout:   playwright.Float(10000),
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			if err != nil {
				log.Errorln("跳转到答题页面错误" + err.Error())
				return false
			}
			c.Push(user.PushId, "text", "已加载每周答题模块")
		}
	case "special":
		{
			//获取专项答题ID
			id, err = getSpecialID(user.ToCookies())
			if err != nil {
				return false
			}
			// id = 77
			// 跳转到专项答题界面
			_, err = page.Goto(fmt.Sprintf("https://pc.xuexi.cn/points/exam-paper-detail.html?id=%d", id), playwright.PageGotoOptions{
				Referer:   playwright.String(MyPointsUri),
				Timeout:   playwright.Float(10000),
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			if err != nil {
				log.Errorln("跳转到答题页面错误" + err.Error())
				return false
			}
			c.Push(user.PushId, "text", "已加载专项答题模块")
		}
	}
	waitForVisibleSelector(page, answerWorkspaceSelectors, 8, 300, 600)
	humanPause(800, 1400)
	if err := ensureAnswerQuestionReady(page); err != nil {
		if isAnswerRoundComplete(page) {
			log.Infoln("[答題] 檢測到結果頁，本輪答題結束")
			return true
		}
		log.Debugln("[答題] 首次進入題目頁時尚未就緒: ", err.Error())
	}
	// 跳转到答题页面，若返回true则说明已答完
	// if getAnswerPage(page, model) {
	//	return
	//}

	tryCount := 0
	for {
	label:
		tryCount++
		if tryCount >= 30 {
			log.Errorln("[答題] 多次循環嘗試答題，已超出30次，自動退出")
			c.Push(user.PushId, "text", "答題模組：嘗試次數過多，已自動退出")
			return false
		}

		// 檢查是否應該提前停止（連續失敗或滑塊失敗過多）
		if authChecker.ShouldStop() && cfg.FastFailEnabled {
			log.Errorln("[答題] 連續失敗次數過多，提前停止答題")
			c.Push(user.PushId, "text", "答題模組：連續失敗次數過多，已提前停止")
			return false
		}

		if authChecker.ShouldStopSlider() && cfg.FastFailEnabled {
			log.Errorln("[答題] 滑塊驗證失敗次數過多，提前停止答題")
			c.Push(user.PushId, "text", "答題模組：滑塊驗證失敗次數過多，請手動處理")
			return false
		}

		if c.IsQuit() {
			return false
		}
		if err := clickPreQuestionAction(page); err != nil {
			log.Warningln("[答題] 预处理按钮点击失败: " + err.Error())
		}
		waitForVisibleSelector(page, answerQuestionSelectors, 4, 400, 800)
		if hasAnswerSliderPrompt(page) {
			if !c.handleAnswerSliderChallenge(page, user, authChecker, "答題頁面") {
				return false
			}
			humanPause(600, 1000)
			goto label
		}
		if err := ensureAnswerQuestionReady(page); err != nil {
			if isAnswerRoundComplete(page) {
				log.Infoln("[答題] 檢測到結果頁，本輪答題結束")
				return true
			}
			log.Debugln("[答題] 題目區域暫未就緒，繼續等待重試: ", err.Error())
			humanPause(500, 1000)
			continue
		}
		switch modelName {
		case "daily":
			{
				// 记录当前得分，不再提前退出
				currentScore := score.Content["daily"].CurrentScore
				log.Infoln("[答題] 继续答题，当前得分:", currentScore, "/ 目标:", targetScore)
			}
		case "weekly":
			{
				// 检测是否已经完成
				if score.Content["weekly"].CurrentScore >= score.Content["weekly"].MaxScore {
					log.Infoln("检测到每周答题已经完成，即将退出答题")

					return true
				}
			}
		}

		// 获取题目类型
		category, err := page.QuerySelector(
			`#app > div > div.layout-body > div > div.detail-body > div.question > div.q-header`)
		if err != nil {
			log.Errorln("没有找到题目元素" + err.Error())

			return false
		}
		if category != nil {
			_ = category.WaitForElementState(`visible`)
			humanPause(300, 600)

			// 获取题目
			question, err := page.QuerySelector(
				`#app > div > div.layout-body > div > div.detail-body > div.question > div.q-body > div`)
			if err != nil {
				log.Errorln("未找到题目问题元素")

				return false
			}
			// 获取题目类型
			categoryText, err := category.TextContent()
			if err != nil {
				log.Errorln("获取题目元素失败" + err.Error())

				return false
			}
			log.Infoln("## 题目类型：" + categoryText)

			// 获取题目的问题
			questionText, err := question.TextContent()
			if err != nil {
				log.Errorln("获取题目问题失败" + err.Error())
				return false
			}

			log.Infoln("## 题目：" + questionText)
			if title == questionText {
				log.Warningln("可能已经卡住，正在重试，重试次数+1")
				retryTimes++
			} else {
				retryTimes = 0
			}
			title = questionText

			// 如果是答错后的重试，尝试点击继续按钮或刷新页面
			if retryTimes > 0 {
				log.Infoln("[答错重试] 检测到答错后重试，尝试进入下一题")

				// 先尝试点击"继续答题"、"下一题"、"确定"等按钮
				continueKeywords := []string{"继续答题", "继续", "下一题", "确定", "查看解析", "关闭"}
				buttonSelectors := []string{
					`#app .action-row > button`,
					`#app .action-row [role="button"]`,
					`.action-row button`,
					`button.ant-btn`,
					`.ant-btn`,
					`button`,
					`[role="button"]`,
				}

				clicked := false
				for _, selector := range buttonSelectors {
					btns, btnErr := page.QuerySelectorAll(selector)
					if btnErr != nil || len(btns) == 0 {
						continue
					}
					btn := pickAnswerActionButton(btns, continueKeywords)
					if btn != nil {
						if clickErr := clickAnswerActionHandle(btn); clickErr == nil {
							log.Infoln("[答错重试] 已点击继续按钮")
							clicked = true
							if advanceErr := waitForAnswerAdvance(page, questionText, buttonSelectors); advanceErr != nil {
								if advanceErr == ErrAnswerComplete {
									log.Infoln("[答題] 本輪答題已完成")
									return true
								}
								log.Warningln("[答错重试] 继续流程巡航失败：", advanceErr.Error())
							}
							break
						}
					}
				}

				// 如果没点到按钮，尝试直接获取选项并选择
				if !clicked {
					options, optErr := getOptions(page)
					if optErr == nil && len(options) > 0 {
						log.Infoln("[答错重试] 可选选项：", options)
						var randomAnswer []string
						if strings.Contains(categoryText, "多选题") {
							randomAnswer = options // 多选题全选
						} else {
							rand2.Seed(time.Now().UnixNano())
							randomAnswer = []string{options[rand2.Intn(len(options))]}
						}
						log.Infoln("[答错重试] 选择：", randomAnswer)
						if radioErr := radioCheck(page, questionText, randomAnswer); radioErr != nil {
							if radioErr == ErrAnswerComplete {
								log.Infoln("[答題] 本輪答題已完成")
								return true
							}
							log.Errorln("[答错重试] 选择失败：", radioErr.Error())
						}
						humanPause(800, 1500)
					} else {
						// 既没有继续按钮也没有选项，可能需要刷新页面
						log.Infoln("[答错重试] 无法找到按钮或选项，尝试刷新页面")
						page.Reload()
						humanPause(600, 1000)
					}
				}
				continue
			}

			// 获取答题帮助 - 尝试多种选择器
			var openTips playwright.ElementHandle
			var tipsFound bool

			// 提示按钮的多种可能选择器
			tipSelectors := []string{
				`#app > div > div.layout-body > div > div.detail-body > div.question > div.q-footer > span`,
				`.q-footer span`,
				`div.q-footer span`,
				`.question .q-footer span`,
				`div.question div.q-footer span`,
				`span[class*="tips"]`,
				`button[class*="tips"]`,
				`.tips-btn`,
			}

			for _, selector := range tipSelectors {
				openTips, err = page.QuerySelector(selector)
				if err == nil && openTips != nil {
					log.Debugln("使用选择器找到提示按钮: ", selector)
					tipsFound = true
					break
				}
			}

			if !tipsFound || openTips == nil {
				log.Errorln("未获取到题目提示信息按钮，嘗試備選方案")

				// 当无法获取提示时，尝试从题库搜索答案
				if len(questionText) > 20 {
					log.Infoln("[備選方案] 嘗試從題庫搜索答案")
					searchAnswer := model.SearchAnswer(questionText)
					if searchAnswer != "" {
						log.Infoln("[題庫] 找到答案: ", searchAnswer)
					}
				}

				// 如果题库也没答案，随机选择并提交
				options, optErr := getOptions(page)
				if optErr == nil && len(options) > 0 {
					log.Infoln("[無提示] 随机选择答案")
					var randomAnswer []string
					if strings.Contains(categoryText, "多选题") {
						// 多选题：选择全部选项
						randomAnswer = options
					} else {
						// 单选题：选择第一个
						randomAnswer = []string{options[0]}
					}
					log.Infoln("[無提示] 选择：", randomAnswer)
					if radioErr := radioCheck(page, questionText, randomAnswer); radioErr != nil {
						if radioErr == ErrAnswerComplete {
							log.Infoln("[答題] 本輪答題已完成")
							return true
						}
						log.Errorln("[無提示] 选择失败：", radioErr.Error())
					}
				}
				humanPause(600, 1200)
				tryCount++
				continue
			}

			log.Debugln("开始尝试获取打开提示信息按钮")
			// 点击提示的按钮
			err = humanClick(openTips)
			if err != nil {
				log.Errorln("点击打开提示信息按钮失败" + err.Error())
				tryCount++
				continue
			}

			// 等待提示内容加载
			log.Debugln("已点击提示按钮，等待内容加载...")
			humanPause(800, 1400)

			// 尝试等待红字提示出现
			_, err = page.WaitForSelector(`font[color="red"]`, playwright.PageWaitForSelectorOptions{
				Timeout: playwright.Float(5000),
			})
			if err != nil {
				log.Debugln("等待红字提示超时，继续获取页面内容")
			} else {
				log.Debugln("检测到红字提示已加载")
			}

			// 获取页面内容
			content, err := page.Content()
			if err != nil {
				log.Errorln("获取网页全体内容失败" + err.Error())
				tryCount++
				continue
			}

			// 额外等待确保内容完整
			humanPause(500, 1000)
			log.Debugln("已获取网页内容")

			// 关闭提示信息
			err = humanClick(openTips)
			if err != nil {
				log.Errorln("点击关闭提示信息按钮失败" + err.Error())
				// 关闭失败不影响继续答题，尝试继续
			}
			log.Debugln("已关闭提示信息")
			// 从整个页面内容获取提示信息
			tips := getTips(content)
			log.Infoln("[提示信息]：", tips)

			if retryTimes > 4 {
				log.Warningln("重试次数太多，即将退出答题")
				options, _ := getOptions(page)
				c.Push(user.PushId, "flush", fmt.Sprintf(
					"答题过程出现异常！！</br>答题渠道：%v</br>题目ID:%v</br>题目类型：%v</br>题目：%v</br>题目选项：%v</br>提示信息：%v</br>", modelName, id, categoryText, questionText, strings.Join(options, " "), strings.Join(tips, " ")))
				return false
			}

			// 填空题
			var answerErr error
			switch {
			case strings.Contains(categoryText, "填空题"):

				// 填充填空题
				answerErr = FillBlank(page, questionText, tips)
			case strings.Contains(categoryText, "多选题"):
				log.Infoln("读取到多选题")
				options, err := getOptions(page)
				if err != nil {
					log.Errorln("获取选项失败" + err.Error())
					return false
				}
				log.Infoln("获取到选项答案：", options)
				log.Infoln("[多选题选项]：", options)
				answer := pickSelectableAnswers(options, tips)

				if len(answer) < 1 {
					answer = append(answer, options...)
					log.Infoln("无法判断答案，自动选择ABCD")
				}
				log.Infoln("根据提示分别选择了", answer)
				// 多选题选择
				answerErr = radioCheck(page, questionText, answer)
			case strings.Contains(categoryText, "单选题"):
				log.Infoln("读取到单选题")
				options, err := getOptions(page)
				if err != nil {
					log.Errorln("获取选项失败" + err.Error())
					return false
				}
				log.Infoln("获取到选项答案：", options)

				var answer []string

				if len(tips) > 1 {
					log.Warningln("检测到单选题出现多个提示信息，即将对提示信息进行合并")
					tip := strings.Join(tips, "")
					tips = []string{tip}
				}

				answer = selectSingleChoiceAnswers(questionText, options, tips)
				if len(answer) < 1 {
					answer = append(answer, options[0])
					log.Infoln("无法判断答案，自动选择A")
				}

				log.Infoln("根据提示分别选择了", answer)
				answerErr = radioCheck(page, questionText, answer)
			}

			// 检测答题是否完成
			if answerErr == ErrAnswerComplete {
				log.Infoln("[答題] 本輪答題已完成")
				// 對於每日答題，完成一輪後檢查積分，決定是否繼續
				if modelName == "daily" {
					if c.checkDailyScoreAndContinue(page, user, &score, cfg.ScoreRetryTimes) {
						continue // 積分未滿，繼續新一輪答題
					}
				}
				return true
			}
			if errors.Is(answerErr, ErrAnswerSliderChallenge) {
				if !c.handleAnswerSliderChallenge(page, user, authChecker, "提交後") {
					return false
				}

				// 滑塊通過後，等待頁面狀態穩定
				humanPause(600, 1000)

				if isAnswerRoundComplete(page) {
					log.Infoln("[答題] 檢測到結果頁，本輪答題結束")
					// 對於每日答題，完成一輪後檢查積分，決定是否繼續
					if modelName == "daily" {
						if c.checkDailyScoreAndContinue(page, user, &score, cfg.ScoreRetryTimes) {
							continue // 積分未滿，繼續新一輪答題
						}
					}
					return true
				}

				// 滑塊可能攔截了提交請求，需要重新選擇答案並提交
				log.Infoln("[答題] 滑塊通過後，嘗試重新提交答案...")

				// 檢測是否還在當前題目頁面（有可選擇的選項）
				currentOptions, optionsErr := getOptions(page)
				if optionsErr == nil && len(currentOptions) > 0 {
					log.Infoln("[答題] 檢測到仍在當前題目，重新選擇答案")

					// 根據題目類型重新選擇答案
					if strings.Contains(categoryText, "多选题") {
						answer := pickSelectableAnswers(currentOptions, tips)
						if len(answer) < 1 {
							answer = currentOptions
						}
						log.Infoln("[答題] 滑塊後重新選擇多選題答案：", answer)
						answerErr = radioCheck(page, questionText, answer)
					} else if strings.Contains(categoryText, "单选题") {
						answer := selectSingleChoiceAnswers(questionText, currentOptions, tips)
						if len(answer) < 1 {
							answer = []string{currentOptions[0]}
						}
						log.Infoln("[答題] 滑塊後重新選擇單選題答案：", answer)
						answerErr = radioCheck(page, questionText, answer)
					} else if strings.Contains(categoryText, "填空题") {
						log.Infoln("[答題] 滑塊後重新填寫答案")
						answerErr = FillBlank(page, questionText, tips)
					}

					// 檢查重新提交後的結果
					if answerErr == ErrAnswerComplete {
						log.Infoln("[答題] 重新提交後答題完成")
						if modelName == "daily" {
							if c.checkDailyScoreAndContinue(page, user, &score, cfg.ScoreRetryTimes) {
								continue
							}
						}
						return true
					}
					if errors.Is(answerErr, ErrAnswerSliderChallenge) {
						// 又出現滑塊，遞歸處理
						log.Warningln("[答題] 重新提交後又出現滑塊，繼續處理")
						continue
					}
					if answerErr != nil {
						log.Warningln("[答題] 重新提交失敗：", answerErr.Error())
					}
				} else {
					// 沒有選項，可能是已經跳轉到下一題或結果頁
					log.Infoln("[答題] 滑塊通過後沒有檢測到選項，檢查頁面狀態")

					if isAnswerRoundComplete(page) {
						log.Infoln("[答題] 滑塊通過後檢測到結果頁，本輪答題結束")
						if modelName == "daily" {
							if c.checkDailyScoreAndContinue(page, user, &score, cfg.ScoreRetryTimes) {
								continue
							}
						}
						return true
					}

					// 嘗試點擊繼續按鈕
					buttonSelectors := []string{
						`#app .action-row > button`,
						`.action-row button`,
						`button.ant-btn`,
						`button`,
					}
					keywords := []string{"下一题", "确定", "提交", "完成", "确认", "继续"}

					waitForVisibleSelector(page, buttonSelectors, 3, 300, 600)
					for _, selector := range buttonSelectors {
						btns, btnErr := page.QuerySelectorAll(selector)
						if btnErr != nil || len(btns) == 0 {
							continue
						}
						btn := pickAnswerActionButton(btns, keywords)
						if btn != nil {
							btnText, _ := btn.TextContent()
							btnText = strings.TrimSpace(btnText)
							if clickErr := clickAnswerActionHandle(btn); clickErr == nil {
								log.Infoln("[答題] 滑塊通過後點擊按鈕：", btnText)
								humanPause(1000, 2000)
								break
							}
						}
					}
				}

				log.Infoln("[答題] 提交後的滑塊驗證已通過，等待加載下一題")
				humanPause(600, 1000)
				continue
			}

			if answerErr != nil {
				log.Errorln("答题操作失败" + answerErr.Error())
				if !authChecker.RecordFailure(answerErr) {
					log.Errorln("[答題] 連續失敗次數過多，停止答題")
					return false
				}
				humanPause(800, 1500)
				continue
			}
			authChecker.Reset()
		}

		// 等待服務器積分同步
		humanPause(800, 1500)
		latestScore, scoreErr := getUserScoreWithRetry(user, cfg.ScoreRetryTimes)
		if scoreErr != nil {
			if _, ok := scoreErr.(*AuthError); ok || CheckAuthError(scoreErr) {
				log.Errorln("[答題] 獲取積分時檢測到登入失效: " + scoreErr.Error())
				c.Push(user.PushId, "text", "答題模組：登入已失效，請重新登入")
				return false
			}
			log.Warningln("[答題] 本轮答题后刷新积分失败: " + scoreErr.Error())
			continue
		}
		score = latestScore
	}
}

func GetAnswerPage(page playwright.Page, model string) bool {
	selectPages, err := page.QuerySelectorAll(`#app .ant-pagination .ant-pagination-item`)
	if err != nil {
		log.Errorln("获取到页码失败")

		return false
	}
	log.Infoln("共获取到", len(selectPages), "页")
	modelName := ""
	modelSlector := ""
	switch model {
	case "daily":
		return false
	case "weekly":
		modelName = "每周答题"
		modelSlector = "button.ant-btn-primary"
	case "special":
		modelName = "专项答题"
		modelSlector = "#app .items .item button"
	}
	for i := 1; i <= len(selectPages); i++ {
		log.Infoln("获取到"+modelName, "第", i, "页")
		err1 := humanClick(selectPages[i-1])
		if err1 != nil {
			log.Errorln("点击页码失败")
		}
		humanPause(400, 800)
		datas, err := page.QuerySelectorAll(modelSlector)
		if err != nil {
			log.Errorln("获取页面内容失败")
			continue
		}
		for _, data := range datas {
			content, err := data.TextContent()
			if err != nil {
				log.Errorln("获取按钮文本失败" + err.Error())
				continue
			}
			if strings.Contains(content, "重新") || strings.Contains(content, "满分") {
				continue
			} else {
				if strings.Contains(content, "电影试题") {
					log.Infoln("发现有未答题的电影试题")
					continue
				}
				enabled, err := data.IsEnabled()
				if err != nil {
					return false
				}
				if enabled {
					log.Infoln("按钮可用")
				}
				data.WaitForElementState("stable", playwright.ElementHandleWaitForElementStateOptions{Timeout: playwright.Float(10000)})
				humanPause(3000, 5200)
				dblBox, _ := data.BoundingBox()
				var dblPos *playwright.Position
				if dblBox != nil && dblBox.Width > 1 && dblBox.Height > 1 {
					dblPos = &playwright.Position{
						X: dblBox.Width*0.2 + float64(rand2.Intn(int(dblBox.Width*0.6)+1)),
						Y: dblBox.Height*0.2 + float64(rand2.Intn(int(dblBox.Height*0.6)+1)),
					}
				}
				err = data.Click(playwright.ElementHandleClickOptions{
					ClickCount: playwright.Int(2),
					Position:   dblPos,
					Timeout:    playwright.Float(100000),
				})
				if err != nil {
					log.Errorln("点击按钮失败" + err.Error())
					humanPause(400, 800)
					continue
				}
				humanPause(1800, 3200)
				return false
			}
		}
	}
	log.Infoln("检测到每周答题已经完成")
	return true
}

func radioCheck(page playwright.Page, questionText string, answer []string) error {
	radios, err := page.QuerySelectorAll(`.q-answer.choosable`)
	if err != nil {
		log.Errorln("获取选项失败")
		return err
	}
	radios = filterVisibleAnswerHandles(radios)
	if len(radios) == 0 {
		log.Warningln("[答題] 未找到可選選項")
		return checkNextBotton(page, questionText)
	}

	normalizedAnswer := make(map[string]struct{}, len(answer))
	for _, item := range answer {
		normalized := normalizeAnswerButtonText(item)
		if normalized != "" {
			normalizedAnswer[normalized] = struct{}{}
		}
	}
	log.Debugln("获取到", len(radios), "个按钮")

	// 快速閱讀題目
	humanPause(400, 800)

	// 嘗試找到匹配的答案
	found := false
	for _, radio := range radios {
		textContent, err := radio.TextContent()
		if err != nil {
			continue
		}
		if _, ok := normalizedAnswer[normalizeAnswerButtonText(textContent)]; ok {
			// 找到匹配的答案，點擊
			if err := humanClick(radio); err == nil {
				log.Infoln("[答題] 選擇匹配答案：", strings.TrimSpace(textContent))
				found = true
				break
			}
		}
	}

	// 如果沒找到匹配的答案，隨機選擇第一個選項
	if !found {
		if err := humanClick(radios[0]); err == nil {
			text, _ := radios[0].TextContent()
			log.Infoln("[答題] 未找到匹配答案，隨機選擇：", strings.TrimSpace(text))
		}
	}

	// 快速確認
	humanPause(300, 600)

	return checkNextBotton(page, questionText)
}

// 获取选项
func getOptions(page playwright.Page) ([]string, error) {
	handles, err := page.QuerySelectorAll(`.q-answer.choosable`)
	if err != nil {
		log.Errorln("获取选项信息失败")
		return nil, err
	}
	handles = filterVisibleAnswerHandles(handles)
	var options []string
	for _, handle := range handles {
		content, err := handle.TextContent()
		if err != nil {
			return nil, err
		}
		options = append(options, content)
	}
	return options, err
}

// 获取问题提示
// 支持多种HTML格式的红字提示提取
func getTips(data string) []string {
	data = strings.ReplaceAll(data, " ", "")
	data = strings.ReplaceAll(data, "\n", "")

	var tips []string

	// 尝试多种正则模式匹配红字提示
	patterns := []string{
		`<fontcolor="red">(.*?)</font>`,        // 标准格式
		`<fontcolor='red'>(.*?)</font>`,        // 单引号格式
		`<fontcolor=red>(.*?)</font>`,          // 无引号格式
		`<spanclass="red">(.*?)</span>`,        // span标签格式
		`<spanstyle="color:red">(.*?)</span>`,  // style格式
		`<spanstyle="color:red;">(.*?)</span>`, // style带分号
		`class="answer-tip"[^>]*>([^<]+)<`,     // 答案提示class
	}

	for _, pattern := range patterns {
		compile := regexp.MustCompile(pattern)
		match := compile.FindAllStringSubmatch(data, -1)
		for _, i := range match {
			if len(i) > 1 && i[1] != "" {
				// 清理提取的内容
				tip := strings.TrimSpace(i[1])
				if tip != "" && !containsTip(tips, tip) {
					tips = append(tips, tip)
				}
			}
		}
	}

	// 如果上述模式都没有匹配到，尝试从提示区域提取文本
	if len(tips) == 0 {
		// 尝试匹配提示区域的文本内容
		tipAreaPattern := regexp.MustCompile(`class="[^"]*tips?[^"]*"[^>]*>([^<]+)<`)
		matches := tipAreaPattern.FindAllStringSubmatch(data, -1)
		for _, m := range matches {
			if len(m) > 1 && m[1] != "" {
				tip := strings.TrimSpace(m[1])
				if tip != "" && len(tip) > 2 && !containsTip(tips, tip) {
					tips = append(tips, tip)
				}
			}
		}
	}

	if len(tips) == 0 {
		log.Warningln("检测到未获取到提示信息")
	} else {
		log.Infoln("成功提取到", len(tips), "条提示信息")
	}

	return tips
}

// 检查提示是否已存在（去重用）
func containsTip(tips []string, tip string) bool {
	for _, t := range tips {
		if t == tip {
			return true
		}
	}
	return false
}

func normalizeAnswerButtonText(text string) string {
	replacer := strings.NewReplacer(
		" ", "",
		"\n", "",
		"\t", "",
		"\r", "",
		" ", "",
	)
	return replacer.Replace(strings.TrimSpace(text))
}

func filterVisibleAnswerHandles(handles []playwright.ElementHandle) []playwright.ElementHandle {
	visible := make([]playwright.ElementHandle, 0, len(handles))
	for _, handle := range handles {
		ok, err := handle.IsVisible()
		if err != nil || !ok {
			continue
		}
		visible = append(visible, handle)
	}
	return visible
}

func pickAnswerActionButton(handles []playwright.ElementHandle, keywords []string) playwright.ElementHandle {
	normalizedKeywords := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		normalized := normalizeAnswerButtonText(keyword)
		if normalized != "" {
			normalizedKeywords = append(normalizedKeywords, normalized)
		}
	}

	for _, handle := range handles {
		ok, err := handle.IsVisible()
		if err != nil || !ok {
			continue
		}

		enabled, err := handle.IsEnabled()
		if err == nil && !enabled {
			continue
		}

		text, err := handle.TextContent()
		if err != nil {
			continue
		}
		normalizedText := normalizeAnswerButtonText(text)
		if normalizedText == "" {
			continue
		}
		for _, keyword := range normalizedKeywords {
			if strings.Contains(normalizedText, keyword) {
				return handle
			}
		}
	}
	return nil
}

// humanClick 在元素 BoundingBox 內隨機偏移 ±30% 範圍點擊，模擬真人點擊位置不精確
func humanClick(el playwright.ElementHandle) error {
	box, err := el.BoundingBox()
	if err != nil || box == nil || box.Width < 1 || box.Height < 1 {
		return el.Click()
	}
	offsetX := box.Width*0.2 + float64(rand2.Intn(int(box.Width*0.6)+1))
	offsetY := box.Height*0.2 + float64(rand2.Intn(int(box.Height*0.6)+1))
	return el.Click(playwright.ElementHandleClickOptions{
		Position: &playwright.Position{X: offsetX, Y: offsetY},
	})
}

func clickAnswerActionHandle(handle playwright.ElementHandle) error {
	if handle == nil {
		return errors.New("未找到可点击元素")
	}
	_ = handle.WaitForElementState("visible", playwright.ElementHandleWaitForElementStateOptions{
		Timeout: playwright.Float(5000),
	})
	_ = handle.WaitForElementState("stable", playwright.ElementHandleWaitForElementStateOptions{
		Timeout: playwright.Float(5000),
	})

	// 獲取元素位置，模擬鼠標移動軌跡
	box, err := handle.BoundingBox()
	if err == nil && box != nil {
		// 先移動到元素附近（不是直接到元素上）
		randomOffsetX := float64(rand2.Intn(100) - 50)
		randomOffsetY := float64(rand2.Intn(80) - 40)
		_ = handle.Hover(playwright.ElementHandleHoverOptions{
			Timeout: playwright.Float(3000),
			Position: &playwright.Position{
				X: box.Width/2 + randomOffsetX,
				Y: box.Height/2 + randomOffsetY,
			},
		})

		// 短暫停頓，模擬猶豫
		humanPause(300, 800)

		// 再移動到元素中心附近（帶隨機偏移）
		clickX := box.Width*0.2 + float64(rand2.Intn(maxInt(int(box.Width*0.6), 1)))
		clickY := box.Height*0.2 + float64(rand2.Intn(maxInt(int(box.Height*0.6), 1)))
		_ = handle.Hover(playwright.ElementHandleHoverOptions{
			Timeout:  playwright.Float(3000),
			Position: &playwright.Position{X: clickX, Y: clickY},
		})
	} else {
		_ = handle.Hover(playwright.ElementHandleHoverOptions{
			Timeout: playwright.Float(3000),
		})
	}

	// 添加點擊前的停頓（模擬思考）
	humanPause(500, 1200)

	var clickPos *playwright.Position
	if box != nil {
		clickPos = &playwright.Position{
			X: box.Width*0.2 + float64(rand2.Intn(maxInt(int(box.Width*0.6), 1))),
			Y: box.Height*0.2 + float64(rand2.Intn(maxInt(int(box.Height*0.6), 1))),
		}
	}

	err = handle.Click(playwright.ElementHandleClickOptions{
		Timeout:  playwright.Float(10000),
		Position: clickPos,
	})
	if err == nil {
		return nil
	}

	return handle.Click(playwright.ElementHandleClickOptions{
		Timeout: playwright.Float(10000),
		Force:   playwright.Bool(true),
	})
}

func buildTipCandidates(tips []string) []string {
	replacer := strings.NewReplacer("（", "", "）", "", "(", "", ")", "", "“", "", "”", "", "\"", "")
	splitter := regexp.MustCompile(`[，、,；;|/ ]+`)
	seen := make(map[string]struct{})
	candidates := make([]string, 0, len(tips))
	appendCandidate := func(value string) {
		value = strings.TrimSpace(replacer.Replace(value))
		if value == "" {
			return
		}
		key := normalizeAnswerButtonText(value)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, value)
	}

	for _, tip := range tips {
		appendCandidate(tip)
		for _, part := range splitter.Split(tip, -1) {
			appendCandidate(part)
		}
	}

	return candidates
}

func matchSelectableAnswers(options []string, tips []string) []string {
	candidates := buildTipCandidates(tips)
	matches := make([]string, 0, len(options))
	seen := make(map[string]struct{})
	for _, option := range options {
		optionKey := normalizeAnswerButtonText(cleanSelectableAnswerText(option))
		if optionKey == "" {
			continue
		}
		for _, candidate := range candidates {
			candidateKey := normalizeAnswerButtonText(cleanSelectableAnswerText(candidate))
			if candidateKey == "" {
				continue
			}
			if optionKey != candidateKey {
				continue
			}
			if _, ok := seen[optionKey]; ok {
				break
			}
			seen[optionKey] = struct{}{}
			matches = append(matches, option)
			break
		}
	}
	return matches
}

func pickSelectableAnswers(options []string, tips []string) []string {
	matches := matchSelectableAnswers(options, tips)

	joinedTipKey := normalizeSemanticAnswerText(strings.Join(tips, ""))
	if joinedTipKey != "" {
		containsMatches := make([]string, 0, len(options))
		for _, option := range options {
			optionKey := normalizeSemanticAnswerText(option)
			if optionKey == "" {
				continue
			}
			if strings.Contains(joinedTipKey, optionKey) {
				containsMatches = append(containsMatches, option)
			}
		}
		if len(containsMatches) > 0 {
			matches = append(matches, containsMatches...)
		}
	}
	if len(matches) > 0 {
		return RemoveRepByLoop(matches)
	}

	candidates := buildTipCandidates(tips)
	limit := len(candidates)
	if limit < 1 {
		limit = 1
	}
	if limit > len(options) {
		limit = len(options)
	}
	return append([]string(nil), options[:limit]...)
}

func cleanSelectableAnswerText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, " ", " ")
	text = strings.TrimSpace(text)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^[A-Za-zＡ-Ｚａ-ｚ][\.．、:：]\s*`),
		regexp.MustCompile(`^\d+[\.．、:：]\s*`),
	}
	for _, pattern := range patterns {
		text = pattern.ReplaceAllString(text, "")
	}
	return strings.TrimSpace(text)
}

func normalizeSemanticAnswerText(text string) string {
	text = cleanSelectableAnswerText(text)
	replacer := strings.NewReplacer(
		" ", "",
		"\n", "",
		"\t", "",
		"\r", "",
		" ", "",
		"，", "",
		"。", "",
		"、", "",
		",", "",
		".", "",
		"：", "",
		":", "",
		"；", "",
		";", "",
		"（", "",
		"）", "",
		"(", "",
		")", "",
		"“", "",
		"”", "",
		"《", "",
		"》", "",
		"？", "",
		"?", "",
		"！", "",
		"!", "",
		"‘", "",
		"’", "",
		"【", "",
		"】", "",
	)
	return replacer.Replace(strings.TrimSpace(text))
}

func hasReverseSingleChoicePrompt(questionText string) bool {
	normalized := normalizeSemanticAnswerText(questionText)
	if normalized == "" {
		return false
	}
	patterns := []string{
		"错误的是",
		"不正确的是",
		"有误的是",
		"表述错误",
		"表述有误",
		"说法错误",
		"说法有误",
		"错误说法",
		"有误说法",
		"不属于",
		"不包括",
		"不符合",
		"不是",
		"不恰当",
		"不准确",
		"不可以",
		"不能",
		"不应",
		"不宜",
		"例外的是",
		"除外",
	}
	for _, pattern := range patterns {
		if strings.Contains(normalized, pattern) {
			return true
		}
	}
	return false
}

func containsSemanticNegation(text string) bool {
	normalized := normalizeSemanticAnswerText(text)
	if normalized == "" {
		return false
	}
	patterns := []string{
		"不构成",
		"不属于",
		"不包括",
		"不符合",
		"不正确",
		"不可以",
		"不能",
		"不可",
		"不应",
		"不宜",
		"不是",
		"不得",
		"不要",
		"无需",
		"无须",
		"无证",
		"错误",
		"有误",
		"未",
		"无",
		"非",
		"勿",
		"莫",
	}
	for _, pattern := range patterns {
		if strings.Contains(normalized, pattern) {
			return true
		}
	}
	return false
}

func longestCommonSubsequenceLength(left string, right string) int {
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	if len(leftRunes) == 0 || len(rightRunes) == 0 {
		return 0
	}
	dp := make([]int, len(rightRunes)+1)
	for i := 1; i <= len(leftRunes); i++ {
		prev := 0
		for j := 1; j <= len(rightRunes); j++ {
			current := dp[j]
			if leftRunes[i-1] == rightRunes[j-1] {
				dp[j] = prev + 1
			} else if dp[j-1] > dp[j] {
				dp[j] = dp[j-1]
			}
			prev = current
		}
	}
	return dp[len(rightRunes)]
}

func scoreSingleChoiceSimilarity(option string, candidate string, reverse bool) int {
	optionKey := normalizeSemanticAnswerText(option)
	candidateKey := normalizeSemanticAnswerText(candidate)
	if optionKey == "" || candidateKey == "" {
		return -1
	}
	score := longestCommonSubsequenceLength(optionKey, candidateKey) * 10
	if strings.Contains(optionKey, candidateKey) || strings.Contains(candidateKey, optionKey) {
		score += 30
	}
	diff := utf8.RuneCountInString(optionKey) - utf8.RuneCountInString(candidateKey)
	if diff < 0 {
		diff = -diff
	}
	score -= diff * 2
	optionNegation := containsSemanticNegation(optionKey)
	candidateNegation := containsSemanticNegation(candidateKey)
	if reverse {
		if optionNegation != candidateNegation {
			score += 40
		}
	} else if optionNegation != candidateNegation {
		score -= 20
	}
	return score
}

func pickMostSimilarSingleChoiceOption(options []string, tips []string, reverse bool) string {
	candidates := buildTipCandidates(tips)
	bestScore := -1
	bestOption := ""
	for _, option := range options {
		for _, candidate := range candidates {
			score := scoreSingleChoiceSimilarity(option, candidate, reverse)
			if score > bestScore {
				bestScore = score
				bestOption = option
			}
		}
	}
	if bestScore < 20 {
		return ""
	}
	return bestOption
}

func selectSingleChoiceAnswers(questionText string, options []string, tips []string) []string {
	reverse := hasReverseSingleChoicePrompt(questionText)
	if reverse {
		if answer := pickMostSimilarSingleChoiceOption(options, tips, true); answer != "" {
			return []string{answer}
		}
	}

	if exact := matchSelectableAnswers(options, tips); len(exact) > 0 {
		return []string{exact[0]}
	}

	if answer := pickMostSimilarSingleChoiceOption(options, tips, false); answer != "" {
		return []string{answer}
	}

	return nil
}

func splitAnswerToRunes(answer string) []string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return nil
	}
	result := make([]string, 0, utf8.RuneCountInString(answer))
	for _, r := range []rune(answer) {
		part := strings.TrimSpace(string(r))
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func canonicalRuneBag(text string) string {
	runes := []rune(normalizeSemanticAnswerText(text))
	if len(runes) == 0 {
		return ""
	}
	sort.Slice(runes, func(i, j int) bool {
		return runes[i] < runes[j]
	})
	return string(runes)
}

func hasMultiRuneClickBlankParts(parts []string) bool {
	for _, part := range parts {
		if utf8.RuneCountInString(strings.TrimSpace(part)) > 1 {
			return true
		}
	}
	return false
}

func truncateAnswerRunes(text string, limit int) string {
	if limit <= 0 {
		return text
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return ""
	}
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}

func detectBlankInputLimit(handle playwright.ElementHandle) int {
	if handle == nil {
		return 0
	}
	for _, attr := range []string{"maxlength", "maxLength", "size"} {
		value, err := handle.GetAttribute(attr)
		if err != nil {
			continue
		}
		limit, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil && limit > 0 {
			return limit
		}
	}
	return 0
}

func buildFallbackBlankAnswer(index int, limit int) string {
	templates := []string{
		"我不知道",
		"我真的不会回答这道题目简直太难了",
		"不知道",
		"我不会",
	}
	if limit > 0 {
		seed := templates[index%len(templates)] + templates[(index+1)%len(templates)] + templates[(index+2)%len(templates)]
		if truncated := truncateAnswerRunes(seed, limit); truncated != "" {
			return truncated
		}
		return "答"
	}
	return templates[index%len(templates)]
}

func expandFillBlankAnswers(answer []string, inputs []playwright.ElementHandle) []string {
	if len(inputs) == 0 {
		return nil
	}

	expanded := append([]string(nil), answer...)
	if len(expanded) == 1 && len(inputs) > 1 {
		combined := strings.TrimSpace(expanded[0])
		for _, sep := range []string{"，", "、", ",", " ", "|", "/"} {
			if strings.Contains(combined, sep) {
				parts := make([]string, 0, len(inputs))
				for _, part := range strings.Split(combined, sep) {
					part = strings.TrimSpace(part)
					if part != "" {
						parts = append(parts, part)
					}
				}
				if len(parts) > 1 {
					expanded = parts
					log.Infoln("[填空題] 按分隔符分割后: ", expanded)
					break
				}
			}
		}
		if len(expanded) == 1 {
			runes := splitAnswerToRunes(combined)
			if len(runes) == len(inputs) {
				expanded = runes
				log.Infoln("[填空題] 按字元拆分后: ", expanded)
			}
		}
	}

	if len(expanded) > len(inputs) && len(inputs) == 1 {
		expanded = []string{strings.Join(expanded, "")}
		log.Infoln("[填空題] 合并答案: ", expanded[0])
	}

	result := make([]string, 0, len(inputs))
	for i := 0; i < len(inputs); i++ {
		value := ""
		if len(expanded) > i {
			value = strings.TrimSpace(expanded[i])
		}
		if value == "" {
			value = buildFallbackBlankAnswer(i, detectBlankInputLimit(inputs[i]))
		}
		result = append(result, value)
	}
	return result
}

func currentAnswerQuestionText(page playwright.Page) string {
	selectors := []string{
		`#app .detail-body .question .q-body > div`,
		`.detail-body .question .q-body > div`,
		`.question .q-body > div`,
		`.question .q-body`,
	}
	for _, selector := range selectors {
		handle, err := page.QuerySelector(selector)
		if err != nil || handle == nil {
			continue
		}
		text, err := handle.TextContent()
		if err != nil {
			continue
		}
		text = strings.TrimSpace(text)
		if text != "" {
			return text
		}
	}
	return ""
}

func normalizeAnswerQuestionKey(text string) string {
	key := normalizeSemanticAnswerText(text)
	if key != "" {
		return key
	}
	return normalizeAnswerButtonText(text)
}

func logAnswerStateSnapshot(page playwright.Page, prefix string) {
	result, err := page.Evaluate(`() => {
		const normalize = (value) => String(value || "").replace(/\s+/g, " ").trim();
		const isVisible = (el) => {
			if (!el) return false;
			const rect = el.getBoundingClientRect();
			const style = window.getComputedStyle(el);
			return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
		};
		const bodyText = normalize(document.body ? document.body.innerText || "" : "");
		const selectors = ["button", "[role='button']", ".ant-btn", "a", "span", "div"];
		const seen = new Set();
		const buttons = [];
		for (const selector of selectors) {
			for (const el of Array.from(document.querySelectorAll(selector)).slice(0, 300)) {
				if (!isVisible(el)) continue;
				const text = normalize(el.innerText || el.textContent || "");
				if (!text || text.length > 30 || seen.has(text)) continue;
				seen.add(text);
				buttons.push(text);
				if (buttons.length >= 25) break;
			}
			if (buttons.length >= 25) break;
		}
		return {
			url: window.location.href,
			bodyText: bodyText.slice(0, 800),
			buttons,
		};
	}`)
	if err != nil {
		log.Warningln(prefix, "抓取頁面快照失敗：", err.Error())
		return
	}
	payload, ok := result.(map[string]interface{})
	if !ok {
		log.Warningln(prefix, "頁面快照格式異常")
		return
	}
	url, _ := payload["url"].(string)
	bodyText, _ := payload["bodyText"].(string)
	buttons := []string{}
	if rawButtons, ok := payload["buttons"].([]interface{}); ok {
		for _, item := range rawButtons {
			text, ok := item.(string)
			if ok && strings.TrimSpace(text) != "" {
				buttons = append(buttons, text)
			}
		}
	}
	log.Warningln(prefix, "URL:", url)
	log.Warningln(prefix, "頁面文字:", bodyText)
	log.Warningln(prefix, "可見按鈕:", buttons)
}

func uniqueSelectableAnswerTexts(options []string) []string {
	seen := make(map[string]struct{}, len(options))
	result := make([]string, 0, len(options))
	for _, option := range options {
		cleaned := cleanSelectableAnswerText(option)
		key := normalizeAnswerButtonText(cleaned)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, cleaned)
	}
	sort.Slice(result, func(i, j int) bool {
		left := utf8.RuneCountInString(result[i])
		right := utf8.RuneCountInString(result[j])
		if left != right {
			return left > right
		}
		return result[i] < result[j]
	})
	return result
}

func segmentAnswerByOptions(answer string, options []string, blankCount int) []string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return nil
	}
	prepared := uniqueSelectableAnswerTexts(options)
	if len(prepared) == 0 {
		return nil
	}

	bestScore := -1 << 30
	var best []string
	var dfs func(rem string, path []string)
	dfs = func(rem string, path []string) {
		if rem == "" {
			score := 0
			if blankCount > 0 {
				diff := len(path) - blankCount
				if diff < 0 {
					diff = -diff
				}
				if diff == 0 {
					score += 1000
				} else {
					score -= diff * 100
				}
			}
			for _, item := range path {
				score += utf8.RuneCountInString(item) * 10
			}
			score -= len(path)
			if score > bestScore {
				bestScore = score
				best = append([]string(nil), path...)
			}
			return
		}

		for _, option := range prepared {
			if !strings.HasPrefix(rem, option) {
				continue
			}
			dfs(rem[len(option):], append(path, option))
		}
	}
	dfs(answer, nil)
	return best
}

func buildClickBlankAnswers(tips []string, options []string, blankCount int) []string {
	candidates := buildTipCandidates(tips)
	for _, candidate := range candidates {
		if segmented := segmentAnswerByOptions(candidate, options, blankCount); len(segmented) > 0 {
			if hasMultiRuneClickBlankParts(segmented) || len(segmented) < blankCount {
				return segmented
			}
		}
	}
	if blankCount > 1 {
		for _, candidate := range candidates {
			candidateBag := canonicalRuneBag(candidate)
			if candidateBag == "" {
				continue
			}
			for _, option := range uniqueSelectableAnswerTexts(options) {
				if utf8.RuneCountInString(option) <= 1 {
					continue
				}
				if canonicalRuneBag(option) == candidateBag {
					return []string{option}
				}
			}
		}
	}
	for _, candidate := range candidates {
		if segmented := segmentAnswerByOptions(candidate, options, blankCount); len(segmented) > 0 {
			return segmented
		}
	}

	if blankCount > 0 {
		for _, candidate := range candidates {
			runes := splitAnswerToRunes(candidate)
			if len(runes) == blankCount {
				return runes
			}
		}
		if len(candidates) == blankCount {
			return append([]string(nil), candidates...)
		}
	}

	if len(options) > 0 {
		matches := pickSelectableAnswers(options, tips)
		if len(matches) > 0 {
			result := make([]string, 0, len(matches))
			for _, match := range matches {
				cleaned := cleanSelectableAnswerText(match)
				if cleaned != "" {
					result = append(result, cleaned)
				}
			}
			if len(result) > 0 {
				return result
			}
		}
	}

	return nil
}

func decodeClickBlankState(result interface{}) (int, []string) {
	payload, ok := result.(map[string]interface{})
	if !ok {
		return 0, nil
	}
	blankCount := intFromEvalValue(payload["blankCount"])
	rawOptions, ok := payload["options"].([]interface{})
	if !ok {
		return blankCount, nil
	}
	options := make([]string, 0, len(rawOptions))
	for _, item := range rawOptions {
		text, ok := item.(string)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		options = append(options, text)
	}
	return blankCount, options
}

func getClickBlankState(page playwright.Page) (int, []string, error) {
	result, err := page.Evaluate(`() => {
		const questionRoot = document.querySelector("#app .detail-body .question") || document.querySelector(".question") || document.body;
		const clickBoxes = Array.from(questionRoot.querySelectorAll(".q-body .click-box"));
		const stopTexts = new Set(["提示", "查看提示", "下一题", "确定", "提交", "完成", "确认", "关闭", "上一题", "再来一组"]);
		const isVisible = (el) => {
			if (!el) return false;
			const rect = el.getBoundingClientRect();
			const style = window.getComputedStyle(el);
			return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
		};
		const normalize = (value) => String(value || "").replace(/\s+/g, "").trim();
		const selectors = [
			".q-answer.choosable",
			".q-answer",
			"[class*='answer']",
			"[class*='option']",
			"[class*='word']",
			"[class*='choice']",
			"[class*='blank']",
			"button",
			"[role='button']",
			"span",
			"div"
		];
		const seen = new Set();
		const options = [];
		const isBlankBox = (el) => clickBoxes.some((box) => box === el || box.contains(el));
		for (const selector of selectors) {
			for (const el of Array.from(questionRoot.querySelectorAll(selector)).slice(0, 200)) {
				if (!isVisible(el)) continue;
				if (isBlankBox(el) || el.closest(".q-header") || el.closest(".action-row") || el.closest(".answer-tip") || el.closest("[class*='tips']") || el.closest(".q-footer")) {
					continue;
				}
				const text = normalize(el.innerText || el.textContent || "");
				if (!text || stopTexts.has(text)) continue;
				const textLen = Array.from(text).length;
				if (textLen < 1 || textLen > 8) continue;
				if (seen.has(text)) continue;
				seen.add(text);
				options.push(text);
			}
		}
		return { blankCount: clickBoxes.length, options };
	}`)
	if err != nil {
		return 0, nil, err
	}
	blankCount, options := decodeClickBlankState(result)
	return blankCount, options, nil
}

func fillClickBlank(page playwright.Page, questionText string, answer []string) error {
	cleaned := make([]string, 0, len(answer))
	for _, item := range answer {
		value := cleanSelectableAnswerText(item)
		if value == "" {
			continue
		}
		cleaned = append(cleaned, value)
	}
	if len(cleaned) == 0 {
		return errors.New("未生成点选填空答案")
	}

	result, err := page.Evaluate(`(targets) => {
		const answers = Array.isArray(targets) ? targets.map((item) => String(item || "").replace(/\s+/g, "").trim()).filter(Boolean) : [];
		const questionRoot = document.querySelector("#app .detail-body .question") || document.querySelector(".question") || document.body;
		const clickBoxes = Array.from(questionRoot.querySelectorAll(".q-body .click-box"));
		const stopTexts = new Set(["提示", "查看提示", "下一题", "确定", "提交", "完成", "确认", "关闭", "上一题", "再来一组"]);
		const isVisible = (el) => {
			if (!el) return false;
			const rect = el.getBoundingClientRect();
			const style = window.getComputedStyle(el);
			return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
		};
		const normalize = (value) => String(value || "").replace(/\s+/g, "").trim();
		const tryClick = (el) => {
			if (!el) return false;
			try {
				const _r = el.getBoundingClientRect(); el.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, view: window, clientX: _r.left + _r.width * (0.3 + Math.random() * 0.4), clientY: _r.top + _r.height * (0.3 + Math.random() * 0.4) }));
			} catch (e) {}
			try {
				if (typeof el.click === "function") {
					el.click();
					return true;
				}
			} catch (e) {}
			return false;
		};
		const collectOptions = () => {
			const selectors = [
				".q-answer.choosable",
				".q-answer",
				"[class*='answer']",
				"[class*='option']",
				"[class*='word']",
				"[class*='choice']",
				"[class*='blank']",
				"button",
				"[role='button']",
				"span",
				"div"
			];
			const seen = new Set();
			const options = [];
			const isBlankBox = (el) => clickBoxes.some((box) => box === el || box.contains(el));
			for (const selector of selectors) {
				for (const el of Array.from(questionRoot.querySelectorAll(selector)).slice(0, 200)) {
					if (!isVisible(el)) continue;
					if (isBlankBox(el) || el.closest(".q-header") || el.closest(".action-row") || el.closest(".answer-tip") || el.closest("[class*='tips']") || el.closest(".q-footer")) {
						continue;
					}
					const text = normalize(el.innerText || el.textContent || "");
					if (!text || stopTexts.has(text)) continue;
					const textLen = Array.from(text).length;
					if (textLen < 1 || textLen > 8) continue;
					if (seen.has(text)) continue;
					seen.add(text);
					options.push({ text, el });
				}
			}
			return options;
		};
		const used = [];
		for (let i = 0; i < answers.length; i += 1) {
			const box = clickBoxes[i] || clickBoxes[0];
			if (box) {
				tryClick(box);
			}
			const target = answers[i];
			const options = collectOptions();
			const match = options.find((item) => {
				const optionText = normalize(item.text);
				return optionText === target || optionText.includes(target) || target.includes(optionText);
			});
			if (!match) {
				return {
					ok: false,
					reason: "未找到候选词:" + target,
					options: options.map((item) => item.text),
					blankCount: clickBoxes.length,
					used,
				};
			}
			tryClick(match.el);
			used.push(match.text);
		}
		return { ok: true, blankCount: clickBoxes.length, used };
	}`, cleaned)
	if err != nil {
		return err
	}
	payload, ok := result.(map[string]interface{})
	if !ok {
		return errors.New("点选填空执行结果格式异常")
	}
	if success, _ := payload["ok"].(bool); success {
		log.Infoln("[點選填空] 已點選答案：", cleaned)
		_, _ = page.Evaluate(`() => {
			const target = document.querySelector("#app .detail-body .question") || document.querySelector(".question") || document.body;
			if (target && typeof target.click === "function") {
				target.click();
			}
			return true;
		}`)
		humanPause(1800, 3200)
		if isAnswerRoundComplete(page) {
			log.Infoln("[答題] 點選填空後直接進入結果頁，本輪答題結束")
			return ErrAnswerComplete
		}
		return checkNextBotton(page, questionText)
	}
	reason, _ := payload["reason"].(string)
	options := []string{}
	if rawOptions, ok := payload["options"].([]interface{}); ok {
		for _, item := range rawOptions {
			text, ok := item.(string)
			if ok && strings.TrimSpace(text) != "" {
				options = append(options, text)
			}
		}
	}
	log.Warningln("[點選填空] 可見候選詞：", options)
	if reason == "" {
		reason = "点选填空执行失败"
	}
	return errors.New(reason)
}

func truncateAnswerDebug(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit < 1 || len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

// 填空题
// 简化版：有提示用提示，没提示搜索题库，都没有就填"不知道"
func FillBlank(page playwright.Page, questionText string, tips []string) error {
	log.Infoln("[填空題] 開始處理填空題")

	// 尝试多种选择器获取输入框
	inputSelectors := []string{
		`div.q-body > div > input`,
		`input[type="text"]`,
		`textarea`,
		`.q-body input`,
		`.q-body textarea`,
		`div.q-body input`,
		`div.q-body textarea`,
		`input[placeholder]`,
		`input.blank-input`,
		`.q-body [contenteditable="true"]`,
		`.q-body [contenteditable="plaintext-only"]`,
		`.q-body [contenteditable]`,
		`.q-body [class*="blank"] input`,
		`.q-body [class*="blank"] textarea`,
		`.q-body [class*="blank"] [contenteditable]`,
	}

	var inouts []playwright.ElementHandle
	var err error

	for _, selector := range inputSelectors {
		inouts, err = page.QuerySelectorAll(selector)
		inouts = filterVisibleAnswerHandles(inouts)
		if err == nil && len(inouts) > 0 {
			log.Infoln("[填空題] 使用選擇器 '", selector, "' 找到 ", len(inouts), " 個填空框")
			break
		}
	}

	if len(inouts) == 0 {
		blankCount, clickOptions, clickErr := getClickBlankState(page)
		if clickErr == nil && blankCount > 0 {
			log.Infoln("[填空題] 檢測到點選填空，空格數：", blankCount, " 候選詞：", clickOptions)
			answer := buildClickBlankAnswers(tips, clickOptions, blankCount)
			if len(answer) > 0 {
				log.Infoln("[填空題] 点选填空分段结果：", answer)
				return fillClickBlank(page, questionText, answer)
			}
			log.Warningln("[填空題] 未能根據提示生成點選填空答案")
		}
		options, optErr := getOptions(page)
		if optErr == nil && len(options) > 0 {
			log.Infoln("[填空題] 未找到輸入框，檢測到可點選答案，改用點選模式")
			log.Infoln("[填空題] 获取到点选答案：", options)
			answer := pickSelectableAnswers(options, tips)
			log.Infoln("[填空題] 根据提示分别选择了", answer)
			return radioCheck(page, questionText, answer)
		}
		if html, htmlErr := page.InnerHTML(`.q-body`); htmlErr == nil {
			log.Warningln("[填空題] q-body HTML片段: ", truncateAnswerDebug(html, 1800))
		}
		if snapshot, snapErr := page.Evaluate(`() => {
			const selectors = [
				".q-answer",
				".q-answer.choosable",
				".q-body [class*='blank']",
				".q-body [class*='word']",
				".q-body [class*='answer']",
				".q-body button",
				".q-body [role='button']",
				".q-body span",
				".q-body div"
			];
			const result = [];
			const isVisible = (el) => {
				if (!el) return false;
				const rect = el.getBoundingClientRect();
				const style = window.getComputedStyle(el);
				return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
			};
			for (const selector of selectors) {
				for (const el of Array.from(document.querySelectorAll(selector)).slice(0, 40)) {
					if (!isVisible(el)) continue;
					const text = (el.textContent || "").trim();
					if (!text) continue;
					result.push({
						selector,
						tag: el.tagName,
						className: String(el.className || ""),
						text: text.slice(0, 60)
					});
					if (result.length >= 25) {
						return result;
					}
				}
			}
			return result;
		}`); snapErr == nil {
			log.Warningf("[填空題] 點選候選元素: %v", snapshot)
		}
		log.Warningln("[填空題] 未找到任何填空框，頁面結構可能已變")
		return errors.New("未找到可编辑的填空框")
	}

	log.Infoln("[填空題] 获取到", len(inouts), "个填空框")

	var answer []string

	// 优先使用提示
	if len(tips) > 0 {
		answer = tips
		log.Infoln("[填空題] 使用提示信息: ", tips)
	} else {
		// 没有提示，尝试题库搜索
		log.Infoln("[填空題] 无提示，尝试题库搜索")

		// 获取题目内容
		if questionText != "" {
			searchAnswer := model.SearchAnswer(questionText)
			if searchAnswer != "" {
				log.Infoln("[填空題] 题库找到: ", searchAnswer)
				answer = append(answer, searchAnswer)
			}
		}

		// 题库也没找到，用默认答案
		if len(answer) == 0 {
			for i := 0; i < len(inouts); i++ {
				answer = append(answer, "不知道")
			}
			log.Infoln("[填空題] 无答案，使用默认值填充")
		}
	}

	answer = expandFillBlankAnswers(answer, inouts)

	// 填充答案
	for i := 0; i < len(inouts); i++ {
		ans := buildFallbackBlankAnswer(i, detectBlankInputLimit(inouts[i]))
		if len(answer) > i && strings.TrimSpace(answer[i]) != "" {
			ans = strings.TrimSpace(answer[i])
		}

		log.Infoln("[填空題] 填入: ", ans)
		if err := inouts[i].Fill(ans); err != nil {
			log.Errorln("[填空題] 填充失败: ", err.Error())
			return fmt.Errorf("填空框第%d项填充失败: %w", i+1, err)
		}

		humanPause(300, 600)
	}

	humanPause(400, 800)
	return checkNextBotton(page, questionText)
}

// 检查下一题按钮
// 返回值：nil 继续，ErrAnswerComplete 答题结束
var ErrAnswerComplete = errors.New("答题已完成")
var ErrAnswerSliderChallenge = errors.New("答题遇到滑块验证")

func clickAnswerContinueButton(page playwright.Page, buttonSelectors []string) error {
	continueKeywords := []string{"继续答题", "继续", "下一题", "关闭", "完成", "确认", "知道了"}
	waitForVisibleSelector(page, buttonSelectors, 4, 300, 700)
	for _, selector := range buttonSelectors {
		btns, err := page.QuerySelectorAll(selector)
		if err != nil || len(btns) == 0 {
			continue
		}
		btn := pickAnswerActionButton(btns, continueKeywords)
		if btn == nil {
			continue
		}
		text, _ := btn.TextContent()
		text = strings.TrimSpace(text)
		if clickErr := clickAnswerActionHandle(btn); clickErr == nil {
			log.Infoln("[下一題] 已點擊繼續按鈕：", text)

			// 如果點擊的是「完成」按鈕，檢測是否有滑塊驗證
			if strings.Contains(text, "完成") {
				humanPause(800, 1500)
				// 調試：打印頁面上所有可能的滑塊元素
				detectAndLogSliderElements(page)

				if hasAnswerSliderPrompt(page) {
					log.Infoln("[答題] 點擊「完成」後檢測到滑塊驗證")
					return ErrAnswerSliderChallenge
				}
			}

			humanPause(500, 1000)
			if isAnswerRoundComplete(page) {
				log.Infoln("[答題] 檢測到結果頁，本輪答題結束")
				return ErrAnswerComplete
			}
			return nil
		}
	}
	result, evalErr := page.Evaluate(`(keywords) => {
		const normalizedKeywords = Array.isArray(keywords) ? keywords.map((item) => String(item || "").replace(/\s+/g, "").trim()).filter(Boolean) : [];
		const isVisible = (el) => {
			if (!el) return false;
			const rect = el.getBoundingClientRect();
			const style = window.getComputedStyle(el);
			return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
		};
		const normalize = (value) => String(value || "").replace(/\s+/g, "").trim();
		const tryClick = (el) => {
			if (!el) return false;
			try {
				const _r = el.getBoundingClientRect(); el.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, view: window, clientX: _r.left + _r.width * (0.3 + Math.random() * 0.4), clientY: _r.top + _r.height * (0.3 + Math.random() * 0.4) }));
			} catch (e) {}
			try {
				if (typeof el.click === "function") {
					el.click();
					return true;
				}
			} catch (e) {}
			return false;
		};
		const selectors = ["button", "[role='button']", ".ant-btn", "div", "span", "a"];
		const seen = new Set();
		for (const selector of selectors) {
			for (const el of Array.from(document.querySelectorAll(selector)).slice(0, 300)) {
				if (!isVisible(el)) continue;
				if (el.closest(".q-body") || el.closest(".q-header")) continue;
				const text = normalize(el.innerText || el.textContent || "");
				if (!text || seen.has(text)) continue;
				seen.add(text);
				if (!normalizedKeywords.some((keyword) => text.includes(keyword))) {
					continue;
				}
				if (tryClick(el)) {
					return { ok: true, text };
				}
			}
		}
		return { ok: false };
	}`, continueKeywords)
	if evalErr == nil {
		if payload, ok := result.(map[string]interface{}); ok {
			if success, _ := payload["ok"].(bool); success {
				text, _ := payload["text"].(string)
				log.Infoln("[下一題] 已透過全局搜尋點擊繼續按鈕：", text)

				// 如果點擊的是「完成」按鈕，檢測滑塊
				if strings.Contains(text, "完成") {
					humanPause(800, 1500)
					detectAndLogSliderElements(page)
					if hasAnswerSliderPrompt(page) {
						log.Infoln("[答題] 全局搜索點擊「完成」後檢測到滑塊驗證")
						return ErrAnswerSliderChallenge
					}
				}

				humanPause(500, 1000)
				if isAnswerRoundComplete(page) {
					log.Infoln("[答題] 檢測到結果頁，本輪答題結束")
					return ErrAnswerComplete
				}
				return nil
			}
		}
	}
	if isAnswerRoundComplete(page) {
		log.Infoln("[答題] 檢測到結果頁，本輪答題結束")
		return ErrAnswerComplete
	}
	return nil
}

func waitForAnswerAdvance(page playwright.Page, previousQuestionText string, buttonSelectors []string) error {
	previousKey := normalizeAnswerQuestionKey(previousQuestionText)
	for attempt := 0; attempt < 8; attempt++ {
		if isAnswerRoundComplete(page) {
			log.Infoln("[答題] 檢測到結果頁，本輪答題結束")
			return ErrAnswerComplete
		}
		if previousKey != "" {
			currentText := currentAnswerQuestionText(page)
			currentKey := normalizeAnswerQuestionKey(currentText)
			if currentKey != "" && currentKey != previousKey {
				log.Infoln("[答題] 已前進到下一題：", truncateAnswerDebug(currentText, 120))
				return nil
			}
		}
		if hasAnswerSliderPrompt(page) {
			log.Warningln("[答題] 提交後出現滑塊驗證，等待上層處理")
			return ErrAnswerSliderChallenge
		}
		if err := clickAnswerContinueButton(page, buttonSelectors); err != nil {
			return err
		}
		if attempt < 7 {
			humanPause(400, 800)
		}
	}
	if previousKey != "" {
		log.Warningln("[答題] 提交後仍停留在原題，將交由上層重試")
	}
	logAnswerStateSnapshot(page, "[答題] 提交後頁面快照")
	return nil
}

// waitForSystemJudgment 等待系統判斷答案完成
// 點擊「確定」後，系統需要時間判斷答案是否正確，此時「下一題」按鈕是灰色的
// 需要等待「下一題」按鈕變為可點擊狀態
func waitForSystemJudgment(page playwright.Page, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	nextButtonSelectors := []string{
		`#app .action-row > button`,
		`#app .action-row [role="button"]`,
		`#app .action-row > div`,
		`.action-row button`,
		`.action-row [role="button"]`,
	}
	nextKeywords := []string{"下一题", "继续答题", "完成"}
	checkCount := 0
	loggedButtons := false
	lastConfirmDisabled := true

	for time.Now().Before(deadline) {
		checkCount++
		shouldDeepScan := checkCount == 1 || checkCount%3 == 0

		// 優先做低成本 selector 檢查，避免在緊密輪詢中反覆抽取全文文本
		if hasAnswerSliderExactPrompt(page) {
			log.Infoln("[答題] 系統判斷期間出現滑塊驗證")
			return false
		}

		if shouldDeepScan {
			pageText := getAnswerPageText(page)
			if isAnswerCompletionText(pageText) {
				log.Infoln("[答題] 系統判斷完成，已到達結果頁")
				return true
			}
			if containsAnswerFlowBlockedText(pageText) {
				log.Warningln("[答題] 檢測到答題流程佔用提示")
				return false
			}
			if containsAnswerSliderSpecificText(pageText) ||
				((containsAnswerSliderContextText(pageText) || containsAnswerSliderLooseText(pageText)) && hasPotentialAnswerSliderElement(page)) {
				log.Infoln("[答題] 系統判斷期間出現滑塊驗證")
				return false
			}
		}

		// 第一次檢測時，打印頁面狀態（用於調試）
		if checkCount == 1 {
			errorText, _ := page.Evaluate(`() => {
				const errorEl = document.querySelector('.error, .ant-message, .ant-alert, [class*="error"], [class*="loading"]');
				return errorEl ? errorEl.textContent : '';
			}`)
			if errText, ok := errorText.(string); ok && errText != "" {
				log.Infoln("[答題] 調試：檢測到提示信息: ", errText)
			}
		}

		// 檢測「下一題」按鈕是否可點擊
		for selectorIdx, selector := range nextButtonSelectors {
			btns, err := page.QuerySelectorAll(selector)
			if err != nil || len(btns) == 0 {
				continue
			}

			// 前幾次檢測時，打印所有選擇器的按鈕信息（用於調試）
			if !loggedButtons && checkCount <= 3 && selectorIdx < 3 {
				log.Debugln("[答題] 調試：選擇器[", selectorIdx, "] ", selector, " 找到 ", len(btns), " 個按鈕")
				for i, btn := range btns {
					text, _ := btn.TextContent()
					text = strings.TrimSpace(text)
					isDisabled, _ := btn.Evaluate(`el => el.disabled || el.classList.contains('disabled') || el.classList.contains('ant-btn-disabled')`)
					disabled, _ := isDisabled.(bool)
					log.Debugln("[答題] 調試：按鈕[", i, "] 文本='", text, "' 禁用=", disabled)
				}
				if selectorIdx == 2 {
					loggedButtons = true
				}
			}

			for _, btn := range btns {
				text, _ := btn.TextContent()
				text = strings.TrimSpace(strings.ReplaceAll(text, " ", ""))
				// 檢查是否是「下一題」按鈕
				isNextButton := false
				for _, keyword := range nextKeywords {
					if strings.Contains(text, keyword) {
						isNextButton = true
						break
					}
				}
				if !isNextButton {
					continue
				}
				// 檢查按鈕是否可點擊（不是灰色/禁用狀態）
				isDisabled, _ := btn.Evaluate(`el => el.disabled || el.classList.contains('disabled') || el.classList.contains('ant-btn-disabled')`)
				disabled, _ := isDisabled.(bool)
				if !disabled {
					// 額外檢查：按鈕是否可見且可交互
					isVisible, _ := btn.Evaluate(`el => {
						const rect = el.getBoundingClientRect();
						const style = window.getComputedStyle(el);
						return style.display !== 'none' &&
							style.visibility !== 'hidden' &&
							rect.width > 0 &&
							rect.height > 0 &&
							!el.hasAttribute('disabled');
					}`)
					visible, _ := isVisible.(bool)
					if visible {
						// 「完成」按鈕就緒時，CAPTCHA JS 可能尚未加載完成
						// 額外等待並再次檢查滑塊驗證，避免競態條件
						if strings.Contains(text, "完成") {
							log.Infoln("[答題] 「完成」按鈕已可點擊，等待確認是否有滑塊驗證...")
							humanPause(600, 1000)
							if hasAnswerSliderPrompt(page) {
								log.Infoln("[答題] 等待後檢測到滑塊驗證")
								return false
							}
						}
						log.Infoln("[答題] 系統判斷完成，「", text, "」按鈕已可點擊 (檢測次數:", checkCount, ")")
						return true
					} else {
						log.Debugln("[答題] 按鈕「", text, "」未通過可見性檢查")
					}
				} else {
					log.Debugln("[答題] 按鈕「", text, "」被禁用")
				}
			}
		}

		// 每3次檢測打印一次狀態，並檢查「確定」按鈕狀態
		if checkCount%3 == 0 {
			log.Debugln("[答題] 等待系統判斷中... (檢測次數:", checkCount, ")")
			// 檢查「確定」按鈕是否變為非禁用狀態（表示判斷完成但按鈕文本沒變）
			confirmBtns, _ := page.QuerySelectorAll(`#app .action-row > button`)
			for _, btn := range confirmBtns {
				text, _ := btn.TextContent()
				text = strings.TrimSpace(text)
				if text == "确定" || text == "確定" {
					isDisabled, _ := btn.Evaluate(`el => el.disabled || el.classList.contains('disabled') || el.classList.contains('ant-btn-disabled')`)
					disabled, _ := isDisabled.(bool)

					// 如果按鈕從禁用變為可用，說明判斷完成
					if lastConfirmDisabled && !disabled {
						log.Infoln("[答題] 「確定」按鈕已從禁用變為可用，判斷完成 (檢測次數:", checkCount, ")")
						return true
					}

					// 如果按鈕變為可用，嘗試重新點擊
					if !disabled {
						log.Warningln("[答題] 「確定」按鈕已恢復可點擊狀態，嘗試重新點擊 (檢測次數:", checkCount, ")")
						humanClick(btn)
						humanPause(500, 1000)
					}

					lastConfirmDisabled = disabled
					break
				}
			}
		}

		humanPause(650, 1050)
	}

	log.Warningln("[答題] 等待系統判斷超時 (檢測次數:", checkCount, ")")

	// 超時時捕獲頁面狀態用於調試
	pageContent, _ := page.Evaluate(`() => {
		const buttons = Array.from(document.querySelectorAll('button')).map(b => ({
			text: b.textContent.trim(),
			disabled: b.disabled
		}));
		const errorEl = document.querySelector('.error, .ant-message, .ant-alert, [class*="error"]');
		return {
			url: window.location.href,
			buttons: buttons,
			errorText: errorEl ? errorEl.textContent : null
		};
	}`)
	if content, ok := pageContent.(map[string]interface{}); ok {
		log.Warningln("[答題] 超時時頁面狀態: URL=", content["url"])
		if btns, ok := content["buttons"].([]interface{}); ok {
			for i, b := range btns {
				if btn, ok := b.(map[string]interface{}); ok {
					log.Warningln("[答題] 超時時按鈕[", i, "] 文本='", btn["text"], "' 禁用=", btn["disabled"])
				}
			}
		}
		if content["errorText"] != nil {
			log.Warningln("[答題] 超時時錯誤信息: ", content["errorText"])
		}
	}

	return false
}

func checkNextBotton(page playwright.Page, previousQuestionText string) error {
	keywords := []string{"下一题", "确定", "提交", "完成", "确认"}
	buttonSelectors := []string{
		`#app .action-row > button`,
		`#app .action-row [role="button"]`,
		`#app .action-row > div`,
		`.action-row button`,
		`.action-row [role="button"]`,
		`button.ant-btn`,
		`.ant-btn`,
		`button[class*="submit"]`,
		`button[class*="next"]`,
		`button`,
	}
	if isAnswerRoundComplete(page) {
		log.Infoln("[答題] 檢測到結果頁，本輪答題結束")
		return ErrAnswerComplete
	}
	waitForVisibleSelector(page, buttonSelectors, 4, 200, 400)

	var lastErr error

	// 第一阶段：点击提交/确定按钮
	for _, selector := range buttonSelectors {
		btns, err := page.QuerySelectorAll(selector)
		if err != nil || len(btns) == 0 {
			continue
		}
		btn := pickAnswerActionButton(btns, keywords)
		if btn == nil {
			continue
		}
		btnText, _ := btn.TextContent()
		btnText = strings.TrimSpace(btnText)

		// 快速確認
		humanPause(200, 500)

		if err := humanClick(btn); err != nil {
			lastErr = err
			continue
		}
		log.Infoln("[下一題] 已點擊按鈕：", btnText)

		// 點擊「確定」後，等待系統判斷完成
		log.Infoln("[答題] 等待系統判斷答案...")

		// 等待頁面穩定，預留足夠時間讓 CAPTCHA JS 加載
		_ = page.WaitForLoadState()
		humanPause(1200, 2000)

		// 檢測是否有滑塊
		if hasAnswerSliderPrompt(page) {
			log.Warningln("[答題] 提交後出現滑塊驗證，返回上層處理")
			return ErrAnswerSliderChallenge
		}

		// 等待系統判斷完成（最多等待15秒，加快速度）
		if !waitForSystemJudgment(page, 15*time.Second) {
			// 可能是滑塊或其他問題
			if hasAnswerSliderDeepPrompt(page) {
				log.Warningln("[答題] 等待判斷期間檢測到滑塊驗證")
				return ErrAnswerSliderChallenge
			}
			pageText := getAnswerPageText(page)
			if containsAnswerFlowBlockedText(pageText) {
				log.Warningln("[答題] 檢測到「多端同時作答」限制提示，等待後重試")
				humanPause(20000, 30000)
			}
			log.Warningln("[答題] 系統判斷超時，嘗試刷新頁面")
			// 刷新頁面重試
			page.Reload(playwright.PageReloadOptions{
				Timeout:   playwright.Float(15000),
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			humanPause(1000, 2000)

			// 刷新後再次檢測
			if isAnswerRoundComplete(page) {
				log.Infoln("[答題] 刷新後檢測到結果頁，本輪答題結束")
				return ErrAnswerComplete
			}
			if hasAnswerSliderDeepPrompt(page) {
				log.Warningln("[答題] 刷新後檢測到滑塊驗證")
				return ErrAnswerSliderChallenge
			}
			if containsAnswerFlowBlockedText(getAnswerPageText(page)) {
				return fmt.Errorf("檢測到答題流程佔用，刷新後仍未恢復")
			}
			// 返回錯誤，讓上層處理
			return fmt.Errorf("系統判斷超時，刷新後仍未恢復")
		}

		if isAnswerRoundComplete(page) {
			log.Infoln("[答題] 檢測到結果頁，本輪答題結束")
			return ErrAnswerComplete
		}

		if err := waitForAnswerAdvance(page, previousQuestionText, buttonSelectors); err != nil {
			return err
		}
		if isAnswerRoundComplete(page) {
			log.Infoln("[答題] 檢測到結果頁，本輪答題結束")
			return ErrAnswerComplete
		}
		return nil
	}

	result, evalErr := page.Evaluate(`(keywords) => {
		const normalizedKeywords = Array.isArray(keywords) ? keywords.map((item) => String(item || "").replace(/\s+/g, "").trim()).filter(Boolean) : [];
		const isVisible = (el) => {
			if (!el) return false;
			const rect = el.getBoundingClientRect();
			const style = window.getComputedStyle(el);
			return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
		};
		const normalize = (value) => String(value || "").replace(/\s+/g, "").trim();
		const tryClick = (el) => {
			if (!el) return false;
			try {
				const _r = el.getBoundingClientRect(); el.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, view: window, clientX: _r.left + _r.width * (0.3 + Math.random() * 0.4), clientY: _r.top + _r.height * (0.3 + Math.random() * 0.4) }));
			} catch (e) {}
			try {
				if (typeof el.click === "function") {
					el.click();
					return true;
				}
			} catch (e) {}
			return false;
		};
		const selectors = ["button", "[role='button']", ".ant-btn", "div", "span", "a"];
		const seen = new Set();
		for (const selector of selectors) {
			for (const el of Array.from(document.querySelectorAll(selector)).slice(0, 300)) {
				if (!isVisible(el)) continue;
				if (el.closest(".q-body") || el.closest(".q-header")) continue;
				const text = normalize(el.innerText || el.textContent || "");
				if (!text || seen.has(text)) continue;
				seen.add(text);
				if (!normalizedKeywords.some((keyword) => text.includes(keyword))) {
					continue;
				}
				if (tryClick(el)) {
					return { ok: true, text };
				}
			}
		}
		return { ok: false };
	}`, keywords)
	if evalErr == nil {
		if payload, ok := result.(map[string]interface{}); ok {
			if success, _ := payload["ok"].(bool); success {
				text, _ := payload["text"].(string)
				log.Infoln("[下一題] 已透過全局搜尋點擊按鈕：", text)
				humanPause(600, 1200)
				if isAnswerRoundComplete(page) {
					log.Infoln("[答題] 檢測到結果頁，本輪答題結束")
					return ErrAnswerComplete
				}
				if err := waitForAnswerAdvance(page, previousQuestionText, buttonSelectors); err != nil {
					return err
				}
				return nil
			}
		}
	}

	if lastErr != nil {
		log.Errorln("[下一題] 點擊按鈕失敗: " + lastErr.Error())
		return fmt.Errorf("点击答题操作按钮失败: %w", lastErr)
	}

	if snapshot, snapErr := page.Evaluate(`() => {
		const isVisible = (el) => {
			if (!el) return false;
			const rect = el.getBoundingClientRect();
			const style = window.getComputedStyle(el);
			return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
		};
		const normalize = (value) => String(value || "").replace(/\s+/g, " ").trim();
		const selectors = ["button", "[role='button']", ".ant-btn", "div", "span", "a"];
		const result = [];
		const seen = new Set();
		for (const selector of selectors) {
			for (const el of Array.from(document.querySelectorAll(selector)).slice(0, 200)) {
				if (!isVisible(el)) continue;
				if (el.closest(".q-body") || el.closest(".q-header")) continue;
				const text = normalize(el.innerText || el.textContent || "");
				if (!text || text.length > 20 || seen.has(selector + ":" + text)) continue;
				seen.add(selector + ":" + text);
				result.push({ selector, text });
				if (result.length >= 20) {
					return result;
				}
			}
		}
		return result;
	}`); snapErr == nil {
		log.Warningf("[下一題] 可見操作元素快照: %v", snapshot)
	}
	log.Warningln("[下一題] 未找到任何可點擊按鈕")
	return errors.New("未找到任何可点击的答题操作按钮")
}

// RemoveRepByLoop 通过两重循环过滤重复元素
func RemoveRepByLoop(slc []string) []string {
	var result []string // 存放结果
	for i := range slc {
		flag := true
		for j := range result {
			if slc[i] == result[j] {
				flag = false // 存在重复元素，标识为false
				break
			}
		}
		if flag { // 标识为false，不添加进结果
			result = append(result, slc[i])
		}
	}
	return result
}

// 获取专项答题ID
func getSpecialID(cookies []*http.Cookie) (int, error) {
	c := req.C()
	c.SetCommonCookies(cookies...)
	// 获取专项答题列表
	repo, err := c.R().SetQueryParams(map[string]string{"pageSize": "1000", "pageNo": "1"}).Get(querySpecialList)
	if err != nil {
		log.Errorln("获取专项答题列表错误" + err.Error())
		return 0, err
	}
	dataB64, err := repo.ToString()
	if err != nil {
		log.Errorln("获取专项答题列表获取string错误" + err.Error())
		return 0, err
	}
	// 因为返回内容使用base64编码，所以需要对内容进行转码
	data, err := base64.StdEncoding.DecodeString(gjson.Get(dataB64, "data_str").String())
	if err != nil {
		log.Errorln("获取专项答题列表转换b64错误" + err.Error())
		return 0, err
	}
	// 创建实例对象
	list := new(SpecialList)
	// json序列号
	err = json.Unmarshal(data, list)
	if err != nil {
		log.Errorln("获取专项答题列表转换json错误" + err.Error())
		return 0, err
	}
	log.Infoln(fmt.Sprintf("共获取到专项答题%d个", list.TotalCount))

	// 判断是否配置选题顺序，若ReverseOrder为true则从后面选题
	if conf.GetConfig().ReverseOrder {
		for i := len(list.List) - 1; i >= 0; i-- {
			if list.List[i].TipScore == 0 {
				log.Infoln(fmt.Sprintf("获取到未答专项答题: %v，id: %v", list.List[i].Name, list.List[i].Id))
				return list.List[i].Id, nil
			}
		}
	} else {
		for _, s := range list.List {
			if s.TipScore == 0 {
				log.Infoln(fmt.Sprintf("获取到未答专项答题: %v，id: %v", s.Name, s.Id))
				return s.Id, nil
			}
		}
	}
	log.Warningln("你已不存在未答的专项答题了")
	return 0, errors.New("未找到专项答题")
}

// 获取每周答题ID
func getweekID(cookies []*http.Cookie) (int, error) {
	c := req.C()
	c.SetCommonCookies(cookies...)
	repo, err := c.R().SetQueryParams(map[string]string{"pageSize": "500", "pageNo": "1"}).Get(queryWeekList)
	if err != nil {
		log.Errorln("获取每周答题列表错误" + err.Error())
		return 0, err
	}
	dataB64, err := repo.ToString()
	if err != nil {
		log.Errorln("获取每周答题列表获取string错误" + err.Error())
		return 0, err
	}
	data, err := base64.StdEncoding.DecodeString(gjson.Get(dataB64, "data_str").String())
	if err != nil {
		log.Errorln("获取每周答题列表转换b64错误" + err.Error())
		return 0, err
	}
	list := new(WeekList)
	err = json.Unmarshal(data, list)
	if err != nil {
		log.Errorln("获取每周答题列表转换json错误" + err.Error())
		return 0, err
	}
	log.Infoln(fmt.Sprintf("共获取到每周答题%d个", list.TotalCount))

	if conf.GetConfig().ReverseOrder {
		for i := len(list.List) - 1; i >= 0; i-- {
			for _, practice := range list.List[i].Practices {
				if practice.TipScore == 0 {
					log.Infoln(fmt.Sprintf("获取到未答每周答题: %v，id: %v", practice.Name, practice.Id))
					return practice.Id, nil
				}
			}
		}
	} else {
		for _, s := range list.List {
			for _, practice := range s.Practices {
				if practice.TipScore == 0 {
					log.Infoln(fmt.Sprintf("获取到未答每周答题: %v，id: %v", practice.Name, practice.Id))
					return practice.Id, nil
				}
			}
		}
	}
	log.Warningln("你已不存在未答的每周答题了")
	return 0, errors.New("未找到每周答题")
}

func GetSpecialContent(cookies []*http.Cookie, id int) *SpecialContent {
	response, err := utils.GetClient().R().SetCookies(cookies...).SetQueryParams(map[string]string{
		"type":   "2",
		"id":     strconv.Itoa(id),
		"forced": "true",
	}).Get("https://pc-proxy-api.xuexi.cn/api/exam/service/detail/queryV3")
	if err != nil {
		return nil
	}
	data, err2 := base64.StdEncoding.DecodeString(gjson.GetBytes(response.Bytes(), "data_str").String())
	if err2 != nil {
		log.Errorf("base64 decode failed: %v", err2)
		return nil
	}
	content := new(SpecialContent)
	if err3 := json.Unmarshal(data, content); err3 != nil {
		log.Errorf("json unmarshal failed: %v", err3)
		return nil
	}
	return content
}

// 获取每周答题ID列表
func GetweekIDs(cookies []*http.Cookie) []int {
	c := req.C()
	c.SetCommonCookies(cookies...)
	repo, err := c.R().SetQueryParams(map[string]string{"pageSize": "500", "pageNo": "1"}).Get(queryWeekList)
	if err != nil {
		log.Errorln("获取每周答题列表错误" + err.Error())
		return nil
	}
	dataB64, err := repo.ToString()
	if err != nil {
		log.Errorln("获取每周答题列表获取string错误" + err.Error())
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(gjson.Get(dataB64, "data_str").String())
	if err != nil {
		log.Errorln("获取每周答题列表转换b64错误" + err.Error())
		return nil
	}
	list := new(WeekList)
	err = json.Unmarshal(data, list)
	if err != nil {
		log.Errorln("获取每周答题列表转换json错误" + err.Error())
		return nil
	}
	log.Infoln(fmt.Sprintf("共获取到每周答题%d个", list.TotalCount))
	var ids []int
	for _, l := range list.List {
		for _, practice := range l.Practices {
			ids = append(ids, practice.Id)
		}
	}
	return ids
}

// 获取专项答题ID列表
func GetSpecialIDs(cookies []*http.Cookie) []int {
	c := req.C()

	c.SetCommonCookies(cookies...)
	// 获取专项答题列表
	repo, err := c.R().SetQueryParams(map[string]string{"pageSize": "1000", "pageNo": "1"}).Get(querySpecialList)
	if err != nil {
		log.Errorln("获取专项答题列表错误" + err.Error())
		return nil
	}
	dataB64, err := repo.ToString()
	if err != nil {
		log.Errorln("获取专项答题列表获取string错误" + err.Error())
		return nil
	}
	// 因为返回内容使用base64编码，所以需要对内容进行转码
	data, err := base64.StdEncoding.DecodeString(gjson.Get(dataB64, "data_str").String())
	if err != nil {
		log.Errorln("获取专项答题列表转换b64错误" + err.Error())
		return nil
	}
	// 创建实例对象
	list := new(SpecialList)
	// json序列号
	err = json.Unmarshal(data, list)
	if err != nil {
		log.Errorln("获取专项答题列表转换json错误" + err.Error())
		return nil
	}
	log.Infoln(fmt.Sprintf("共获取到专项答题%d个", list.TotalCount))
	var ids []int
	for _, l := range list.List {
		ids = append(ids, l.Id)
	}
	return ids
}

type SpecialContent struct {
	Perfect   bool `json:"perfect"`
	TotalTime int  `json:"totalTime"`
	Questions []struct {
		HasDescribe bool `json:"hasDescribe"`
		// 提示信息
		QuestionDesc string `json:"questionDesc"`
		QuestionId   int    `json:"questionId"`
		Origin       string `json:"origin"`
		// 答案
		Answers []struct {
			AnswerId int    `json:"answerId"`
			Label    string `json:"label"`
			Content  string `json:"content"`
		} `json:"answers"`
		QuestionScore int `json:"questionScore"`
		// 题目呢偶然
		Body               string `json:"body"`
		OriginTitle        string `json:"originTitle"`
		AllCorrect         bool   `json:"allCorrect"`
		Supplier           string `json:"supplier"`
		QuestionDescOrigin string `json:"questionDescOrigin"`
		QuestionDisplay    int    `json:"questionDisplay"`
		Recommender        string `json:"recommender"`
	} `json:"questions"`
	Type               int    `json:"type"`
	TotalScore         int    `json:"totalScore"`
	PassScore          int    `json:"passScore"`
	FinishedNum        int    `json:"finishedNum"`
	UsedTime           int    `json:"usedTime"`
	Name               string `json:"name"`
	QuestionNum        int    `json:"questionNum"`
	Id                 int    `json:"id"`
	UniqueId           string `json:"uniqueId"`
	TipScoreReasonType int    `json:"tipScoreReasonType"`
}
