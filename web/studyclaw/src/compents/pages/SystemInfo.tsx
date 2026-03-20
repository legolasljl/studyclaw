import React, { useState, useEffect } from 'react';
import os from 'os';

interface SystemInfo {
  cpuUsage: number;
  memoryUsage: number;
}

const SystemMonitor: React.FC = () => {
  const [systemInfo, setSystemInfo] = useState<SystemInfo>({ cpuUsage: 0, memoryUsage: 0 });

  useEffect(() => {
    const fetchSystemInfo = () => {
      // 获取 CPU 信息
      const cpuInfo = os.cpus();
      const totalIdle = cpuInfo.reduce((acc, cpu) => acc + cpu.times.idle, 0);
      const totalUsage = cpuInfo.reduce((acc, cpu) => acc + cpu.times.user + cpu.times.nice + cpu.times.sys + cpu.times.irq, 0);

      // 计算 CPU 利用率
      const cpuUsage = (1 - totalIdle / totalUsage) * 100;

      // 获取内存信息
      const totalMemory = os.totalmem();
      const usedMemory = totalMemory - os.freemem();

      // 计算内存使用率
      const memoryUsage = (usedMemory / totalMemory) * 100;

      // 更新状态
      setSystemInfo({ cpuUsage, memoryUsage });
    };

    // 定时获取系统信息（每秒钟）
    const intervalId = setInterval(fetchSystemInfo, 1000);

    // 在组件卸载时清除定时器
    return () => clearInterval(intervalId);
  }, []);

  return (
    <div>
      <h2>System Monitor</h2>
      <p>CPU Usage: {systemInfo.cpuUsage.toFixed(2)}%</p>
      <p>Memory Usage: {systemInfo.memoryUsage.toFixed(2)}%</p>
    </div>
  );
};

export default SystemMonitor;
