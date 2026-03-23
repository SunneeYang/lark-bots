package common

import (
	"fmt"
)

// HandlerError 处理器错误
type HandlerError struct {
	Code        string
	Message     string
	Cause       error
	ReplyToUser bool
}

// Error 实现 error 接口
func (e *HandlerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 返回底层错误
func (e *HandlerError) Unwrap() error {
	return e.Cause
}

// IsHandlerError 检查错误是否为 HandlerError
func IsHandlerError(err error) bool {
	_, ok := err.(*HandlerError)
	return ok
}

// 预定义错误代码
const (
	ErrCodeAuthFailed      = "AUTH_FAILED"
	ErrCodeTaskNotAllowed  = "TASK_NOT_ALLOWED"
	ErrCodeExecutionFailed = "EXECUTION_FAILED"
	ErrCodeUnauthorized    = "UNAUTHORIZED"
	ErrCodeInvalidMessage  = "INVALID_MESSAGE"
	ErrCodeUnknownBot      = "UNKNOWN_BOT"
	ErrCodeUnknownRole     = "UNKNOWN_ROLE"
)

// NewHandlerError 创建 HandlerError
func NewHandlerError(code, message string, cause error, replyToUser bool) *HandlerError {
	return &HandlerError{
		Code:        code,
		Message:     message,
		Cause:       cause,
		ReplyToUser: replyToUser,
	}
}

// 预定义错误构造函数

// NewAuthFailedError 创建认证失败错误
func NewAuthFailedError(message string, cause error) *HandlerError {
	return NewHandlerError(ErrCodeAuthFailed, message, cause, true)
}

// NewTaskNotAllowedError 创建任务不允许错误
func NewTaskNotAllowedError(taskName string) *HandlerError {
	return NewHandlerError(ErrCodeTaskNotAllowed, fmt.Sprintf("任务不允许执行: %s", taskName), nil, true)
}

// NewExecutionFailedError 创建执行失败错误
func NewExecutionFailedError(message string, cause error) *HandlerError {
	return NewHandlerError(ErrCodeExecutionFailed, message, cause, true)
}

// NewUnauthorizedError 创建未授权错误
func NewUnauthorizedError(message string) *HandlerError {
	return NewHandlerError(ErrCodeUnauthorized, message, nil, false)
}

// NewInvalidMessageError 创建无效消息错误
func NewInvalidMessageError(message string, cause error) *HandlerError {
	return NewHandlerError(ErrCodeInvalidMessage, message, cause, true)
}

// NewUnknownBotError 创建未知机器人错误
func NewUnknownBotError(botID string) *HandlerError {
	return NewHandlerError(ErrCodeUnknownBot, fmt.Sprintf("未知的机器人: %s", botID), nil, false)
}

// NewUnknownRoleError 创建未知角色错误
func NewUnknownRoleError(role string) *HandlerError {
	return NewHandlerError(ErrCodeUnknownRole, fmt.Sprintf("未知的角色: %s", role), nil, false)
}
