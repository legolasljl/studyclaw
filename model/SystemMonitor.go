package model

import (
	"fmt"
	"log"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type CPUUsage float64

func (c CPUUsage) String() string {
	return fmt.Sprintf("%.2f%%", c)
}

type MemoryUsage float64

func (m MemoryUsage) String() string {
	return fmt.Sprintf("%.2f%%", m)
}

type NetworkSpeed float64

func (n NetworkSpeed) String() string {
	return fmt.Sprintf("%.2f MB/s", n)
}

type DiskUsage float64

func (d DiskUsage) String() string {
	return fmt.Sprintf("%.2f%%", d)
}

type SystemMonitorInfo struct {
	CPUUsage    CPUUsage      `json:"cpu_usage"`
	MemoryUsage MemoryUsage   `json:"memory_usage"`
	NetworkSpeed NetworkSpeed `json:"network_speed"`
	DiskUsage   DiskUsage     `json:"disk_usage"`
}

// RunSystemMonitor 定期更新系统监控信息
func RunSystemMonitor(info *SystemMonitorInfo) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 获取 CPU 使用率
		percent, err := cpu.Percent(0, false)
		if err != nil {
    	log.Println("Failed to get CPU usage:", err)
		}
		if len(percent) > 0 {
    	info.CPUUsage = CPUUsage(percent[0])
		}

		// 获取内存使用率
		memInfo, err := mem.VirtualMemory()
		if err != nil {
			log.Println("Failed to get memory usage:", err)
		}
		info.MemoryUsage = MemoryUsage(memInfo.UsedPercent)

		// 获取硬盘占用率
		diskUsage, err := disk.Usage("/")
		if err != nil {
			log.Println("Failed to get disk usage:", err)
		}
		info.DiskUsage = DiskUsage(diskUsage.UsedPercent)

		// 获取网络速度
		netInfo, err := net.IOCounters(true)
		if err != nil {
			log.Println("Failed to get network speed:", err)
		} else if len(netInfo) > 0 {
			var networkSpeedTotal float64
			for _, iface := range netInfo {
				networkSpeedTotal += float64(iface.BytesSent+iface.BytesRecv) / 1024 / 1024
			}
			info.NetworkSpeed = NetworkSpeed(networkSpeedTotal)
		}
	}
}