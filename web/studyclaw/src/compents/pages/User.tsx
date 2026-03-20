import React, { Component } from "react";
import { Dialog, Toast } from "antd-mobile";

import { deleteUser, getExpiredUsers, getScore, getUsers, stopStudy, study } from "../../utils/api";

type ScoreSnapshot = {
  nick: string;
  totalScore: number;
  todayScore: number;
  articleScore: string;
  articlePercentage: string;
  videoScore: string;
  videoPercentage: string;
  dailyScore: string;
  dailyPercentage: string;
};

type UserRecord = {
  nick: string;
  uid: string;
  token: string;
  login_time: number;
  is_study: boolean;
};

const emptyScore: ScoreSnapshot = {
  nick: "",
  totalScore: 0,
  todayScore: 0,
  articleScore: "0/0",
  articlePercentage: "0%",
  videoScore: "0/0",
  videoPercentage: "0%",
  dailyScore: "0/0",
  dailyPercentage: "0%",
};

class Users extends Component<any, any> {
  constructor(props: any) {
    super(props);
    this.state = {
      level: "2",
      users: [] as UserRecord[],
      expiredUsers: [] as UserRecord[],
      searchTerm: "",
      filterType: "全部",
      sortBy: "login_desc",
      scoreSnapshots: {} as Record<string, ScoreSnapshot>,
      selectedScore: null as ScoreSnapshot | null,
      loadingScore: false,
    };
  }

  static SummaryPanel(users: UserRecord[], expiredUsers: UserRecord[]) {
    const totalUsers = users.length + expiredUsers.length;
    const activeUsers = users.length;
    const inactiveUsers = expiredUsers.length;
    const studyingUsers = users.filter((user) => user.is_study).length;

    return {
      totalUsers,
      activeUsers,
      inactiveUsers,
      studyingUsers,
    };
  }

  componentDidMount() {
    this.fetchUserData();
    this.setState({
      level: sessionStorage.getItem("level") || "2",
    });
  }

  formatTimestamp = (value: number) => {
    const date = new Date(value * 1000);
    return `${date.getFullYear()}年${`${date.getMonth() + 1}`.padStart(2, "0")}月${`${date.getDate()}`.padStart(2, "0")}日 ${`${date.getHours()}`.padStart(2, "0")}:${`${date.getMinutes()}`.padStart(2, "0")}:${`${date.getSeconds()}`.padStart(2, "0")}`;
  };

  getOnlineDays = (value: number) => {
    const now = Date.now();
    return Math.max(Math.floor((now - value * 1000) / (1000 * 60 * 60 * 24)), 0);
  };

  convertToPercentage = (current: number, total: number) => {
    if (total <= 0) {
      return "0%";
    }
    return `${Math.round((current / total) * 100)}%`;
  };

  parseScorePayload = (payload: string, nick = ""): ScoreSnapshot => {
    const totalScoreMatch = payload.match(/(?:當前學習總積分|当前学习总积分)[:：](\d+)/);
    const todayScoreMatch = payload.match(/今日得分：(\d+)/);
    const articleScoreMatch = payload.match(/(?:文章學習|文章学习)[:：](\d+)\/(\d+)/);
    const videoScoreMatch = payload.match(/(?:視頻學習|视频学习)[:：](\d+)\/(\d+)/);
    const dailyScoreMatch = payload.match(/(?:每日答題|每日答题)[:：](\d+)\/(\d+)/);

    const articleCurrent = articleScoreMatch ? parseInt(articleScoreMatch[1], 10) : 0;
    const articleMax = articleScoreMatch ? parseInt(articleScoreMatch[2], 10) : 0;
    const videoCurrent = videoScoreMatch ? parseInt(videoScoreMatch[1], 10) : 0;
    const videoMax = videoScoreMatch ? parseInt(videoScoreMatch[2], 10) : 0;
    const dailyCurrent = dailyScoreMatch ? parseInt(dailyScoreMatch[1], 10) : 0;
    const dailyMax = dailyScoreMatch ? parseInt(dailyScoreMatch[2], 10) : 0;

    return {
      nick,
      totalScore: totalScoreMatch ? parseInt(totalScoreMatch[1], 10) : 0,
      todayScore: todayScoreMatch ? parseInt(todayScoreMatch[1], 10) : 0,
      articleScore: `${articleCurrent}/${articleMax}`,
      articlePercentage: this.convertToPercentage(articleCurrent, articleMax),
      videoScore: `${videoCurrent}/${videoMax}`,
      videoPercentage: this.convertToPercentage(videoCurrent, videoMax),
      dailyScore: `${dailyCurrent}/${dailyMax}`,
      dailyPercentage: this.convertToPercentage(dailyCurrent, dailyMax),
    };
  };

