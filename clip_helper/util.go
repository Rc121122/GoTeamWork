package clip_helper

import "fmt"

// HumanFileSize returns a readable representation of bytes.
func HumanFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		exp++
		div *= unit
	}
	value := float64(size) / float64(div)
	return fmt.Sprintf("%.1f %cB", value, "KMGTPE"[exp])
}
