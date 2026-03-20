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

        return (
            <div className="page-stack">
                <section className="page-section">
                    <div className="section-heading">
                        <div>
                            <span className="section-kicker">Overview</span>
                            <h2>今日運行總覽</h2>
                            <p>從帳戶可用性到執行中任務，先看全局，再決定是否手動介入。</p>
                        </div>
                    </div>

                    <div className="summary-grid">
                        <article className="summary-card">
                            <span className="summary-card__label">總帳戶數</span>
                            <strong>{summaryPanelData.totalUsers}</strong>
                            <p>已接入控制台的帳戶總數。</p>
                        </article>
                        <article className="summary-card">
                            <span className="summary-card__label">有效帳戶</span>
                            <strong>{summaryPanelData.activeUsers}</strong>
                            <p>登入狀態有效，可直接執行任務。</p>
                        </article>
                        <article className="summary-card">
                            <span className="summary-card__label">失效帳戶</span>
                            <strong>{summaryPanelData.inactiveUsers}</strong>
                            <p>需要重新授權或重新接入的帳戶。</p>
                        </article>
                        <article className="summary-card">
                            <span className="summary-card__label">執行中任務</span>
                            <strong>{summaryPanelData.studyingUsers}</strong>
                            <p>目前正在自動學習的帳戶數。</p>
                        </article>
                    </div>
                </section>

                <section className="insight-grid">
                    <article className="page-card page-card--accent">
                        <span className="section-kicker">Health Check</span>
                        <h3>控制台健康度</h3>
                        <div className="insight-metric">
                            <span>有效帳戶佔比</span>
                            <strong>{activeRate}%</strong>
                        </div>
                        <div className="insight-metric">
                            <span>執行中任務佔比</span>
                            <strong>{runningRate}%</strong>
                        </div>
                        <p>若有效帳戶比例偏低，優先到「用戶管理」重新掃碼接入；若執行中帳戶過多，可檢查日誌確認是否有阻塞。</p>
                    </article>

                    <article className="page-card">
                        <span className="section-kicker">Quick Actions</span>
                        <h3>下一步操作</h3>
                        <div className="quick-action-list">
                            <button className="ghost-button" onClick={() => this.props.navigate("/home/user")}>
                                打開用戶管理
                            </button>
                            <button className="ghost-button" onClick={() => this.props.navigate("/home/other/log")}>
                                查看實時日誌
                            </button>
                            <button className="ghost-button" onClick={() => this.props.navigate("/home/help")}>
                                打開使用說明
                            </button>
                        </div>
                    </article>
                </section>
            </div>
        );
    }
}

export default Overview;