  fetchScoreSnapshot = async (token: string, nick = "") => {
    try {
      const resp = await getScore(token);
      return this.parseScorePayload(resp.data, nick);
    } catch (_error) {
      return { ...emptyScore, nick };
    }
  };

  fetchUserData = async () => {
    const [usersResp, expiredResp] = await Promise.all([getUsers(), getExpiredUsers()]);
    const users = usersResp.data || [];
    const expiredUsers = expiredResp.data || [];
    const allUsers = [...users, ...expiredUsers];
    const scoreEntries = await Promise.all(
      allUsers.map(async (user: UserRecord) => [user.token, await this.fetchScoreSnapshot(user.token, user.nick)] as const),
    );

    const scoreSnapshots = scoreEntries.reduce((acc, [token, score]) => {
      acc[token] = score;
      return acc;
    }, {} as Record<string, ScoreSnapshot>);

    this.setState({
      users,
      expiredUsers,
      scoreSnapshots,
      level: sessionStorage.getItem("level") || "2",
    });
  };

  openScore = async (token: string, nick: string) => {
    this.setState({ loadingScore: true });
    const selectedScore = await this.fetchScoreSnapshot(token, nick);
    this.setState({
      selectedScore,
      loadingScore: false,
    });
  };

  closeScore = () => {
    this.setState({
      selectedScore: null,
      loadingScore: false,
    });
  };

  toggleStudy = async (uid: string, isStudy: boolean, isExpired: boolean) => {
    if (isExpired) {
      Toast.show("該帳戶已失效，請先重新掃碼接入。");
      return;
    }

    try {
      if (isStudy) {
        await stopStudy(uid);
        Toast.show("已停止學習");
      } else {
        await study(uid);
        Toast.show("開始學習成功");
      }
      await this.fetchUserData();
    } catch (_error) {
      Toast.show(isStudy ? "停止學習失敗，請稍後重試。" : "開始學習失敗，請稍後重試。");
    }
  };

  deleteUserRecord = (uid: string, nick: string) => {
    Dialog.confirm({ content: `確定刪除用戶 ${nick} 嗎？` }).then(async (confirmed: boolean) => {
      if (!confirmed) {
        return;
      }

      const resp = await deleteUser(uid);
      if (!resp.success) {
        Dialog.show({ content: resp.error || "刪除失敗", closeOnMaskClick: true, closeOnAction: true });
        return;
      }
      await this.fetchUserData();
    });
  };

  handleSearch = (event: { target: { value: string } }) => {
    this.setState({ searchTerm: event.target.value });
  };

  setFilter = (filterType: string) => {
    this.setState({ filterType });
  };

  setSort = (sortBy: string) => {
    this.setState({ sortBy });
  };

  matchesFilter = (user: UserRecord, isExpired: boolean) => {
    const { filterType } = this.state;
    if (filterType === "全部") {
      return true;
    }
    if (filterType === "學習中") {
      return !isExpired && user.is_study;
    }
    if (filterType === "待機中") {
      return !isExpired && !user.is_study;
    }
    if (filterType === "已失效") {
      return isExpired;
    }
    return true;
  };

  sortUsers = (users: UserRecord[]) => {
    const { sortBy, scoreSnapshots } = this.state;
    return [...users].sort((left, right) => {
      switch (sortBy) {
        case "nick_asc":
          return left.nick.localeCompare(right.nick, "zh-Hans-CN");
        case "today_desc":
          return (scoreSnapshots[right.token]?.todayScore || 0) - (scoreSnapshots[left.token]?.todayScore || 0);
        case "total_desc":
          return (scoreSnapshots[right.token]?.totalScore || 0) - (scoreSnapshots[left.token]?.totalScore || 0);
        case "login_asc":
          return left.login_time - right.login_time;
        case "login_desc":
        default:
          return right.login_time - left.login_time;
      }
    });
  };

