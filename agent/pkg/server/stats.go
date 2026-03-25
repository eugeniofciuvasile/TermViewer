package server

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type SystemStats struct {
	CPUUsage      float64 `json:"cpu_usage"`
	RAMUsedGB     float64 `json:"ram_used_gb"`
	RAMTotalGB    float64 `json:"ram_total_gb"`
	DiskPercent   float64 `json:"disk_percent"`
	UptimeSeconds uint64  `json:"uptime_seconds"`
}

var startTime = time.Now()
var lastCPUCombined uint64
var lastCPUIdle uint64

func getSystemStats() SystemStats {
	stats := SystemStats{
		UptimeSeconds: uint64(time.Since(startTime).Seconds()),
	}

	if runtime.GOOS == "linux" {
		fillLinuxMemoryStats(&stats)
		fillLinuxCPUStats(&stats)
	} else {
		// Fallback for non-linux (macOS/Windows)
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		stats.RAMUsedGB = float64(m.Alloc) / 1024 / 1024 / 1024
		stats.RAMTotalGB = float64(m.Sys) / 1024 / 1024 / 1024
	}

	// Disk usage (Unix-friendly)
	var stat syscall.Statfs_t
	wd, err := os.Getwd()
	if err == nil {
		if err := syscall.Statfs(wd, &stat); err == nil {
			total := stat.Blocks * uint64(stat.Bsize)
			free := stat.Bfree * uint64(stat.Bsize)
			used := total - free
			if total > 0 {
				stats.DiskPercent = float64(used) / float64(total) * 100
			}
		}
	}

	return stats
}

func fillLinuxMemoryStats(stats *SystemStats) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer file.Close()

	var memTotal, memAvailable uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d kB", &memTotal)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %d kB", &memAvailable)
		}
	}

	if memTotal > 0 {
		stats.RAMTotalGB = float64(memTotal) / 1024 / 1024
		stats.RAMUsedGB = float64(memTotal-memAvailable) / 1024 / 1024
	}
}

func fillLinuxCPUStats(stats *SystemStats) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "cpu" {
			return
		}

		var total uint64
		for i := 1; i < len(fields); i++ {
			val, _ := strconv.ParseUint(fields[i], 10, 64)
			total += val
		}
		idle, _ := strconv.ParseUint(fields[4], 10, 64)

		if lastCPUCombined > 0 {
			diffTotal := total - lastCPUCombined
			diffIdle := idle - lastCPUIdle
			if diffTotal > 0 {
				stats.CPUUsage = float64(diffTotal-diffIdle) / float64(diffTotal) * 100
			}
		}

		lastCPUCombined = total
		lastCPUIdle = idle
	}
}
