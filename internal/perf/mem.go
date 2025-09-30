package perf

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

func ReadVMPeak() int {
	fo, err := os.Open("/proc/self/status")
	if err != nil {
		slog.Debug("Failed to read /proc/self/status.", "err", err)
		return 0
	}
	defer fo.Close() //nolint:errcheck

	scanner := bufio.NewScanner(fo)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "VmPeak:") {
			continue
		}

		fields := strings.Fields(line)
		value, err := strconv.Atoi(fields[1])
		if err != nil {
			slog.Debug("Failed to parse VmPeak.", "err", err)
			return 0
		}
		return value
	}

	if err := scanner.Err(); err != nil {
		slog.Debug("Failed to read from file.", "err", err)
	}

	return 0
}

func FormatBytes(value int) string {
	const divisor = 1024.
	const step = 512.
	units := []string{"B", "KiB", "MiB", "GiB"}

	unitIndex := 0
	var f float64
	for f = float64(value); f > step; f /= divisor {
		unitIndex++
	}
	return strings.Replace(fmt.Sprintf("%.1f%s", f, units[unitIndex]), ".0", "", 1)
}