  getVisibleUsers = (users: UserRecord[], isExpired: boolean) => {
    const { searchTerm } = this.state;
    const normalizedSearch = searchTerm.trim().toLowerCase();

    return this.sortUsers(
      users.filter((user) => {
        const matchesSearch =
          normalizedSearch === "" ||
          user.nick.toLowerCase().includes(normalizedSearch) ||
          user.uid.toLowerCase().includes(normalizedSearch);
        return matchesSearch && this.matchesFilter(user, isExpired);
      }),
    );
  };

  renderUserCard = (user: UserRecord, isExpired: boolean) => {
    const score = this.state.scoreSnapshots[user.token] || { ...emptyScore, nick: user.nick };
    const lastCharacter = user.nick ? user.nick[user.nick.length - 1] : "?";

    return (
      <article key={user.uid} className={`user-card${isExpired ? " user-card--expired" : ""}`}>
        <div className="user-card__header">
          <div className="user-card__identity">
            <span className="user-card__avatar">{lastCharacter}</span>
            <div>
              <span className="user-chip">{isExpired ? "已失效" : user.is_study ? "學習中" : "待機中"}</span>
              <h3>{user.nick}</h3>
              <p>UID {user.uid}</p>
            </div>
          </div>
          <span className={`user-state${user.is_study && !isExpired ? " user-state--running" : ""}`}>
            {isExpired ? "需要重新接入" : user.is_study ? "執行中" : "可啟動"}
          </span>
        </div>

        <div className="user-card__meta">
          <div>
            <span>登入時間</span>
            <strong>{this.formatTimestamp(user.login_time)}</strong>
          </div>
          <div>
            <span>接入時長</span>
            <strong>{this.getOnlineDays(user.login_time)} 天</strong>
          </div>
          <div>
            <span>今日得分</span>
            <strong>{score.todayScore}</strong>
          </div>
          <div>
            <span>當前總積分</span>
            <strong>{score.totalScore}</strong>
          </div>
        </div>

        <div className="user-card__progress">
          <div className="mini-progress">
            <div className="mini-progress__head">
              <strong>文章學習</strong>
              <span>{score.articleScore}</span>
            </div>
            <div className="mini-progress__track">
              <div className="mini-progress__value" style={{ width: score.articlePercentage }} />
            </div>
          </div>

          <div className="mini-progress">
            <div className="mini-progress__head">
              <strong>視頻學習</strong>
              <span>{score.videoScore}</span>
            </div>
            <div className="mini-progress__track">
              <div className="mini-progress__value mini-progress__value--accent" style={{ width: score.videoPercentage }} />
            </div>
          </div>

          <div className="mini-progress">
            <div className="mini-progress__head">
              <strong>每日答題</strong>
              <span>{score.dailyScore}</span>
            </div>
            <div className="mini-progress__track">
              <div className="mini-progress__value mini-progress__value--daily" style={{ width: score.dailyPercentage }} />
            </div>
          </div>
        </div>

        <div className="user-card__actions">
          <button
            className="ghost-button"
            onClick={() => this.toggleStudy(user.uid, user.is_study, isExpired)}
          >
            {user.is_study && !isExpired ? "停止學習" : "開始學習"}
          </button>
          <button className="ghost-button ghost-button--muted" onClick={() => this.openScore(user.token, user.nick)}>
            積分詳情
          </button>
          {this.state.level === "1" ? (
            <button className="ghost-button ghost-button--danger" onClick={() => this.deleteUserRecord(user.uid, user.nick)}>
              刪除用戶
            </button>
          ) : null}
        </div>
      </article>
    );
  };

