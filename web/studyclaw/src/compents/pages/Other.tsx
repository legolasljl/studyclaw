import React, { Component } from "react";
import { Dialog, Toast } from "antd-mobile";
import { restart } from "../../utils/api";

class Other extends Component<any, any> {
    constructor(props: any) {
        super(props);
        this.state = {
            level: "2"
        };
    }

    componentDidMount() {
        this.setState({
            level: sessionStorage.getItem("level")
        });
    }

    onRestart = () => {
        Dialog.confirm({ confirmText: "確認", cancelText: "取消", content: "確認要重新啟動服務嗎？" }).then((result) => {
            if (result) {
                restart().then(() => {
                    Toast.show("重新啟動完成");
                });
            }
        });
    };

    render() {
        const isAdmin = this.state.level === "1";

        return (
            <section className="page-section" style={{ display: isAdmin ? "block" : "none" }}>
                <div className="section-heading">
                    <div>
                        <span className="section-kicker">Admin Tools</span>
                        <h2>管理工作台</h2>
                        <p>將高風險操作集中到同一頁，避免在用戶列表中分散處理。</p>
                    </div>
                </div>

                <div className="action-grid">
                    <button className="action-card" onClick={() => this.props.navigate('/home/other/log')}>
                        <span className="las la-stream" />
                        <strong>查看日誌</strong>
                        <p>持續觀察後端輸出與任務執行狀態。</p>
                    </button>

                    <button className="action-card" onClick={() => this.props.navigate('/home/other/config')}>
                        <span className="las la-sliders-h" />
                        <strong>配置管理</strong>
                        <p>直接檢視與保存 YAML 配置內容。</p>
                    </button>

                    <button className="action-card action-card--danger" onClick={this.onRestart}>
                        <span className="las la-redo-alt" />
                        <strong>重啟程序</strong>
                        <p>對後端服務執行一次明確的重啟動作。</p>
                    </button>
                </div>
            </section>
        );
    }
}

export default Other;
