package orchestration

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// MemoryInfo contains system memory statistics
type MemoryInfo struct {
	TotalRAM       uint64 // Total physical RAM in bytes
	FreeRAM        uint64 // Free RAM in bytes
	AvailableRAM   uint64 // Available RAM (includes reclaimable)
	SwapUsed       uint64 // Swap memory in use
	MemoryPressure string // "low", "medium", "high", "critical"
}

// GetMemoryInfo retrieves current system memory statistics
func GetMemoryInfo() (*MemoryInfo, error) {
	info := &MemoryInfo{}

	// Get total physical memory using sysctl
	totalRAM, err := getSysctlUint64("hw.memsize")
	if err != nil {
		return nil, fmt.Errorf("failed to get total memory: %w", err)
	}
	info.TotalRAM = totalRAM

	// Get VM statistics using vm_stat command
	vmStats, err := getVMStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get VM statistics: %w", err)
	}

	pageSize := uint64(syscall.Getpagesize())

	// Calculate memory values from vm_stat output
	info.FreeRAM = vmStats["free"] * pageSize
	inactive := vmStats["inactive"] * pageSize
	info.AvailableRAM = info.FreeRAM + inactive

	// Get swap usage (best effort)
	swapUsed, _ := getSwapUsage()
	info.SwapUsed = swapUsed

	// Determine memory pressure level
	availableGB := float64(info.AvailableRAM) / (1024 * 1024 * 1024)
	swapGB := float64(info.SwapUsed) / (1024 * 1024 * 1024)

	// Pressure calculation considers both available RAM and swap usage
	if availableGB < 5 || swapGB > 20 {
		info.MemoryPressure = "critical"
	} else if availableGB < 10 || swapGB > 10 {
		info.MemoryPressure = "high"
	} else if availableGB < 15 || swapGB > 5 {
		info.MemoryPressure = "medium"
	} else {
		info.MemoryPressure = "low"
	}

	return info, nil
}

// getSysctlUint64 retrieves a uint64 value from sysctl
func getSysctlUint64(name string) (uint64, error) {
	out, err := exec.Command("sysctl", "-n", name).Output()
	if err != nil {
		return 0, err
	}
	val, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// getVMStats parses output from vm_stat command
func getVMStats() (map[string]uint64, error) {
	out, err := exec.Command("vm_stat").Output()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]uint64)
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		// Parse lines like "Pages free:                               12345."
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) != 2 {
				continue
			}

			key := strings.ToLower(strings.TrimSpace(parts[0]))
			key = strings.TrimPrefix(key, "pages ")
			valueStr := strings.TrimSpace(strings.TrimSuffix(parts[1], "."))

			value, err := strconv.ParseUint(valueStr, 10, 64)
			if err != nil {
				continue
			}

			stats[key] = value
		}
	}

	return stats, nil
}

// getSwapUsage retrieves swap memory usage (macOS specific)
func getSwapUsage() (uint64, error) {
	out, err := exec.Command("sysctl", "-n", "vm.swapusage").Output()
	if err != nil {
		// If we can't get swap usage, return 0 (not critical)
		return 0, nil
	}

	// Parse output like "total = 1024.00M  used = 512.00M  free = 512.00M"
	line := string(out)
	if strings.Contains(line, "used = ") {
		parts := strings.Split(line, "used = ")
		if len(parts) > 1 {
			usedStr := strings.Fields(parts[1])[0]
			// Convert to bytes
			value, unit := parseMemorySize(usedStr)
			return value * unit, nil
		}
	}

	return 0, nil
}

// parseMemorySize parses a size string like "512.00M" into value and unit multiplier
func parseMemorySize(sizeStr string) (uint64, uint64) {
	sizeStr = strings.TrimSpace(sizeStr)

	// Extract numeric part
	var numStr string
	var unit string
	for i, c := range sizeStr {
		if c >= '0' && c <= '9' || c == '.' {
			numStr += string(c)
		} else {
			unit = sizeStr[i:]
			break
		}
	}

	value, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, 1
	}

	// Determine multiplier based on unit
	var multiplier uint64 = 1
	unit = strings.ToUpper(strings.TrimSpace(unit))
	switch unit {
	case "K":
		multiplier = 1024
	case "M":
		multiplier = 1024 * 1024
	case "G":
		multiplier = 1024 * 1024 * 1024
	}

	return uint64(value), multiplier
}

// ModelRequirements defines memory requirements for different model sizes
type ModelRequirements struct {
	Name          string
	MinMemoryGB   float64
	RecommendedGB float64
	Description   string
}

var (
	// Model tiers based on memory requirements
	ModelTiers = []ModelRequirements{
		{
			Name:          "small",
			MinMemoryGB:   2,
			RecommendedGB: 4,
			Description:   "3B parameter models (e.g., llama3.2:3b) - Fast, minimal memory",
		},
		{
			Name:          "medium",
			MinMemoryGB:   9,
			RecommendedGB: 12,
			Description:   "14B parameter models (e.g., qwen2.5:14b) - Balanced quality/memory",
		},
		{
			Name:          "large",
			MinMemoryGB:   19,
			RecommendedGB: 25,
			Description:   "32B parameter models (e.g., qwen2.5-coder:32b) - Highest quality",
		},
	}
)

// RecommendModelSize returns the recommended model size based on available memory
// taskComplexity can be "simple", "medium", or "complex"
func RecommendModelSize(taskComplexity string) (string, error) {
	memInfo, err := GetMemoryInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get memory info: %w", err)
	}

	availableGB := float64(memInfo.AvailableRAM) / (1024 * 1024 * 1024)

	// Adjust based on task complexity
	// Complex tasks need more buffer for prompt processing
	var requiredBuffer float64
	switch taskComplexity {
	case "simple":
		requiredBuffer = 2.0 // Less overhead for simple tasks
	case "complex":
		requiredBuffer = 5.0 // More overhead for complex prompts
	default: // "medium"
		requiredBuffer = 3.0
	}

	// Select largest model that fits in available memory with buffer
	for i := len(ModelTiers) - 1; i >= 0; i-- {
		tier := ModelTiers[i]
		if availableGB >= (tier.RecommendedGB + requiredBuffer) {
			return tier.Name, nil
		}
	}

	// If even small model doesn't fit, still return it but with warning
	// The system will try anyway, might work with swap
	return "small", fmt.Errorf("low memory warning: only %.2f GB available", availableGB)
}

// FormatMemoryInfo returns a human-readable string of memory information
func FormatMemoryInfo(info *MemoryInfo) string {
	totalGB := float64(info.TotalRAM) / (1024 * 1024 * 1024)
	freeGB := float64(info.FreeRAM) / (1024 * 1024 * 1024)
	availableGB := float64(info.AvailableRAM) / (1024 * 1024 * 1024)
	swapGB := float64(info.SwapUsed) / (1024 * 1024 * 1024)

	return fmt.Sprintf(
		"Memory: %.1f GB total, %.1f GB free, %.1f GB available, %.1f GB swap used (Pressure: %s)",
		totalGB, freeGB, availableGB, swapGB, info.MemoryPressure,
	)
}
