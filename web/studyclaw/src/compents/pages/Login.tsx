import React, { FormEvent, useState } from "react";

import "../../stylel.css";
import { checkToken, login } from "../../utils/api";

type LoginProps = {
  navigate: (path: string) => void;
};

function Login(props: LoginProps) {
  const [account, setAccount] = useState("admin");
  const [password, setPassword] = useState("admin");
  const [showGuide, setShowGuide] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const submit = async (event: FormEvent) => {
    event.preventDefault();

    if (!account.trim() || !password.trim()) {
      setError("請輸入管理帳號與密碼。");
      return;
    }

    setSubmitting(true);
    setError("");

    try {
      const payload = JSON.stringify({
        account: account.trim(),
        password: password.trim(),
      });
      const resp = await login(payload);

      if (!resp.success) {
        setError(resp.message || "登入失敗，請確認配置中的 Web 帳號密碼。");
        return;
      }

      window.localStorage.setItem("studyclaw_token", resp.data);
      const tokenInfo = await checkToken();

      if (!tokenInfo) {
        setError("登入態校驗失敗，請稍後重試。");
        return;
      }

      sessionStorage.setItem("level", tokenInfo.data === 1 ? "1" : "2");
      props.navigate("/home");
    } catch (_error) {
      setError("登入請求失敗，請檢查服務是否可用。");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className={`login-shell${showGuide ? " login-shell--guide" : ""}`}>
      <div className="login-shell__backdrop" />
      <div className="login-layout">
        <section className="login-aside">
          <span className="login-eyebrow">STUDYCLAW / ACCESS</span>
          <h1>收斂成可部署、可觀察、可多帳號接入的學習控制台。</h1>
          <p>
            這裡只保留必要的後台能力。登入後即可管理帳戶、監控文章與音頻積分，
            並直接查看配置與日誌。
          </p>

          <div className="login-aside__metrics">
            <div>
              <strong>文章 12 / 視頻 12</strong>
              <span>唯一學習目標</span>
            </div>
            <div>
              <strong>WEB / LOG / CONFIG</strong>
              <span>同一條管理鏈路</span>
            </div>
          </div>

          <div className="login-aside__footer">
            <button className="panel-switch" onClick={() => setShowGuide(!showGuide)}>
              {showGuide ? "返回登入" : "查看接入流程"}
            </button>
            <a href="https://github.com/legolasljl/studyclaw" target="_blank" rel="noreferrer">
              GitHub 專案
            </a>
          </div>
        </section>

        <section className="login-card">
          {!showGuide ? (
            <>
              <div className="login-card__header">
                <span className="login-card__label">Access Gate</span>
                <h2>進入控制台</h2>
                <p>預設帳號密碼為 admin / admin，實際以配置檔中的 Web 帳密為準。</p>
              </div>

              <form className="login-form" onSubmit={submit}>
                <label className="login-field">
                  <span>帳號</span>
                  <input
                    value={account}
                    onChange={(event) => setAccount(event.target.value)}
                    autoComplete="username"
                    placeholder="輸入管理帳號"
                  />
                </label>

                <label className="login-field">
                  <span>密碼</span>
                  <input
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    type="password"
                    autoComplete="current-password"
                    placeholder="輸入管理密碼"
                  />
                </label>

                {error ? <p className="login-error">{error}</p> : null}

                <button type="submit" className="login-submit" disabled={submitting}>
                  {submitting ? "登入中..." : "登入控制台"}
                </button>
              </form>
            </>
          ) : (
            <div className="guide-panel">
              <span className="login-card__label">Procedure</span>
              <h2>首次接入流程</h2>
              <ol className="guide-list">
                <li>使用 Web 管理員帳號登入後台。</li>
                <li>進入「用戶管理」，點擊「添加用戶」生成登入二維碼。</li>
                <li>掃碼授權後，即可查看分數、啟動學習與追蹤日誌。</li>
                <li>管理員可在「管理台」直接編輯配置與查看運行輸出。</li>
              </ol>

              <div className="guide-note">
                <strong>專案地址</strong>
                <a href="https://github.com/legolasljl/studyclaw" target="_blank" rel="noreferrer">
                  https://github.com/legolasljl/studyclaw
                </a>
              </div>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

export default Login;
