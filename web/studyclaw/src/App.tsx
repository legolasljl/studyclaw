import React, { useEffect, useState } from 'react';
import './App.css';
import { checkToken } from "./utils/api";
import Login from './compents/pages/Login';
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import Home from './compents/Home';

function App() {
    const navigate = useNavigate();
    const location = useLocation();
    const [checking, setChecking] = useState(true);

    useEffect(() => {
        let active = true;

        checkToken().then((t) => {
            if (!active) {
                return;
            }

            if (!t || !t.success) {
                sessionStorage.removeItem("level");
                if (location.pathname !== "/login") {
                    navigate("/login", { replace: true });
                }
            } else {
                sessionStorage.setItem("level", t.data === 1 ? "1" : "2");
                if (location.pathname === "/" || location.pathname === "/login") {
                    navigate("/home", { replace: true });
                }
            }
        }).catch(() => {
            if (!active) {
                return;
            }
            sessionStorage.removeItem("level");
            navigate("/login", { replace: true });
        }).finally(() => {
            if (active) {
                setChecking(false);
            }
        });

        return () => {
            active = false;
        };
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    if (checking) {
        return (
            <div className="boot-screen">
                <div className="boot-screen__panel">
                    <span className="boot-screen__eyebrow">STUDYCLAW</span>
                    <h1>正在校驗控制台會話</h1>
                    <p>檢查登入狀態並準備文章與音頻學習面板。</p>
                </div>
            </div>
        );
    }

    return (
        <Routes>
            <Route path="/" element={<Navigate to="/login" replace />} />
            <Route path="/login" element={<Login navigate={navigate} />} />
            <Route path="/home/*" element={<Home navigate={navigate} location={location} />} />
        </Routes>
    );
}

export default App;
