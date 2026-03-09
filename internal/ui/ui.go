package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
)

// Logger instance
var Logger *log.Logger

func init() {
	Logger = log.NewWithOptions(nil, log.Options{
		ReportCaller:    false,
		ReportTimestamp: false,
		Level:           log.InfoLevel,
	})
}

// Logo ASCII art for Vaulty
const logo = `
██╗   ██╗ █████╗ ██╗   ██╗██╗  ████████╗██╗   ██╗
██║   ██║██╔══██╗██║   ██║██║  ╚══██╔══╝╚██╗ ██╔╝
██║   ██║███████║██║   ██║██║     ██║    ╚████╔╝ 
╚██╗ ██╔╝██╔══██║██║   ██║██║     ██║     ╚██╔╝  
 ╚████╔╝ ██║  ██║╚██████╔╝███████╗██║      ██║   
  ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝      ╚═╝   
`

// PrintLogo prints the Vaulty ASCII logo
func PrintLogo() {
	fmt.Println(TitleStyle.Render(logo))
}

// PrintSuccess prints a success message with emoji
func PrintSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(SuccessStyle.Render("✅ " + msg))
}

// PrintError prints an error message with emoji
func PrintError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(ErrorStyle.Render("❌ " + msg))
}

// PrintWarning prints a warning message with emoji
func PrintWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(WarningStyle.Render("⚠️  " + msg))
}

// PrintInfo prints an info message with emoji
func PrintInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(InfoStyle.Render("📦 " + msg))
}

// PrintLock prints a lock message with emoji
func PrintLock(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(TitleStyle.Render("🔒 " + msg))
}

// PrintUnlock prints an unlock message with emoji
func PrintUnlock(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(SuccessStyle.Render("🔓 " + msg))
}

// PrintCloud prints a cloud message with emoji
func PrintCloud(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(InfoStyle.Render("☁️  " + msg))
}

// PrintSave prints a save message with emoji
func PrintSave(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(SuccessStyle.Render("💾 " + msg))
}

// PrintKey prints a key message with emoji
func PrintKey(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(TitleStyle.Render("🔑 " + msg))
}

// PrintStats prints a stats message with emoji
func PrintStats(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(InfoStyle.Render("📊 " + msg))
}

// PrintSparkle prints a sparkle message with emoji
func PrintSparkle(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(TitleStyle.Render("✨ " + msg))
}

// FormatBytes formats bytes into human-readable format
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

// FormatTime formats a duration into human-readable format
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
