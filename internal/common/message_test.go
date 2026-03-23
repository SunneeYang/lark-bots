package common

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatTaskMessage(t *testing.T) {
	result := FormatTaskMessage("deploy.sh", "执行成功", "部署完成")

	expected := "任务 [deploy.sh] 执行成功\n输出：\n部署完成"
	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestFormatTaskMessage_LongOutput(t *testing.T) {
	// 创建超过 10 行的输出
	var output string
	for i := 0; i < 15; i++ {
		output += fmt.Sprintf("Line %d\n", i)
	}

	result := FormatTaskMessage("test.sh", "completed", output)

	if !strings.Contains(result, "（输出已截断") {
		t.Error("Expected truncation message for long output")
	}

	// 验证只有前 10 行
	lines := strings.Split(result, "\n")
	outputStartIndex := -1
	for i, line := range lines {
		if strings.Contains(line, "输出：") {
			outputStartIndex = i
			break
		}
	}

	if outputStartIndex >= 0 {
		// 从 "输出：" 之后到截断消息之间的行数应该 <= 10
		count := 0
		for i := outputStartIndex + 1; i < len(lines); i++ {
			if strings.Contains(lines[i], "（输出已截断") {
				break
			}
			count++
		}

		if count > 10 {
			t.Errorf("Expected max 10 lines before truncation, got %d", count)
		}
	}
}

func TestFormatTaskMessage_EmptyOutput(t *testing.T) {
	result := FormatTaskMessage("test.sh", "completed", "")

	if strings.Contains(result, "输出：") {
		t.Error("Expected no output section for empty output")
	}
}

func TestFormatErrorMessage(t *testing.T) {
	result := FormatErrorMessage("deploy.sh", "配置文件不存在")

	expected := "任务 [deploy.sh] 执行失败\n原因：配置文件不存在"
	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestEscapeOutput(t *testing.T) {
	input := `Line 1
Line 2
Line "with quotes"`

	result := EscapeOutput(input)

	if !strings.Contains(result, "\\n") {
		t.Error("Expected newline escaping")
	}

	if !strings.Contains(result, "\\\"") {
		t.Error("Expected quote escaping")
	}

	// 验证换行符被正确转义
	if strings.Contains(result, "\n") {
		t.Error("Newlines should be escaped")
	}

	// 验证引号被正确转义
	if strings.Contains(result, `"`) && !strings.Contains(result, `\"`) {
		t.Error("Quotes should be escaped")
	}
}

func TestEscapeOutput_MultipleNewlines(t *testing.T) {
	input := "Line1\nLine2\nLine3"
	result := EscapeOutput(input)

	expectedCount := strings.Count(input, "\n")
	actualCount := strings.Count(result, "\\n")

	if expectedCount != actualCount {
		t.Errorf("Expected %d escaped newlines, got %d", expectedCount, actualCount)
	}
}
