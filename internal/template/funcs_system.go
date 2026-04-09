package template

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/0funct0ry/xwebs/internal/build"
)

var (
	statsCache struct {
		cpu     string
		mem     string
		disk    string
		updated time.Time
	}
	statsMu sync.Mutex
)

func (e *Engine) registerSystemFuncs() {
	if e.sandboxed {
		disabled := func(name string) func(...interface{}) (interface{}, error) {
			return func(args ...interface{}) (interface{}, error) {
				return nil, fmt.Errorf("function %s is disabled in sandbox mode", name)
			}
		}

		e.funcs["env"] = disabled("env")
		e.funcs["shell"] = disabled("shell")
		e.funcs["hostname"] = disabled("hostname")
		e.funcs["pid"] = disabled("pid")
		e.funcs["cwd"] = disabled("cwd")
		e.funcs["fileRead"] = disabled("fileRead")
		e.funcs["fileExists"] = disabled("fileExists")
		e.funcs["glob"] = disabled("glob")
		e.funcs["tempFile"] = disabled("tempFile")
		e.funcs["user"] = disabled("user")
		e.funcs["home"] = disabled("home")
		e.funcs["xwebsVersion"] = disabled("xwebsVersion")
		e.funcs["tty"] = disabled("tty")
		e.funcs["cpuUsage"] = disabled("cpuUsage")
		e.funcs["memUsage"] = disabled("memUsage")
		e.funcs["diskUsage"] = disabled("diskUsage")
		return
	}

	e.funcs["env"] = func(key string) string {
		return os.Getenv(key)
	}

	e.funcs["shell"] = func(command string) (string, error) {
		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return string(output), err
		}
		return strings.TrimSpace(string(output)), nil
	}

	e.funcs["hostname"] = func() string {
		h, _ := os.Hostname()
		return h
	}

	e.funcs["pid"] = func() int {
		return os.Getpid()
	}

	e.funcs["cwd"] = func() string {
		c, _ := os.Getwd()
		return c
	}

	e.funcs["fileRead"] = func(path string) (string, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	e.funcs["fileExists"] = func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	}

	e.funcs["glob"] = func(pattern string) ([]string, error) {
		return filepath.Glob(pattern)
	}

	e.funcs["tempFile"] = func(pattern string) (string, error) {
		f, err := os.CreateTemp("", pattern)
		if err != nil {
			return "", err
		}
		f.Close()
		return f.Name(), nil
	}

	e.funcs["user"] = func() string {
		u, err := user.Current()
		if err != nil {
			return os.Getenv("USER")
		}
		return u.Username
	}

	e.funcs["home"] = func() string {
		h, err := os.UserHomeDir()
		if err != nil {
			return os.Getenv("HOME")
		}
		return h
	}

	e.funcs["xwebsVersion"] = func() string {
		return build.Version
	}

	e.funcs["tty"] = func() string {
		t := os.Getenv("TTY")
		if t != "" {
			return t
		}
		out, err := exec.Command("tty").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
		return "unknown"
	}

	e.funcs["cpuUsage"] = func() string {
		updateStatsIfNeeded()
		statsMu.Lock()
		defer statsMu.Unlock()
		return statsCache.cpu
	}

	e.funcs["memUsage"] = func() string {
		updateStatsIfNeeded()
		statsMu.Lock()
		defer statsMu.Unlock()
		return statsCache.mem
	}

	e.funcs["diskUsage"] = func() string {
		updateStatsIfNeeded()
		statsMu.Lock()
		defer statsMu.Unlock()
		return statsCache.disk
	}
}

