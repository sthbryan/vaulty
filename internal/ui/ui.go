package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
)

var Logger *log.Logger

func init() {
	Logger = log.NewWithOptions(nil, log.Options{
		ReportCaller:    false,
		ReportTimestamp: false,
		Level:           log.InfoLevel,
	})
}

const logo = `
██╗   ██╗ █████╗ ██╗   ██╗██╗  ████████╗██╗   ██╗
██║   ██║██╔══██╗██║   ██║██║  ╚══██╔══╝╚██╗ ██╔╝
██║   ██║███████║██║   ██║██║     ██║    ╚████╔╝ 
╚██╗ ██╔╝██╔══██║██║   ██║██║     ██║     ╚██╔╝  
 ╚████╔╝ ██║  ██║╚██████╔╝███████╗██║      ██║   
  ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝      ╚═╝   
`

func PrintLogo() {
	fmt.Println(TitleStyle.Render(logo))
}

func PrintSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(SuccessStyle.Render("✅ " + msg))
}

func PrintError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(ErrorStyle.Render("❌ " + msg))
}

func PrintWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(WarningStyle.Render("⚠️  " + msg))
}

func PrintInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(InfoStyle.Render("📦 " + msg))
}

func PrintLock(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(TitleStyle.Render("🔒 " + msg))
}

func PrintUnlock(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(SuccessStyle.Render("🔓 " + msg))
}

func PrintCloud(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(InfoStyle.Render("☁️  " + msg))
}

func PrintSave(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(SuccessStyle.Render("💾 " + msg))
}

func PrintKey(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(TitleStyle.Render("🔑 " + msg))
}

func PrintStats(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(InfoStyle.Render("📊 " + msg))
}

func PrintSparkle(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(TitleStyle.Render("✨ " + msg))
}

func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func FormatTime(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d)/float64(time.Millisecond))
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
