package lib

import (
	"errors"
	"testing"
	"time"
)

func TestNewAuthChecker(t *testing.T) {
	tests := []struct {
		name              string
		config            AuthCheckerConfig
		maxFailures       int
		cooldownPeriod    time.Duration
		maxSliderFailures int
		moduleName        string
	}{
		{
			name: "default values",
			config: AuthCheckerConfig{
				ModuleName: "test",
			},
			maxFailures:       3,
			cooldownPeriod:    5 * time.Minute,
			maxSliderFailures: 3,
			moduleName:        "test",
		},
		{
			name: "custom values",
			config: AuthCheckerConfig{
				MaxFailures:       5,
				CooldownPeriod:    10 * time.Minute,
				MaxSliderFailures: 2,
				ModuleName:        "custom",
			},
			maxFailures:       5,
			cooldownPeriod:    10 * time.Minute,
			maxSliderFailures: 2,
			moduleName:        "custom",
		},
		{
			name: "zero values use defaults",
			config: AuthCheckerConfig{
				MaxFailures:       0,
				CooldownPeriod:    0,
				MaxSliderFailures: 0,
				ModuleName:        "",
			},
			maxFailures:       3,
			cooldownPeriod:    5 * time.Minute,
			maxSliderFailures: 3,
			moduleName:        "未知模組",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewAuthChecker(tt.config)
			if checker.maxFailures != tt.maxFailures {
				t.Errorf("maxFailures = %d, want %d", checker.maxFailures, tt.maxFailures)
			}
			if checker.cooldownPeriod != tt.cooldownPeriod {
				t.Errorf("cooldownPeriod = %v, want %v", checker.cooldownPeriod, tt.cooldownPeriod)
			}
			if checker.maxSliderFailures != tt.maxSliderFailures {
				t.Errorf("maxSliderFailures = %d, want %d", checker.maxSliderFailures, tt.maxSliderFailures)
			}
			if checker.moduleName != tt.moduleName {
				t.Errorf("moduleName = %s, want %s", checker.moduleName, tt.moduleName)
			}
		})
	}
}

func TestAuthChecker_RecordFailure(t *testing.T) {
	checker := NewAuthChecker(AuthCheckerConfig{
		MaxFailures: 3,
		ModuleName:  "test",
	})

	// First two failures should allow continuation
	if !checker.RecordFailure(errors.New("error 1")) {
		t.Error("RecordFailure should return true for first failure")
	}
	if checker.GetFailureCount() != 1 {
		t.Errorf("failureCount = %d, want 1", checker.GetFailureCount())
	}

	if !checker.RecordFailure(errors.New("error 2")) {
		t.Error("RecordFailure should return true for second failure")
	}
	if checker.GetFailureCount() != 2 {
		t.Errorf("failureCount = %d, want 2", checker.GetFailureCount())
	}

	// Third failure should indicate stop
	if checker.RecordFailure(errors.New("error 3")) {
		t.Error("RecordFailure should return false for third failure (max reached)")
	}
	if checker.GetFailureCount() != 3 {
		t.Errorf("failureCount = %d, want 3", checker.GetFailureCount())
	}

	// Should stop now
	if !checker.ShouldStop() {
		t.Error("ShouldStop should return true after max failures")
	}
}

func TestAuthChecker_RecordSliderFailure(t *testing.T) {
	checker := NewAuthChecker(AuthCheckerConfig{
		MaxSliderFailures: 2,
		ModuleName:        "test",
	})

	// First slider failure should allow continuation
	if !checker.RecordSliderFailure() {
		t.Error("RecordSliderFailure should return true for first failure")
	}

	// Second slider failure should indicate stop
	if checker.RecordSliderFailure() {
		t.Error("RecordSliderFailure should return false for second failure (max reached)")
	}

	// Should stop slider now
	if !checker.ShouldStopSlider() {
		t.Error("ShouldStopSlider should return true after max slider failures")
	}
}

