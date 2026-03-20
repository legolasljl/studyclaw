import React, { ChangeEvent, Component } from "react";
import { Dialog, Toast } from "antd-mobile";

import { getConfig, saveConfig } from "../../utils/api";

class Config extends Component<any, any> {
  constructor(props: any) {
    super(props);
    this.state = {
      config: "",
      loading: true,
      saving: false,
    };
  }

  componentDidMount() {
    this.loadConfig();
  }

  loadConfig = () => {
    this.setState({ loading: true });
    getConfig()
      .then((value) => {
        this.setState({
          config: value.data || "",
          loading: false,
        });
      })
      .catch(() => {
        this.setState({ loading: false });
        Dialog.show({
          content: "讀取配置失敗，請稍後重試。",
          closeOnMaskClick: true,
          closeOnAction: true,
        });
      });
  };

  onChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    this.setState({
      config: event.target.value,
    });
  };

  onSave = () => {
    this.setState({ saving: true });
    saveConfig(this.state.config)
      .then((resp) => {
        if (resp.code === 200) {
          Toast.show("儲存成功");
          return;
        }
        Dialog.show({
          content: "配置提交失敗：" + (resp.error || resp.message || "未知錯誤"),
          closeOnMaskClick: true,
          closeOnAction: true,
        });
      })
      .catch(() => {
        Dialog.show({
          content: "配置提交失敗，請檢查服務是否可用。",
          closeOnMaskClick: true,
          closeOnAction: true,
        });
      })
      .finally(() => {
        this.setState({ saving: false });
      });
  };

  render() {
    const { config, loading, saving } = this.state;

    return (
      <section className="page-section">
        <div className="section-heading section-heading--split">
          <div>
            <span className="section-kicker">Config Studio</span>
            <h2>配置檔編輯</h2>
            <p>直接查看並編輯目前生效的 YAML 配置。這裡優先保證穩定可用，不再依賴 Monaco 編輯器。</p>
          </div>
          <div className="section-actions">
            <button className="ghost-button" onClick={this.loadConfig} disabled={loading || saving}>
              重新載入
            </button>
            <button className="primary-button" onClick={this.onSave} disabled={loading || saving}>
              {saving ? "儲存中..." : "儲存配置"}
            </button>
          </div>
        </div>

        <div className="page-card editor-shell">
          <div className="config-editor__meta">
            <span>{loading ? "正在載入配置..." : "已載入目前配置"}</span>
            <span>{config.split("\n").length} 行</span>
          </div>

          <textarea
            className="config-editor"
            value={config}
            onChange={this.onChange}
            spellCheck={false}
            disabled={loading}
          />
        </div>
      </section>
    );
  }
}

export default Config;
