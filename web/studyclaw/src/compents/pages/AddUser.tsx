import React, { Component } from "react";
// import { Toast } from "antd-mobile";
import { checkQrCode, getLink, getToken } from "../../utils/api";
import QRCode from "qrcode.react";
import Users from "./User";

class AddUser extends Component<any, any> {
  private usersRef: React.RefObject<any>;
  constructor(props: any) {
    super(props);
    this.state = {
      showPopup: false,
      img: "你還未取得登入連結",
      link: "",
      timer: null, // 添加 timer 属性
      check: null, // 添加 check 属性
      showExpiration: false, // 添加 showExpiration 属性
      token: {success:false},
    };
    this.usersRef = React.createRef();
  }
  
  componentWillUnmount() {
    if (this.state.check !== undefined) {
      clearInterval(this.state.check);
    }
    clearTimeout(this.state.timer);
  }
  isMobile = () => {
    return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
      window.navigator.userAgent
    );
  };

  isWechat = () => {
    return /MicroMessenger/i.test(window.navigator.userAgent);

  };

  click = async () => {
    this.setState({
      showPopup: true,
      img: "你還未取得登入連結",
      link: "正在載入，請稍候...",
      showExpiration: false,
      token: { success: false },
    });

    let data;
    try {
      data = await getLink();
    } catch {
      this.setState({ link: "", showExpiration: true });
      return;
    }
    this.setState({
      img: data.url,
      link: data.code,
    });
    let timer = setTimeout(() => {
      this.setState({ showExpiration: true });
    }, 5 * 60 * 1000);
    this.setState({ timer });

    let check = setInterval(async () => {
      try {
        let resp = await checkQrCode(data.code);
        if (resp && resp.success) {
          clearInterval(check);
          let token = await getToken(resp.data.split("=")[1], data.sign);
          if (token && token.success) {
            this.setState({
              link: "",
              token: token,
            });
            this.usersRef.current.fetchUserData();
          }
        }
      } catch {
        // ignore polling errors
      }
    }, 1000);

    this.setState({
      check: check,
    });

    setTimeout(() => {
      clearInterval(check);
    }, 1000 * 300);

    /* let element = document.createElement("a");
    element.href =
      "dtxuexi://appclient/page/study_feeds?url=" + escape(data.url);
  element.click();*/
  };

  redirectToApp = async () => {
    console.log(this.isMobile());
    if (this.isMobile()) {
      let element = document.createElement("a");
      element.href =
        "dtxuexi://appclient/page/study_feeds?url=" + escape(this.state.img);
      element.click();
    }
  };

  hidePopup = () => {
    clearInterval(this.state.check); // 清除计时器
    clearTimeout(this.state.timer);
    this.setState({ showPopup: false ,
    check: null,
    timer: null, // 重置计时器引用
    showExpiration: false, // 重置 showExpiration 状态
  });
  };
  callFetchUserData = () => {
    if (this.usersRef.current) {
      this.usersRef.current.fetchUserData();
    }
  };
  render() {
    return (
      <>
        <section className="page-section">
          <div className="section-heading">
            <div>
              <span className="section-kicker">User Onboarding</span>
              <h2>掃碼接入學習帳戶</h2>
              <p>生成二維碼後，用學習強國 App 授權即可把帳戶加入控制台。</p>
            </div>
          </div>

          <div className="onboarding-grid">
            <button className="onboarding-card onboarding-card--primary" onClick={this.click}>
              <div className="onboarding-card__icon">
                <span className="las la-user-plus"></span>
              </div>
              <div>
                <strong>添加用戶</strong>
                <p>產生授權二維碼，接入新的學習帳戶。</p>
              </div>
            </button>

            <article className="onboarding-card">
              <strong>使用提示</strong>
              <p>電腦端建議直接掃碼；手機瀏覽器可嘗試跳轉到學習強國 App；微信內不支援直接拉起 App。</p>
            </article>
          </div>

          {this.state.showPopup && (
            <div id="loginpopup-container" className="xshow">
              <div id="loginpopup" className="xshow">
                <span className="xclose" onClick={this.hidePopup}>
                  &times;
                </span>
                <div className="xqrcode">
                {this.state.link === "正在載入，請稍候..." ? (
                      <svg style = {{
                        position: "absolute",
                        left: "50%",
                        top: "50%",
                        transform: "translate(-50%, -50%) matrix(1, 0, 0, 1, 0, 0)"
                      }} preserveAspectRatio="xMidYMid meet" viewBox="0 0 187.3 93.7" height="300px" width="400px">
                    <path d="M93.9,46.4c9.3,9.5,13.8,17.9,23.5,17.9s17.5-7.8,17.5-17.5s-7.8-17.6-17.5-17.5c-9.7,0.1-13.3,7.2-22.1,17.1 				c-8.9,8.8-15.7,17.9-25.4,17.9s-17.5-7.8-17.5-17.5s7.8-17.5,17.5-17.5S86.2,38.6,93.9,46.4z" stroke-miterlimit="10" stroke-linejoin="round" stroke-linecap="round" stroke-width="4" fill="none" id="outline" stroke="#4E4FEB"></path>
                    <path d="M93.9,46.4c9.3,9.5,13.8,17.9,23.5,17.9s17.5-7.8,17.5-17.5s-7.8-17.6-17.5-17.5c-9.7,0.1-13.3,7.2-22.1,17.1 				c-8.9,8.8-15.7,17.9-25.4,17.9s-17.5-7.8-17.5-17.5s7.8-17.5,17.5-17.5S86.2,38.6,93.9,46.4z" stroke-miterlimit="10" stroke-linejoin="round" stroke-linecap="round" stroke-width="4" stroke="#4E4FEB" fill="none" opacity="0.05" id="outline-bg"></path>
                  </svg>
                    ) : (<QRCode
                    id="qrCode"
                    value={this.state.img}
                    size={250}
                    fgColor="#000000"
                    style={{
                      margin: "auto",
                      display: this.state.img === "你還未取得登入連結"
                        ? "none"
                        : "block",
                    }} />)}
                </div>
                {(this.state.link === "正在載入，請稍候...") && (<div className="expiration-text">
                      正在載入，請稍候...
                  </div>)}
                {!this.isMobile() && !(this.state.link === "正在載入，請稍候...") && (
                  <div className="nomobil">
                    <span>
                      注意：偵測到目前為桌面瀏覽器，此模式僅支援掃碼登入。
                    </span>
                  </div>
                )}
                {this.isMobile() && !this.isWechat() &&!(this.state.link === "正在載入，請稍候...") && (
                  <button onClick={this.redirectToApp} className="myButton1">
                    點擊開啟手機 App
                  </button>
                )}
                {this.isWechat() && this.isMobile() && !(this.state.link === "正在載入，請稍候...") && (
                   <div className="nomobil">
                      <span>注意：偵測到目前為微信環境，此模式僅支援掃碼登入。</span>
                    </div>
                )}
                {this.state.showExpiration && (
                  <div className="xxoverlay">
                    <div className="refresh-button" onClick={this.click}>
                      <svg viewBox="64 64 896 896" data-icon="sync" width="1em" height="1em" fill="currentColor" aria-hidden="true" focusable="false">
                        <path d="M168 504.2c1-43.7 10-86.1 26.9-126 17.3-41 42.1-77.7 73.7-109.4S337 212.3 378 195c42.4-17.9 87.4-27 133.9-27s91.5 9.1 133.8 27A341.5 341.5 0 0 1 755 268.8c9.9 9.9 19.2 20.4 27.8 31.4l-60.2 47a8 8 0 0 0 3 14.1l175.7 43c5 1.2 9.9-2.6 9.9-7.7l.8-180.9c0-6.7-7.7-10.5-12.9-6.3l-56.4 44.1C765.8 155.1 646.2 92 511.8 92 282.7 92 96.3 275.6 92 503.8a8 8 0 0 0 8 8.2h60c4.4 0 7.9-3.5 8-7.8zm756 7.8h-60c-4.4 0-7.9 3.5-8 7.8-1 43.7-10 86.1-26.9 126-17.3 41-42.1 77.8-73.7 109.4A342.45 342.45 0 0 1 512.1 856a342.24 342.24 0 0 1-243.2-100.8c-9.9-9.9-19.2-20.4-27.8-31.4l60.2-47a8 8 0 0 0-3-14.1l-175.7-43c-5-1.2-9.9 2.6-9.9 7.7l-.7 181c0 6.7 7.7 10.5 12.9 6.3l56.4-44.1C258.2 868.9 377.8 932 512.2 932c229.2 0 415.5-183.7 419.8-411.8a8 8 0 0 0-8-8.2z">
                        </path>
                      </svg>
                    </div>
                    <div className="expiration-text">二維碼已過期，點擊重新整理。</div>
                  </div>
                )}
                {this.state.token.success && (
                  <div className="xxoverlay">
                    <div className="refresh-button" onClick={this.click}>
                      <svg viewBox="64 64 896 896" data-icon="sync" width="1em" height="1em" fill="currentColor" aria-hidden="true" focusable="false">
                        <path d="M168 504.2c1-43.7 10-86.1 26.9-126 17.3-41 42.1-77.7 73.7-109.4S337 212.3 378 195c42.4-17.9 87.4-27 133.9-27s91.5 9.1 133.8 27A341.5 341.5 0 0 1 755 268.8c9.9 9.9 19.2 20.4 27.8 31.4l-60.2 47a8 8 0 0 0 3 14.1l175.7 43c5 1.2 9.9-2.6 9.9-7.7l.8-180.9c0-6.7-7.7-10.5-12.9-6.3l-56.4 44.1C765.8 155.1 646.2 92 511.8 92 282.7 92 96.3 275.6 92 503.8a8 8 0 0 0 8 8.2h60c4.4 0 7.9-3.5 8-7.8zm756 7.8h-60c-4.4 0-7.9 3.5-8 7.8-1 43.7-10 86.1-26.9 126-17.3 41-42.1 77.8-73.7 109.4A342.45 342.45 0 0 1 512.1 856a342.24 342.24 0 0 1-243.2-100.8c-9.9-9.9-19.2-20.4-27.8-31.4l60.2-47a8 8 0 0 0-3-14.1l-175.7-43c-5-1.2-9.9 2.6-9.9 7.7l-.7 181c0 6.7 7.7 10.5 12.9 6.3l56.4-44.1C258.2 868.9 377.8 932 512.2 932c229.2 0 415.5-183.7 419.8-411.8a8 8 0 0 0-8-8.2z">
                        </path>
                      </svg>
                    </div>
                    <div className="expiration-text">已登入；若要繼續新增，請點擊重新整理。</div>
                  </div>
                )}
              </div>
            </div>
          )}
        </section>
        <Users ref={this.usersRef} />
      </>
    );
  }
}

export default AddUser;
