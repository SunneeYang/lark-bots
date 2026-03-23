package common

import (
	"errors"
	"testing"
)

func TestHandlerError_Error(t *testing.T) {
	err := &HandlerError{
		Code:        "AUTH_FAILED",
		Message:     "认证失败",
		ReplyToUser: true,
	}

	expected := "[AUTH_FAILED] 认证失败"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestHandlerError_Error_WithCause(t *testing.T) {
	cause := errors.New("underlying error")
	err := &HandlerError{
		Code:        "EXECUTION_FAILED",
		Message:     "执行失败",
		Cause:       cause,
		ReplyToUser: true,
	}

	expected := "[EXECUTION_FAILED] 执行失败: underlying error"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestHandlerError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &HandlerError{
		Code:        "TEST_ERROR",
		Message:     "测试错误",
		Cause:       cause,
		ReplyToUser: false,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Expected unwrapped error to be the cause, got %v", unwrapped)
	}
}

func TestHandlerError_Unwrap_Nil(t *testing.T) {
	err := &HandlerError{
		Code:        "TEST_ERROR",
		Message:     "测试错误",
		ReplyToUser: false,
	}

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Expected nil unwrapped error, got %v", unwrapped)
	}
}

func TestIsHandlerError(t *testing.T) {
	handlerErr := &HandlerError{
		Code:    "TEST_ERROR",
		Message: "测试错误",
	}

	if !IsHandlerError(handlerErr) {
		t.Error("Expected IsHandlerError to return true for HandlerError")
	}

	regularErr := errors.New("regular error")
	if IsHandlerError(regularErr) {
		t.Error("Expected IsHandlerError to return false for regular error")
	}

	nilErr := error(nil)
	if IsHandlerError(nilErr) {
		t.Error("Expected IsHandlerError to return false for nil error")
	}
}

func TestNewHandlerError(t *testing.T) {
	cause := errors.New("cause error")
	err := NewHandlerError("TEST_CODE", "测试消息", cause, true)

	if err.Code != "TEST_CODE" {
		t.Errorf("Expected Code 'TEST_CODE', got '%s'", err.Code)
	}

	if err.Message != "测试消息" {
		t.Errorf("Expected Message '测试消息', got '%s'", err.Message)
	}

	if err.Cause != cause {
		t.Error("Expected Cause to be the provided error")
	}

	if !err.ReplyToUser {
		t.Error("Expected ReplyToUser to be true")
	}
}

func TestNewAuthFailedError(t *testing.T) {
	cause := errors.New("invalid credentials")
	err := NewAuthFailedError("认证失败", cause)

	if err.Code != ErrCodeAuthFailed {
		t.Errorf("Expected Code '%s', got '%s'", ErrCodeAuthFailed, err.Code)
	}

	if !err.ReplyToUser {
		t.Error("Expected ReplyToUser to be true for AuthFailedError")
	}

	if err.Cause != cause {
		t.Error("Expected Cause to be preserved")
	}
}

func TestNewTaskNotAllowedError(t *testing.T) {
	err := NewTaskNotAllowedError("malicious.sh")

	if err.Code != ErrCodeTaskNotAllowed {
		t.Errorf("Expected Code '%s', got '%s'", ErrCodeTaskNotAllowed, err.Code)
	}

	if !err.ReplyToUser {
		t.Error("Expected ReplyToUser to be true for TaskNotAllowedError")
	}

	if err.Cause != nil {
		t.Error("Expected nil Cause for TaskNotAllowedError")
	}
}

func TestNewExecutionFailedError(t *testing.T) {
	cause := errors.New("script not found")
	err := NewExecutionFailedError("脚本执行失败", cause)

	if err.Code != ErrCodeExecutionFailed {
		t.Errorf("Expected Code '%s', got '%s'", ErrCodeExecutionFailed, err.Code)
	}

	if !err.ReplyToUser {
		t.Error("Expected ReplyToUser to be true for ExecutionFailedError")
	}

	if err.Cause != cause {
		t.Error("Expected Cause to be preserved")
	}
}

func TestNewUnauthorizedError(t *testing.T) {
	err := NewUnauthorizedError("未授权的访问")

	if err.Code != ErrCodeUnauthorized {
		t.Errorf("Expected Code '%s', got '%s'", ErrCodeUnauthorized, err.Code)
	}

	if err.ReplyToUser {
		t.Error("Expected ReplyToUser to be false for UnauthorizedError")
	}

	if err.Cause != nil {
		t.Error("Expected nil Cause for UnauthorizedError")
	}
}

func TestErrorCodes(t *testing.T) {
	// Test that all error code constants are defined
	codes := []string{
		ErrCodeAuthFailed,
		ErrCodeTaskNotAllowed,
		ErrCodeExecutionFailed,
		ErrCodeUnauthorized,
		ErrCodeInvalidMessage,
		ErrCodeUnknownBot,
		ErrCodeUnknownRole,
	}

	expectedCodes := []string{
		"AUTH_FAILED",
		"TASK_NOT_ALLOWED",
		"EXECUTION_FAILED",
		"UNAUTHORIZED",
		"INVALID_MESSAGE",
		"UNKNOWN_BOT",
		"UNKNOWN_ROLE",
	}

	for i, code := range codes {
		if code != expectedCodes[i] {
			t.Errorf("Expected error code '%s', got '%s'", expectedCodes[i], code)
		}
	}
}

func TestNewInvalidMessageError(t *testing.T) {
	cause := errors.New("malformed JSON")
	err := NewInvalidMessageError("消息格式错误", cause)

	if err.Code != ErrCodeInvalidMessage {
		t.Errorf("Expected Code '%s', got '%s'", ErrCodeInvalidMessage, err.Code)
	}

	if !err.ReplyToUser {
		t.Error("Expected ReplyToUser to be true for InvalidMessageError")
	}

	if err.Cause != cause {
		t.Error("Expected Cause to be preserved")
	}
}

func TestNewUnknownBotError(t *testing.T) {
	err := NewUnknownBotError("cli_nonexistent")

	if err.Code != ErrCodeUnknownBot {
		t.Errorf("Expected Code '%s', got '%s'", ErrCodeUnknownBot, err.Code)
	}

	if err.ReplyToUser {
		t.Error("Expected ReplyToUser to be false for UnknownBotError")
	}

	if err.Cause != nil {
		t.Error("Expected nil Cause for UnknownBotError")
	}
}

func TestNewUnknownRoleError(t *testing.T) {
	err := NewUnknownRoleError("invalid_role")

	if err.Code != ErrCodeUnknownRole {
		t.Errorf("Expected Code '%s', got '%s'", ErrCodeUnknownRole, err.Code)
	}

	if err.ReplyToUser {
		t.Error("Expected ReplyToUser to be false for UnknownRoleError")
	}

	if err.Cause != nil {
		t.Error("Expected nil Cause for UnknownRoleError")
	}
}