func updateStatsIfNeeded() {
	statsMu.Lock()
	if time.Since(statsCache.updated) < 2*time.Second {
		statsMu.Unlock()
		return
	}
	statsMu.Unlock()

	// Update stats in the background or synchronized if first time
	// For now, let's do it synchronized but with a lock to prevent multiple concurrent updates
	statsMu.Lock()
	defer statsMu.Unlock()
	
	// Double check after acquiring lock
	if time.Since(statsCache.updated) < 2*time.Second {
		return
	}

	// Gather stats
	statsCache.cpu = getCPUUsage()
	statsCache.mem = getMemUsage()
	statsCache.disk = getDiskUsage()
	statsCache.updated = time.Now()
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func getCPUUsage() string {
	if runtime.GOOS == "darwin" {
		// top -l 1 -n 0 is fast enough and gives CPU usage
		out, err := exec.Command("top", "-l", "1", "-n", "0").Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.Contains(line, "CPU usage:") {
					// Format: CPU usage: 10.45% user, 10.45% sys, 79.10% idle
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						usageParts := strings.Split(parts[1], ",")
						if len(usageParts) > 2 {
							idlePart := strings.TrimSpace(usageParts[2])
							if strings.HasSuffix(idlePart, " idle") {
								idleStr := strings.TrimSuffix(idlePart, " idle")
								idleStr = strings.TrimSuffix(idleStr, "%")
								idle, _ := strconv.ParseFloat(idleStr, 64)
								return fmt.Sprintf("%.1f%%", 100.0-idle)
							}
						}
					}
				}
			}
		}
	} else if runtime.GOOS == "linux" {
		// On Linux, we can read /proc/loadavg
		out, err := os.ReadFile("/proc/loadavg")
		if err == nil {
			fields := strings.Fields(string(out))
			if len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return "0.0%"
}

func getMemUsage() string {
	if runtime.GOOS == "darwin" {
		// Total
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			total, _ := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)

			// Used via vm_stat
			vout, verr := exec.Command("vm_stat").Output()
			if verr == nil {
				lines := strings.Split(string(vout), "\n")
				var active, wired, compressed uint64
				pageSize := uint64(4096) // Default
				for _, line := range lines {
					if strings.Contains(line, "page size of") {
						parts := strings.Fields(line)
						if len(parts) > 7 {
							pageSize, _ = strconv.ParseUint(parts[7], 10, 64)
						}
					}
					if strings.HasPrefix(line, "Pages active:") {
						fields := strings.Fields(line)
						active, _ = strconv.ParseUint(strings.TrimSuffix(fields[2], "."), 10, 64)
					}
					if strings.HasPrefix(line, "Pages wired down:") {
						fields := strings.Fields(line)
						wired, _ = strconv.ParseUint(strings.TrimSuffix(fields[3], "."), 10, 64)
					}
					if strings.HasPrefix(line, "Pages occupied by compressor:") {
						fields := strings.Fields(line)
						compressed, _ = strconv.ParseUint(strings.TrimSuffix(fields[4], "."), 10, 64)
					}
				}
				used := (active + wired + compressed) * pageSize
				pct := float64(used) / float64(total) * 100
				return fmt.Sprintf("%s (%.0f%%)", formatBytes(used), pct)
			}
			return formatBytes(total)
		}
	} else if runtime.GOOS == "linux" {
		// Read /proc/meminfo
		data, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			lines := strings.Split(string(data), "\n")
			var total, free uint64
			for _, line := range lines {
				if strings.HasPrefix(line, "MemTotal:") {
					fields := strings.Fields(line)
					total, _ = strconv.ParseUint(fields[1], 10, 64)
					total *= 1024 // KB to bytes
				}
				if strings.HasPrefix(line, "MemAvailable:") {
					fields := strings.Fields(line)
					free, _ = strconv.ParseUint(fields[1], 10, 64)
					free *= 1024
				}
			}
			if total > 0 {
				used := total - free
				pct := float64(used) / float64(total) * 100
				return fmt.Sprintf("%s (%.0f%%)", formatBytes(used), pct)
			}
		}
	}
	return "0.0GB"
}

func getDiskUsage() string {
	// Root disk usage info
	// We can use 'df -h /' and parse it as a simple cross-platform way
	out, err := exec.Command("df", "-h", "/").Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 5 {
				// fields[2] is used, fields[4] is percent on many systems
				return fmt.Sprintf("%s (%s)", fields[2], fields[4])
			}
		}
	}
	return "0%"
}
