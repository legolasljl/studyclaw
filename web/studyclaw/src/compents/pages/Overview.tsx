import React, { Component } from "react";
import Users from "./User";
import { getExpiredUsers, getUsers } from "../../utils/api";

class Overview extends Component<any, any> {
    constructor(props: any) {
        super(props);
        this.state = {
            users: [],
            expired_users: [],
        };
    }

    componentDidMount() {
        this.fetchUserData();
    }

    fetchUserData = () => {
        getUsers().then(users => {
            this.setState({
                users: users.data || [],
            });

            getExpiredUsers().then(expiredUsers => {
                this.setState({
                    expired_users: expiredUsers.data || [],
                });
            });
        });
    };

    render() {
        const { users, expired_users } = this.state;
        const summaryPanelData = Users.SummaryPanel(users, expired_users);
        const totalUsers = Math.max(summaryPanelData.totalUsers, 1);
        const activeRate = Math.round((summaryPanelData.activeUsers / totalUsers) * 100);
        const runningRate = Math.round((summaryPanelData.studyingUsers / totalUsers) * 100);
        const healthRate = summaryPanelData.totalUsers === 0
            ? 98
            : Math.max(36, Math.min(99, Math.round(activeRate * 0.72 + runningRate * 0.28)));
        const statCards = [
            {
                title: "Daily Overview",
                label: "Total Users",
                value: summaryPanelData.totalUsers,
                delta: `${summaryPanelData.activeUsers} ready`,
                icon: "las la-users",
            },
            {
                title: "Daily Overview",
                label: "Active Learners",
                value: summaryPanelData.activeUsers,
                delta: `${activeRate}% active`,
                icon: "las la-user-check",
            },
            {
                title: "Daily Overview",
                label: "Tasks Completed",
                value: summaryPanelData.studyingUsers,
                delta: `${runningRate}% running`,
                icon: "las la-tasks",
            },
        ];
        const activityItems = [
            {
                label: summaryPanelData.totalUsers === 0 ? "Console ready for onboarding" : `${summaryPanelData.totalUsers} users synced into console`,
                time: "just now",
                tone: "good",
            },
            {
                label: `${summaryPanelData.activeUsers} accounts available for immediate study`,
                time: "sync",
                tone: "good",
            },
            {
                label: `${summaryPanelData.inactiveUsers} accounts require re-authentication`,
                time: "watch",
                tone: summaryPanelData.inactiveUsers > 0 ? "warn" : "good",
            },
            {
                label: `${summaryPanelData.studyingUsers} jobs currently running`,
                time: "runtime",
                tone: summaryPanelData.studyingUsers > 0 ? "info" : "muted",
            },
        ];

        return (
            <div className="overview-board">
                <div className="overview-main">
                    <section className="overview-stats-grid">
                        {statCards.map((card) => (
                            <article className="overview-stat-card" key={card.label}>
                                <div className="overview-stat-card__head">
                                    <div>
                                        <span>{card.title}</span>
                                        <strong>{card.label}</strong>
                                    </div>
                                    <div className="overview-stat-card__icon">
                                        <span className={card.icon} />
                                    </div>
                                </div>
                                <div className="overview-stat-card__value">{card.value}</div>
                                <div className="overview-stat-card__footer">
                                    <small>{card.delta}</small>
                                    <span className="overview-sparkline" />
                                </div>
                            </article>
                        ))}
                    </section>

                    <article className="overview-health-card">
                        <div className="overview-health-card__header">
                            <div>
                                <span className="section-kicker">Console Health</span>
                                <h3>控制台健康度</h3>
                            </div>
                            <div className="overview-health-card__meta">
                                <span>Active {activeRate}%</span>
                                <span>Running {runningRate}%</span>
                            </div>
                        </div>
                        <div className="overview-health-card__score">{healthRate}%</div>
                        <div className="overview-health-card__track">
                            <div className="overview-health-card__value" style={{ width: `${healthRate}%` }} />
                        </div>
                        <p>{summaryPanelData.inactiveUsers > 0 ? "部分帳戶需要重新登入，建議優先處理失效帳戶。" : "System operating normally，帳戶與執行鏈路目前健康。"}</p>
                    </article>
                </div>

                <aside className="overview-side">
                    <article className="overview-side-card">
                        <span className="section-kicker">Quick Actions</span>
                        <h3>快捷操作</h3>
                        <div className="overview-action-list">
                            <button className="overview-action-pill" onClick={() => this.props.navigate("/home/user")}>
                                <span className="las la-user-plus" />
                                <span>新增用戶</span>
                            </button>
                            <button className="overview-action-pill" onClick={() => this.props.navigate("/home/other")}>
                                <span className="las la-sliders-h" />
                                <span>管理控制台</span>
                            </button>
                            <button className="overview-action-pill" onClick={() => this.props.navigate("/home/other/log")}>
                                <span className="las la-file-alt" />
                                <span>查看系統日誌</span>
                            </button>
                            <button className="overview-action-pill" onClick={() => this.props.navigate("/home/help")}>
                                <span className="las la-book-open" />
                                <span>部署說明</span>
                            </button>
                        </div>
                    </article>

                    <article className="overview-side-card">
                        <span className="section-kicker">Recent Activity</span>
                        <h3>最近活動</h3>
                        <div className="overview-activity-list">
                            {activityItems.map((item) => (
                                <div className="overview-activity-item" key={`${item.label}-${item.time}`}>
                                    <span className={`overview-activity-item__dot overview-activity-item__dot--${item.tone}`} />
                                    <span className="overview-activity-item__label">{item.label}</span>
                                    <small>{item.time}</small>
                                </div>
                            ))}
                        </div>
                    </article>
                </aside>
            </div>
        );
    }
}

export default Overview;
