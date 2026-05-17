package log

import (
	"testing"
)

func TestInit_DefaultInfo(t *testing.T) {
	Init("")
	if Logger == nil {
		t.Fatal("expected Logger to be initialized")
	}
}

func TestInit_Debug(t *testing.T) {
	Init("debug")
	if Logger == nil {
		t.Fatal("expected Logger to be initialized")
	}
}

func TestInit_Warn(t *testing.T) {
	Init("warn")
	if Logger == nil {
		t.Fatal("expected Logger to be initialized")
	}
}

func TestInit_Error(t *testing.T) {
	Init("error")
	if Logger == nil {
		t.Fatal("expected Logger to be initialized")
	}
}

func TestInit_UnknownDefaultsToInfo(t *testing.T) {
	Init("invalid")
	if Logger == nil {
		t.Fatal("expected Logger to be initialized")
	}
}

func TestDebug_NilLogger(t *testing.T) {
	Logger = nil
	Debug("test") // should not panic
}

func TestInfo_NilLogger(t *testing.T) {
	Logger = nil
	Info("test") // should not panic
}

func TestWarn_NilLogger(t *testing.T) {
	Logger = nil
	Warn("test") // should not panic
}

func TestError_NilLogger(t *testing.T) {
	Logger = nil
	Error("test") // should not panic
}

func TestDebug_WithLogger(t *testing.T) {
	Init("debug")
	Debug("test message", "key", "value") // should not panic
}

func TestInfo_WithLogger(t *testing.T) {
	Init("info")
	Info("test message", "key", "value") // should not panic
}

func TestWarn_WithLogger(t *testing.T) {
	Init("warn")
	Warn("test message") // should not panic
}

func TestError_WithLogger(t *testing.T) {
	Init("error")
	Error("test message") // should not panic
}
