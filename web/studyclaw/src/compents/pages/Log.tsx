import React, { Component } from "react";
import { getLog } from "../../utils/api";
import { TextArea } from "antd-mobile";

class Log extends Component<any, any> {
    timer: any;

    constructor(props: any) {
        super(props);
        this.state = {
            data: ""
        };
    }

    reverse = (str: string): string => {
        return str.split("\n").reverse().join("\n").trim();
    };

    componentDidMount() {
        getLog().then(data => {
            this.setState({
                data: this.reverse(data)
            });
        });
        this.timer = setInterval(() => {
            getLog().then((data: string) => {
                this.setState({
                    data: this.reverse(data)
                });
            });
        }, 30000);
    }

    componentWillUnmount() {
        clearInterval(this.timer);
    }

    render() {
        return (
            <section className="page-section">
                <div className="section-heading section-heading--split">
                    <div>
                        <span className="section-kicker">Runtime Trace</span>
                        <h2>實時日誌</h2>
                        <p>日誌每 30 秒刷新一次，適合追蹤登入、文章、視頻與答題任務。</p>
                    </div>
                    <button className="ghost-button" onClick={() => window.history.back()}>
                        返回
                    </button>
                </div>

                <div className="log-shell">
                    <TextArea autoSize disabled={true} value={this.state.data} />
                </div>
            </section>
        );
    }
}

export default Log;
