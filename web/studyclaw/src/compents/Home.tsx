import React, { useEffect, useState } from "react";
import { NavLink, Route, Routes } from "react-router-dom";

import "../App.css";
import "./css/style.css";
import "./css/1.3.0/css/line-awesome.min.css";
import AddUser from "./pages/AddUser";
import Config from "./pages/Config";
import Help from "./pages/Help";
import Log from "./pages/Log";
import Other from "./pages/Other";
import Overview from "./pages/Overview";
import { getStudyMode } from "../utils/api";

type HomeProps = {
  navigate: (path: string) => void;
  location: {
    pathname: string;
  };
};

const navItems = [
  { to: "/home", label: "總覽", icon: "las la-chart-pie" },
  { to: "/home/user", label: "用戶管理", icon: "las la-users" },
  { to: "/home/other", label: "管理台", icon: "las la-sliders-h", adminOnly: true },
  { to: "/home/help", label: "說明", icon: "las la-book-open" },
];

const pageMeta = (pathname: string, isAdmin: boolean) => {
  if (pathname.startsWith("/home/user")) {
    return {
      eyebrow: "USER HUB",
      title: "用戶管理",
      description: "掃碼接入、查看積分與手動操作都集中在這裡。",
    };
  }
  if (pathname.startsWith("/home/other/config")) {
    return {
      eyebrow: "CONFIG",
      title: "配置編輯",
      description: "維持 studyclaw 的唯一模式與部署配置。",
    };
  }
  if (pathname.startsWith("/home/other/log")) {
    return {
      eyebrow: "LOG",
      title: "執行日誌",
      description: "追蹤文章與音頻任務的即時輸出。",
    };
  }
  if (pathname.startsWith("/home/other")) {
    return {
      eyebrow: "CONTROL",
      title: "管理台",
      description: isAdmin ? "重啟、配置與日誌入口。" : "這個區域僅管理員可用。",
    };
  }
  if (pathname.startsWith("/home/help")) {
    return {
      eyebrow: "MANUAL",
      title: "使用說明",
      description: "查看部署、登入、推送與定時任務說明。",
    };
  }
  return {
    eyebrow: "OVERVIEW",
    title: "運行總覽",
    description: "快速掌握帳戶狀態與今日文章音頻進度。",
  };
};

const modeConfig: Record<number, { label: string; title: string; desc: string }> = {
  1: { label: "模式 A", title: "文章 + 音頻", desc: "先讀文章，再補足音頻學習，不執行每日答題。" },
  2: { label: "模式 B", title: "文章 + 音頻 + 每日答題", desc: "先讀文章，再音頻學習，最後自動完成每日答題。" },
};

function Home(props: HomeProps) {
  const [level, setLevel] = useState(sessionStorage.getItem("level") || "2");
  const [studyMode, setStudyMode] = useState(1);
  const isAdmin = level === "1";
  const meta = pageMeta(props.location.pathname, isAdmin);

  useEffect(() => {
    setLevel(sessionStorage.getItem("level") || "2");
  }, [props.location.pathname]);

  useEffect(() => {
    getStudyMode()
      .then((resp: any) => {
        if (resp?.data?.model) {
          setStudyMode(resp.data.model);
        }
      })
      .catch(() => {});
  }, []);

  const logout = () => {
    window.localStorage.removeItem("studyclaw_token");
    sessionStorage.removeItem("level");
    props.navigate("/login");
  };

  return (
    <div className="dashboard-shell">
      <input type="checkbox" id="menu-toggle" />
      <div className="dashboard-overlay">
        <label htmlFor="menu-toggle" />
      </div>

      <aside className="dashboard-sidebar">
        <div className="sidebar-panel">
          <div className="brand-block">
            <div className="brand-mark">SC</div>
            <div>
              <p className="brand-kicker">studyclaw</p>
              <h2>學習控制台</h2>
              <span>{isAdmin ? "管理員視角" : "普通用戶視角"}</span>
            </div>
          </div>

          <div className="sidebar-mode-group">
            {[1, 2].map((mode) => {
              const cfg = modeConfig[mode];
              const isActive = studyMode === mode;
              return (
                <div key={mode} className={`sidebar-status-card${isActive ? " sidebar-status-card--active" : ""}`}>
                  <span className="sidebar-status-card__label">{cfg.label}{isActive ? " · 當前" : ""}</span>
                  <strong>{cfg.title}</strong>
                  <p>{cfg.desc}</p>
                </div>
              );
            })}
          </div>

          <nav className="sidebar-nav">
            {navItems
              .filter((item) => !item.adminOnly || isAdmin)
              .map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.to === "/home"}
                  className={({ isActive }) => `sidebar-link${isActive ? " active" : ""}`}
                >
                  <span className={item.icon} />
                  <span>{item.label}</span>
                </NavLink>
              ))}
          </nav>

          <div className="sidebar-footer-card">
            <span className="sidebar-status-card__label">部署提示</span>
            <strong>正式入口已統一</strong>
            <p>建議直接使用 `/studyclaw/`，舊的靜態入口會自動轉回新網址，避免再次白屏。</p>
          </div>
        </div>
      </aside>

      <div className="dashboard-main">
        <header className="dashboard-header">
          <div className="dashboard-header__lead">
            <label htmlFor="menu-toggle" className="menu-toggle-button">
              <span className="las la-bars" />
            </label>
            <div>
              <span className="page-eyebrow">{meta.eyebrow}</span>
              <h1>{meta.title}</h1>
              <p>{meta.description}</p>
            </div>
          </div>

          <div className="dashboard-header__actions">
            <div className="status-pill">
              <span className="las la-layer-group" />
              <span>{isAdmin ? "ADMIN" : "USER"}</span>
            </div>
            <button onClick={logout} className="header-logout">
              <span className="las la-sign-out-alt" />
              <span>退出</span>
            </button>
          </div>
        </header>

        <main className="dashboard-content">
          <Routes>
            <Route path="" element={<Overview navigate={props.navigate} location={props.location} />} />
            <Route path="/user" element={<AddUser level={level} navigate={props.navigate} location={props.location} />} />
            <Route path="/other" element={<Other navigate={props.navigate} location={props.location} />} />
            <Route path="/help" element={<Help navigate={props.navigate} location={props.location} />} />
            <Route path="/other/log" element={<Log navigate={props.navigate} location={props.location} />} />
            <Route path="/other/config" element={<Config navigate={props.navigate} location={props.location} />} />
          </Routes>
        </main>
      </div>
    </div>
  );
}

export default Home;
