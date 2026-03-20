package lib

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// AuthChecker 統一的登入狀態檢測器
// 用於追蹤連續失敗次數並判斷是否應該提前停止
type AuthChecker struct {
	mu                sync.Mutex
	maxFailures       int           // 最大連續失敗次數
	failureCount      int           // 當前失敗計數
	cooldownPeriod    time.Duration // 冷卻時間
	lastFailureTime   time.Time     // 上次失敗時間
	sliderFailCount   int           // 滑塊驗證失敗計數
	maxSliderFailures int           // 滑塊驗證最大失敗次數
	moduleName        string        // 模組名稱，用於日誌
}

// AuthCheckerConfig 配置參數
type AuthCheckerConfig struct {
	MaxFailures       int           // 最大連續失敗次數
	CooldownPeriod    time.Duration // 冷卻時間
	MaxSliderFailures int           // 滑塊驗證最大失敗次數
	ModuleName        string        // 模組名稱
}

// NewAuthChecker 建立新的 AuthChecker 實例
func NewAuthChecker(config AuthCheckerConfig) *AuthChecker {
	// 設定預設值
	if config.MaxFailures <= 0 {
		config.MaxFailures = 3
	}
	if config.CooldownPeriod <= 0 {
		config.CooldownPeriod = 5 * time.Minute
	}
	if config.MaxSliderFailures <= 0 {
		config.MaxSliderFailures = 3
	}
	if config.ModuleName == "" {
		config.ModuleName = "未知模組"
	}

	return &AuthChecker{
		maxFailures:       config.MaxFailures,
		cooldownPeriod:    config.CooldownPeriod,
		maxSliderFailures: config.MaxSliderFailures,
		moduleName:        config.ModuleName,
	}
}

// RecordFailure 記錄一次失敗
// 返回是否應該繼續執行
func (a *AuthChecker) RecordFailure(err error) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.failureCount++
	a.lastFailureTime = time.Now()

	if err != nil {
		log.Warningf("[%s] 第 %d 次失敗: %v", a.moduleName, a.failureCount, err)
	} else {
		log.Warningf("[%s] 第 %d 次失敗", a.moduleName, a.failureCount)
	}

	return a.failureCount < a.maxFailures
}

// RecordAuthFailure 記錄登入相關失敗（更嚴重）
// 登入失效類錯誤會導致提前停止
func (a *AuthChecker) RecordAuthFailure(err error) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.failureCount++
	a.lastFailureTime = time.Now()

	log.Errorf("[%s] 登入狀態異常 (第 %d 次): %v", a.moduleName, a.failureCount, err)

	// 登入失效類錯誤，直接計入嚴重失敗
	return a.failureCount < a.maxFailures
}

// RecordSliderFailure 記錄滑塊驗證失敗
// 返回是否應該繼續嘗試
func (a *AuthChecker) RecordSliderFailure() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.sliderFailCount++
	log.Warningf("[%s] 滑塊驗證失敗第 %d 次", a.moduleName, a.sliderFailCount)

	return a.sliderFailCount < a.maxSliderFailures
}

// Reset 重置失敗計數
func (a *AuthChecker) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.failureCount = 0
	a.sliderFailCount = 0
	log.Debugf("[%s] 失敗計數已重置", a.moduleName)
}

// ShouldStop 判斷是否應該提前停止
func (a *AuthChecker) ShouldStop() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.failureCount >= a.maxFailures
}

// ShouldStopSlider 判斷滑塊驗證是否應該停止
func (a *AuthChecker) ShouldStopSlider() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.sliderFailCount >= a.maxSliderFailures
}

// GetFailureCount 獲取當前失敗計數
func (a *AuthChecker) GetFailureCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.failureCount
}

// IsInCooldown 判斷是否在冷卻期內
func (a *AuthChecker) IsInCooldown() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.lastFailureTime.IsZero() {
		return false
	}

	return time.Since(a.lastFailureTime) < a.cooldownPeriod
}

// CheckAuthError 檢查錯誤是否為登入失效相關錯誤
// 返回是否為登入失效錯誤
func CheckAuthError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	authErrorPatterns := []string{
		"登录",      // 登入
		"login",    // 登入
		"cookie",   // cookie 相關
		"认证",      // 認證
		"auth",     // 認證
		"secure_check", // 安全檢查頁面
		"过期",      // 過期
		"expired",  // 過期
		"token",    // token 相關
		"未授權",     // 未授權
		"unauthorized", // 未授權
	}

	for _, pattern := range authErrorPatterns {
		if strings.Contains(strings.ToLower(errMsg), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// CheckNetworkError 檢查錯誤是否為網路相關錯誤
// 返回是否為網路錯誤
func CheckNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	networkErrorPatterns := []string{
		"timeout",      // 超時
		"connection",   // 連線
		"network",      // 網路
		"dns",          // DNS
		"refused",      // 連線被拒絕
		"reset",        // 連線重置
		"broken pipe",  // 管道斷裂
		"EOF",          // 意外結束
	}

	for _, pattern := range networkErrorPatterns {
		if strings.Contains(strings.ToLower(errMsg), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// CategorizeError 分類錯誤類型
// 返回錯誤類型描述
func CategorizeError(err error) string {
	if err == nil {
		return ""
	}

	if CheckAuthError(err) {
		return "登入失效"
	}

	if CheckNetworkError(err) {
		return "網路異常"
	}

	return "未知錯誤"
}

// FormatErrorMessage 格式化錯誤訊息
func FormatErrorMessage(moduleName string, err error, failureCount int, maxFailures int) string {
	errorType := CategorizeError(err)
	return fmt.Sprintf("[%s] %s (第 %d/%d 次): %v",
		moduleName, errorType, failureCount, maxFailures, err)
}

// AuthError 登入失效錯誤
type AuthError struct {
	Message string
	Cause   error
}

func (e *AuthError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("登入失效: %s (原因: %v)", e.Message, e.Cause)
	}
	return fmt.Sprintf("登入失效: %s", e.Message)
}

func (e *AuthError) Unwrap() error {
	return e.Cause
}

// NewAuthError 建立登入失效錯誤
func NewAuthError(message string, cause error) *AuthError {
	return &AuthError{
		Message: message,
		Cause:   cause,
	}
}

// NetworkError 網路錯誤
type NetworkError struct {
	Message string
	Cause   error
}

func (e *NetworkError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("網路異常: %s (原因: %v)", e.Message, e.Cause)
	}
	return fmt.Sprintf("網路異常: %s", e.Message)
}

func (e *NetworkError) Unwrap() error {
	return e.Cause
}

// NewNetworkError 建立網路錯誤
func NewNetworkError(message string, cause error) *NetworkError {
	return &NetworkError{
		Message: message,
		Cause:   cause,
	}
}