# 滑塊驗證效能回歸審查與穩定化修補

- [x] 確認目標 commit 與工作樹狀態
- [x] 讀取 `b292674` diff 與相關呼叫路徑
- [x] 檢查是否存在忙迴圈、無界重試、資源未釋放或 goroutine 膨脹
- [x] 以測試或實際執行重現/驗證判斷
- [x] 整理 review findings、風險與建議修正
- [x] 先補滑塊文本判定的定向測試
- [x] 收窄滑塊偵測：精確 selector 優先、移除泛化文本誤判
- [x] 降低 `study.go` 背景滑鼠漂移成本，保留低頻可信事件
- [x] 跑定向測試與基本編譯驗證

## Review

- 主要回歸點 1：`lib/study.go` 將原本高頻迴圈中的 `page.Evaluate(MouseEvent)` 改成 Playwright 原生 `Mouse.Move(..., Steps)`；這條路徑在文章/影片/音頻學習每秒都可能觸發，會把瀏覽器層級輸入事件量放大。
- 主要回歸點 2：`waitForSystemJudgment()` 新增「完成」按鈕可點擊後再次檢查滑塊；但 `hasAnswerSliderPrompt()` 的 selector / 文本規則很寬，可能把非 CAPTCHA DOM 判成滑塊，導致進入昂貴的滑塊重試流程。
- 驗證補充：`go test` 與 `go test ./lib/... ./model/...` 皆因既有測試/資料庫問題失敗，無法作為此次回歸的直接驗證證據；結論主要來自 diff + 呼叫頻率 + 執行路徑分析。
- 修補驗證：`go test ./lib -run 'TestContainsAnswerSliderSpecificText|TestNormalizeAnswerButtonText|TestIsAnswerCompletionText'` 通過；`go test ./lib -run TestDecodeSliderPosition` 通過；`go test ./lib -run '^$'` 編譯通過。
- 二次修補驗證：`go test ./lib -run 'TestContainsAnswerSliderSpecificText|TestContainsAnswerSliderLooseText|TestContainsAnswerFlowBlockedText|TestNormalizeAnswerButtonText|TestIsAnswerCompletionText|TestDecodeSliderPosition'` 通過；`go test ./lib -run '^$'` 編譯通過。
