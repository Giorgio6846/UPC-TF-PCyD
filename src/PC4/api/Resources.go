package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	ds "pc4/DistributedSystem"
	"pc4/tools"
)

func ResourcesConsumption(w http.ResponseWriter, r *http.Request) {
	cpuPct, err := getCPUPercent()
	if err != nil {
		http.Error(w, "failed to read cpu", http.StatusInternalServerError)
		return
	}

	totalKB, availKB, usedKB, usedPct, err := getMemInfo()
	if err != nil {
		http.Error(w, "failed to read memory", http.StatusInternalServerError)
		return
	}

	sent, recv, err := getNetDev()
	if err != nil {
		http.Error(w, "failed to read network", http.StatusInternalServerError)
		return
	}

	workers := ds.GetWorkers()

	resp := tools.ResourcesResponse{
		CPUPercent:        cpuPct,
		MemoryTotalKB:     totalKB,
		MemoryAvailableKB: availKB,
		MemoryUsedKB:      usedKB,
		MemoryUsedPercent: usedPct,
		NetworkBytesSent:  sent,
		NetworkBytesRecv:  recv,
		Workers:           workers,
		TimestampMs:       time.Now().UnixMilli(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// getCPUPercent reads /proc/stat twice (short interval) and computes CPU usage percentage.
func getCPUPercent() (float64, error) {
	t1total, t1idle, err := readProcStat()
	if err != nil {
		return 0, err
	}
	time.Sleep(120 * time.Millisecond)
	t2total, t2idle, err := readProcStat()
	if err != nil {
		return 0, err
	}
	totalDelta := float64(t2total - t1total)
	idleDelta := float64(t2idle - t1idle)
	if totalDelta <= 0 {
		return 0, nil
	}
	usage := (1.0 - (idleDelta / totalDelta)) * 100.0
	return usage, nil
}

func readProcStat() (uint64, uint64, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			// fields[0] == "cpu"
			var total uint64 = 0
			var idle uint64 = 0
			for i := 1; i < len(fields); i++ {
				v, _ := strconv.ParseUint(fields[i], 10, 64)
				total += v
				if i == 4 { // idle is the 4th field after cpu (index 4)
					idle = v
				}
			}
			return total, idle, nil
		}
	}
	return 0, 0, scanner.Err()
}

// getMemInfo parses /proc/meminfo and returns totalKB, availableKB, usedKB, usedPercent
func getMemInfo() (uint64, uint64, uint64, float64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var totalKB uint64
	var availKB uint64
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		switch key {
		case "MemTotal":
			v, _ := strconv.ParseUint(parts[1], 10, 64)
			totalKB = v
		case "MemAvailable":
			v, _ := strconv.ParseUint(parts[1], 10, 64)
			availKB = v
		}
		if totalKB > 0 && availKB > 0 {
			break
		}
	}
	if totalKB == 0 {
		return 0, 0, 0, 0, nil
	}
	usedKB := totalKB - availKB
	usedPct := (float64(usedKB) / float64(totalKB)) * 100.0
	return totalKB, availKB, usedKB, usedPct, scanner.Err()
}

// getNetDev sums bytes sent/received across interfaces (skips loopback)
func getNetDev() (uint64, uint64, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	// skip first two header lines
	if !scanner.Scan() {
		return 0, 0, scanner.Err()
	}
	if !scanner.Scan() {
		return 0, 0, scanner.Err()
	}
	var totalRecv uint64
	var totalTrans uint64
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" || iface == "" {
			continue
		}
		data := strings.Fields(parts[1])
		if len(data) < 9 {
			continue
		}
		recv, _ := strconv.ParseUint(data[0], 10, 64)
		trans, _ := strconv.ParseUint(data[8], 10, 64)
		totalRecv += recv
		totalTrans += trans
	}
	return totalTrans, totalRecv, scanner.Err()
}
