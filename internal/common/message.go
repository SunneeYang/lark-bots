package common

import (
	"fmt"
	"strings"
)

// FormatTaskMessage 格式化任务消息
func FormatTaskMessage(taskName, status, output string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("任务 [%s] %s\n", taskName, status))

	if output != "" {
		// 处理多行输出
		lines := strings.Split(output, "\n")
		if len(lines) > 10 {
			sb.WriteString("输出：\n")
			for _, line := range lines[:10] {
				sb.WriteString(line + "\n")
			}
			sb.WriteString("...（输出已截断，完整日志请查看任务记录）")
		} else {
			sb.WriteString(fmt.Sprintf("输出：\n%s", output))
		}
	}

	return sb.String()
}

// FormatErrorMessage 格式化错误消息
func FormatErrorMessage(taskName, errorMsg string) string {
	return fmt.Sprintf("任务 [%s] 执行失败\n原因：%s", taskName, errorMsg)
}

// EscapeOutput 转义输出中的特殊字符
func EscapeOutput(output string) string {
	// 转义引号
	output = strings.ReplaceAll(output, "\"", "\\\"")
	// 转义换行符为 \n
	output = strings.ReplaceAll(output, "\n", "\\n")
	return output
}