func TestAuthChecker_Reset(t *testing.T) {
	checker := NewAuthChecker(AuthCheckerConfig{
		MaxFailures:       3,
		MaxSliderFailures: 2,
		ModuleName:        "test",
	})

	// Record some failures
	checker.RecordFailure(errors.New("error 1"))
	checker.RecordFailure(errors.New("error 2"))
	checker.RecordSliderFailure()

	// Verify failures recorded
	if checker.GetFailureCount() != 2 {
		t.Errorf("failureCount = %d, want 2", checker.GetFailureCount())
	}

	// Reset
	checker.Reset()

	// Verify reset
	if checker.GetFailureCount() != 0 {
		t.Errorf("failureCount after reset = %d, want 0", checker.GetFailureCount())
	}
	if checker.ShouldStop() {
		t.Error("ShouldStop should return false after reset")
	}
	if checker.ShouldStopSlider() {
		t.Error("ShouldStopSlider should return false after reset")
	}
}

func TestAuthChecker_IsInCooldown(t *testing.T) {
	checker := NewAuthChecker(AuthCheckerConfig{
		CooldownPeriod: 100 * time.Millisecond,
		ModuleName:     "test",
	})

	// Initially not in cooldown
	if checker.IsInCooldown() {
		t.Error("IsInCooldown should return false initially")
	}

	// Record a failure (sets lastFailureTime)
	checker.RecordFailure(errors.New("error"))

	// Should be in cooldown
	if !checker.IsInCooldown() {
		t.Error("IsInCooldown should return true after failure")
	}

	// Wait for cooldown to expire
	time.Sleep(150 * time.Millisecond)

	// Should not be in cooldown anymore
	if checker.IsInCooldown() {
		t.Error("IsInCooldown should return false after cooldown period")
	}
}

func TestCheckAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "login error",
			err:      errors.New("登录失败"),
			expected: true,
		},
		{
			name:     "login error English",
			err:      errors.New("login required"),
			expected: true,
		},
		{
			name:     "cookie error",
			err:      errors.New("cookie expired"),
			expected: true,
		},
		{
			name:     "auth error",
			err:      errors.New("authentication failed"),
			expected: true,
		},
		{
			name:     "secure_check error",
			err:      errors.New("redirect to secure_check"),
			expected: true,
		},
		{
			name:     "network error",
			err:      errors.New("connection timeout"),
			expected: false,
		},
		{
			name:     "other error",
			err:      errors.New("some random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckAuthError(tt.err)
			if result != tt.expected {
				t.Errorf("CheckAuthError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCheckNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "timeout error",
			err:      errors.New("connection timeout"),
			expected: true,
		},
		{
			name:     "network error",
			err:      errors.New("network unreachable"),
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "dns error",
			err:      errors.New("dns lookup failed"),
			expected: true,
		},
		{
			name:     "EOF error",
			err:      errors.New("unexpected EOF"),
			expected: true,
		},
		{
			name:     "auth error",
			err:      errors.New("login required"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("CheckNetworkError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "auth error",
			err:      errors.New("login required"),
			expected: "登入失效",
		},
		{
			name:     "network error",
			err:      errors.New("connection timeout"),
			expected: "網路異常",
		},
		{
			name:     "unknown error",
			err:      errors.New("something went wrong"),
			expected: "未知錯誤",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CategorizeError(tt.err)
			if result != tt.expected {
				t.Errorf("CategorizeError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAuthError(t *testing.T) {
	cause := errors.New("original error")
	authErr := NewAuthError("test message", cause)

	// Test Error() method
	errMsg := authErr.Error()
	if errMsg == "" {
		t.Error("Error() should return non-empty string")
	}

	// Test Unwrap() method
	unwrapped := authErr.Unwrap()
	if unwrapped != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestNetworkError(t *testing.T) {
	cause := errors.New("original error")
	netErr := NewNetworkError("test message", cause)

	// Test Error() method
	errMsg := netErr.Error()
	if errMsg == "" {
		t.Error("Error() should return non-empty string")
	}

	// Test Unwrap() method
	unwrapped := netErr.Unwrap()
	if unwrapped != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestFormatErrorMessage(t *testing.T) {
	err := errors.New("login required")
	msg := FormatErrorMessage("TestModule", err, 1, 3)

	if msg == "" {
		t.Error("FormatErrorMessage should return non-empty string")
	}

	// Check that message contains expected parts
	if !containsAll(msg, "TestModule", "登入失效", "1", "3") {
		t.Errorf("FormatErrorMessage = %q, should contain module name, error type, and counts", msg)
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !contains(s, substr) {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}