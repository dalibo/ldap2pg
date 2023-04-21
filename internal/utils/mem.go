package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lmittmann/tint"
	"golang.org/x/exp/slog"
)

func ReadVMPeak() int {
	fo, err := os.Open("/proc/self/status")
	if err != nil {
		slog.Debug("Failed to read /proc/self/status.", tint.Err(err))
		return 0
	}
	defer fo.Close()

	scanner := bufio.NewScanner(fo)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "VmPeak:") {
			continue
		}

		fields := strings.Fields(line)
		value, err := strconv.Atoi(fields[1])
		if err != nil {
			slog.Debug("Failed to parse VmPeak.", tint.Err(err))
			return 0
		}
		return value
	}

	if err := scanner.Err(); err != nil {
		slog.Debug("Failed to read from file.", tint.Err(err))
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
