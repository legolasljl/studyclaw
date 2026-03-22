/* eslint-disable react/jsx-no-target-blank */
import React, { Component } from "react";
import { getAbout } from "../../utils/api";

const workflowNotes = [
  "主流程固定為先讀文章，再補足音頻學習。",
  "每日答題代碼仍保留在專案內，但目前不接入正式學習流程。",
  "完成通知只回報總積分、今日得分、文章學習、視頻學習與本次用時。",
];

const accessNotes = [
  "管理員登入帳密來自配置檔中的 web.account / web.password。",
  "普通用戶由 common_user 管理，支援多帳號並限制只能看自己的資料。",
  "新增帳戶請到「用戶管理」產生授權二維碼，再用學習強國 App 掃碼接入。",
];

const deployNotes = [
  "正式對外入口改為 `/studyclaw/`，分享部署網址時請優先使用這個路徑。",
  "根路徑 `/` 仍可使用，會自動導向 `/studyclaw/`。",
  "若頁面打不開，先檢查後端日誌、埠映射與瀏覽器快取。",
];

const notificationTemplate = [
  "xx帳號 已學習完成",
  "當前學習總積分：",
  "今日得分：",
  "本次用時：",
  "文章學習： /12",
  "視頻學習： /12",
];

class Help extends Component<any, any> {
  constructor(props: any) {
    super(props);
    this.state = {
      about: "",
    };
  }

  componentDidMount() {
    getAbout().then((value) => {
      this.setState({
        about: value.data,
      });
    });
  }

  renderList(items: string[]) {
    return (
      <ul>
        {items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    );
  }

  render() {
    return (
      <section className="page-section">
        <div className="section-heading">
          <div>
            <span className="section-kicker">Manual</span>
            <h2>使用說明</h2>
            <p>只保留目前仍有效的接入、部署與通知資訊，避免舊功能描述干擾操作。</p>
          </div>
        </div>

        <div className="manual-grid">
          <article className="page-card manual-card manual-card--wide">
            <span className="section-kicker">Project</span>
            <h3>部署與 Web 入口</h3>
            <p>{this.state.about || "正式入口維持在 /studyclaw/，前端與 API 共用同一條管理鏈路。"}</p>
            {this.renderList(deployNotes)}
            <div className="manual-link-row">
              <a href="https://github.com/legolasljl/studyclaw">GitHub Repo</a>
            </div>
          </article>

          <article className="page-card manual-card">
            <span className="section-kicker">Workflow</span>
            <h3>目前工作流</h3>
            {this.renderList(workflowNotes)}
          </article>

          <article className="page-card manual-card">
            <span className="section-kicker">Access</span>
            <h3>登入與多帳號</h3>
            {this.renderList(accessNotes)}
          </article>

          <article className="page-card manual-card">
            <span className="section-kicker">Notification</span>
            <h3>完成通知格式</h3>
            <pre>
              <code>{notificationTemplate.join("\n")}</code>
            </pre>
          </article>

          <article className="page-card manual-card">
            <span className="section-kicker">Suggested Config</span>
            <h3>建議配置</h3>
            <pre>
              <code>{`web:
  enable: true
  host: 0.0.0.0
  port: 8080
  account: admin
  password: admin
  common_user:
    user1: password1
    user2: password2`}</code>
            </pre>
          </article>
        </div>
      </section>
    );
  }
}

export default Help;
