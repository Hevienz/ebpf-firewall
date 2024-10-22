package utils

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// 获取系统资源使用情况
func GetSystemResourceUsage() (interface{}, error) {
	resourceUsage := struct {
		CPU     CPUInfo     `json:"cpu"`
		Memory  MemoryInfo  `json:"memory"`
		Disk    []DiskInfo  `json:"disk"`
		Host    HostInfo    `json:"host"`
		Network NetworkInfo `json:"network"`
	}{}

	var err error
	// CPU 信息
	resourceUsage.CPU, err = getCPUInfo()
	if err != nil {
		return nil, fmt.Errorf("获取CPU信息失败: %v", err)
	}
	// 内存信息
	resourceUsage.Memory, err = getMemoryInfo()
	if err != nil {
		return nil, fmt.Errorf("获取内存信息失败: %v", err)
	}
	// 磁盘信息
	resourceUsage.Disk, err = getDiskInfo()
	if err != nil {
		return nil, fmt.Errorf("获取磁盘信息失败: %v", err)
	}
	// 主机信息
	resourceUsage.Host, err = getHostInfo()
	if err != nil {
		return nil, fmt.Errorf("获取主机信息失败: %v", err)
	}
	// 网络信息
	resourceUsage.Network, err = getNetworkInfo()
	if err != nil {
		return nil, fmt.Errorf("获取网络信息失败: %v", err)
	}

	return resourceUsage, nil
}

type CPUInfo struct {
	Name          string    `json:"name"`
	PhysicalCores int       `json:"physical_cores"`
	LogicalCores  int       `json:"logical_cores"`
	UsagePerCore  []float64 `json:"usage_per_core"`
	TotalUsage    float64   `json:"total_usage"`
}

func getCPUInfo() (CPUInfo, error) {
	info := CPUInfo{}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return info, err
	}
	if len(cpuInfo) > 0 {
		info.Name = cpuInfo[0].ModelName
	}

	info.PhysicalCores, err = cpu.Counts(false)
	if err != nil {
		return info, err
	}
	info.LogicalCores, err = cpu.Counts(true)
	if err != nil {
		return info, err
	}

	percentages, err := cpu.Percent(time.Second, true)
	if err != nil {
		return info, err
	}
	info.UsagePerCore = percentages

	totalPercentage, err := cpu.Percent(time.Second, false)
	if err != nil {
		return info, err
	}
	if len(totalPercentage) > 0 {
		info.TotalUsage = totalPercentage[0]
	}
	return info, nil
}

type MemoryInfo struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	UsageRate float64 `json:"usage_rate"`
}

func getMemoryInfo() (MemoryInfo, error) {
	info := MemoryInfo{}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return info, err
	}

	info.Total = memInfo.Total
	info.Used = memInfo.Used
	info.Available = memInfo.Available
	info.UsageRate = memInfo.UsedPercent

	return info, nil
}

type DiskInfo struct {
	MountPoint string  `json:"mount_point"`
	Device     string  `json:"device"`
	FSType     string  `json:"fs_type"`
	Total      uint64  `json:"total"`
	Free       uint64  `json:"free"`
	Used       uint64  `json:"used"`
	UsageRate  float64 `json:"usage_rate"`
}

func getDiskInfo() ([]DiskInfo, error) {
	var diskInfos []DiskInfo

	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		diskInfos = append(diskInfos, DiskInfo{
			MountPoint: partition.Mountpoint,
			Device:     partition.Device,
			FSType:     partition.Fstype,
			Total:      usage.Total,
			Free:       usage.Free,
			Used:       usage.Used,
			UsageRate:  usage.UsedPercent,
		})
	}

	return diskInfos, nil
}

type HostInfo struct {
	Hostname     string `json:"hostname"`
	BootTime     uint64 `json:"boot_time"`
	ProcessCount int32  `json:"process_count"`
	OS           string `json:"os"`
	Platform     string `json:"platform"`
	Version      string `json:"version"`
}

func getHostInfo() (HostInfo, error) {
	info := HostInfo{}
	hostInfo, err := host.Info()
	if err != nil {
		return info, err
	}
	info.Hostname = hostInfo.Hostname
	info.BootTime = hostInfo.BootTime
	info.OS = hostInfo.OS
	info.Platform = hostInfo.Platform
	info.Version = hostInfo.PlatformVersion
	return info, nil
}

type NetworkInfoStat struct {
	Family uint32   `json:"family"`
	Type   uint32   `json:"type"`
	Laddr  net.Addr `json:"localaddr"`
	Raddr  net.Addr `json:"remoteaddr"`
	Status string   `json:"status"`
}

type NetworkInfo struct {
	Connections []NetworkInfoStat     `json:"connections"`
	Interfaces  net.InterfaceStatList `json:"interfaces"`
}

func getNetworkInfo() (NetworkInfo, error) {
	info := NetworkInfo{}
	info.Connections = []NetworkInfoStat{}
	interfaces, err := net.Interfaces()
	if err != nil {
		return info, err
	}
	info.Interfaces = interfaces

	connections, err := net.Connections("all")
	if err != nil {
		return info, err
	}
	for _, connection := range connections {
		info.Connections = append(info.Connections, NetworkInfoStat{
			Family: connection.Family,
			Type:   connection.Type,
			Laddr:  connection.Laddr,
			Raddr:  connection.Raddr,
			Status: connection.Status,
		})
	}

	return info, nil
}
