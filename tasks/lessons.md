# Lessons

- 當使用者提供執行日誌指出實際卡點時，優先相信執行證據，重新檢查該熱路徑，而不要只依賴前一輪的靜態 diff 判斷。
- 在 Playwright 流程中，只要有 sub-second 輪詢，就必須先排查 `page.Evaluate(document.body.innerText)`、寬泛 `QuerySelectorAll()`、多次 `TextContent()` 這類高成本操作；它們比一般 Go 迴圈更容易把 CPU 打滿。
- 對驗證流程的偵測要分成「快速檢查」與「完整檢查」兩級，避免把完整頁面文字抽取放進緊密輪詢。
- 收斂偵測條件時不能直接砍掉備援路徑；正確做法是把寬鬆偵測移到低頻深度檢查，而不是完全移除，否則真實驗證彈窗會被漏檢。