  render() {
    const { users, expiredUsers, selectedScore, loadingScore, searchTerm, filterType, sortBy } = this.state;
    const summary = Users.SummaryPanel(users, expiredUsers);
    const visibleUsers = this.getVisibleUsers(users, false);
    const visibleExpiredUsers = this.getVisibleUsers(expiredUsers, true);

    return (
      <section className="page-section">
        <div className="section-heading">
          <div>
            <span className="section-kicker">Users</span>
            <h2>多帳號控制台</h2>
            <p>把掃碼接入、積分查看、啟動學習與失效處理都收在同一塊面板內完成。</p>
          </div>
        </div>

        <div className="summary-grid">
          <article className="summary-card">
            <span className="summary-card__label">TOTAL</span>
            <strong>{summary.totalUsers}</strong>
            <p>已記錄的帳戶數量。</p>
          </article>
          <article className="summary-card">
            <span className="summary-card__label">ACTIVE</span>
            <strong>{summary.activeUsers}</strong>
            <p>目前 Cookie 有效的帳戶。</p>
          </article>
          <article className="summary-card">
            <span className="summary-card__label">RUNNING</span>
            <strong>{summary.studyingUsers}</strong>
            <p>正在執行文章與音頻學習。</p>
          </article>
          <article className="summary-card">
            <span className="summary-card__label">EXPIRED</span>
            <strong>{summary.inactiveUsers}</strong>
            <p>需要重新掃碼接入的帳戶。</p>
          </article>
        </div>

        <div className="page-card user-panel">
          <div className="user-toolbar">
            <select value={filterType} onChange={(event) => this.setFilter(event.target.value)} className="toolbar-select">
              <option value="全部">全部狀態</option>
              <option value="學習中">學習中</option>
              <option value="待機中">待機中</option>
              <option value="已失效">已失效</option>
            </select>

            <select value={sortBy} onChange={(event) => this.setSort(event.target.value)} className="toolbar-select">
              <option value="login_desc">按最近登入</option>
              <option value="login_asc">按最早登入</option>
              <option value="nick_asc">按用戶名</option>
              <option value="today_desc">按今日得分</option>
              <option value="total_desc">按總積分</option>
            </select>

            <div className="toolbar-search">
              <input
                type="text"
                placeholder="搜尋用戶名或 UID"
                value={searchTerm}
                onChange={this.handleSearch}
                className="toolbar-input"
              />
            </div>
          </div>

          <div className="user-section-block">
            <div className="user-section-block__head">
              <h3>有效帳戶</h3>
              <span>{visibleUsers.length} 個</span>
            </div>
            {visibleUsers.length ? (
              <div className="user-card-grid">{visibleUsers.map((user) => this.renderUserCard(user, false))}</div>
            ) : (
              <div className="user-empty">目前篩選條件下沒有有效帳戶。</div>
            )}
          </div>

          <div className="user-section-block">
            <div className="user-section-block__head">
              <h3>失效帳戶</h3>
              <span>{visibleExpiredUsers.length} 個</span>
            </div>
            {visibleExpiredUsers.length ? (
              <div className="user-card-grid">{visibleExpiredUsers.map((user) => this.renderUserCard(user, true))}</div>
            ) : (
              <div className="user-empty">目前沒有失效帳戶。</div>
            )}
          </div>
        </div>

        {selectedScore ? (
          <div className="modal-shell" onClick={this.closeScore}>
            <div className="score-modal" onClick={(event) => event.stopPropagation()}>
              <button className="modal-close" onClick={this.closeScore}>
                ×
              </button>
              <span className="section-kicker">Score Detail</span>
              <h3>{selectedScore.nick}</h3>

              <div className="score-modal__headline">
                <article>
                  <span>當前總積分</span>
                  <strong>{selectedScore.totalScore}</strong>
                </article>
                <article>
                  <span>今日得分</span>
                  <strong>{selectedScore.todayScore}</strong>
                </article>
              </div>

              <div className="score-progress-list">
                <div className="score-progress-card">
                  <div className="score-progress-card__head">
                    <strong>文章學習</strong>
                    <span>{selectedScore.articleScore}</span>
                  </div>
                  <div className="score-progress-card__track">
                    <div className="score-progress-card__value" style={{ width: selectedScore.articlePercentage }} />
                  </div>
                  <small>{selectedScore.articlePercentage}</small>
                </div>

                <div className="score-progress-card">
                  <div className="score-progress-card__head">
                    <strong>視頻學習</strong>
                    <span>{selectedScore.videoScore}</span>
                  </div>
                  <div className="score-progress-card__track">
                    <div className="score-progress-card__value score-progress-card__value--accent" style={{ width: selectedScore.videoPercentage }} />
                  </div>
                  <small>{selectedScore.videoPercentage}</small>
                </div>

                <div className="score-progress-card">
                  <div className="score-progress-card__head">
                    <strong>每日答題</strong>
                    <span>{selectedScore.dailyScore}</span>
                  </div>
                  <div className="score-progress-card__track">
                    <div className="score-progress-card__value score-progress-card__value--daily" style={{ width: selectedScore.dailyPercentage }} />
                  </div>
                  <small>{selectedScore.dailyPercentage}</small>
                </div>
              </div>
            </div>
          </div>
        ) : null}

        {loadingScore ? <div className="user-loading-tip">正在載入積分詳情...</div> : null}
      </section>
    );
  }
}

export default Users;
