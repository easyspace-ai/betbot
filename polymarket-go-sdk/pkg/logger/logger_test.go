package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestStandardLogger_Levels(t *testing.T) {
	tests := []struct {
		name          string
		level         Level
		logFunc       func(Logger)
		shouldLog     bool
		expectedLevel string
	}{
		{
			name:          "Debug logs when level is Debug",
			level:         LevelDebug,
			logFunc:       func(l Logger) { l.Debug("test message") },
			shouldLog:     true,
			expectedLevel: "DEBUG",
		},
		{
			name:          "Debug does not log when level is Info",
			level:         LevelInfo,
			logFunc:       func(l Logger) { l.Debug("test message") },
			shouldLog:     false,
			expectedLevel: "",
		},
		{
			name:          "Info logs when level is Info",
			level:         LevelInfo,
			logFunc:       func(l Logger) { l.Info("test message") },
			shouldLog:     true,
			expectedLevel: "INFO",
		},
		{
			name:          "Info does not log when level is Warn",
			level:         LevelWarn,
			logFunc:       func(l Logger) { l.Info("test message") },
			shouldLog:     false,
			expectedLevel: "",
		},
		{
			name:          "Warn logs when level is Warn",
			level:         LevelWarn,
			logFunc:       func(l Logger) { l.Warn("test message") },
			shouldLog:     true,
			expectedLevel: "WARN",
		},
		{
			name:          "Error logs when level is Error",
			level:         LevelError,
			logFunc:       func(l Logger) { l.Error("test message") },
			shouldLog:     true,
			expectedLevel: "ERROR",
		},
		{
			name:          "Nothing logs when level is None",
			level:         LevelNone,
			logFunc:       func(l Logger) { l.Error("test message") },
			shouldLog:     false,
			expectedLevel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewStandardLogger(tt.level, buf)

			tt.logFunc(logger)

			output := buf.String()
			if tt.shouldLog {
				if output == "" {
					t.Errorf("Expected log output, got none")
				}
				if !strings.Contains(output, tt.expectedLevel) {
					t.Errorf("Expected level %s in output, got: %s", tt.expectedLevel, output)
				}
				if !strings.Contains(output, "test message") {
					t.Errorf("Expected 'test message' in output, got: %s", output)
				}
			} else {
				if output != "" {
					t.Errorf("Expected no log output, got: %s", output)
				}
			}
		})
	}
}

func TestStandardLogger_SetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStandardLogger(LevelInfo, buf)

	// Initially at Info level
	logger.Debug("should not log")
	if buf.String() != "" {
		t.Errorf("Debug should not log at Info level")
	}

	// Change to Debug level
	logger.SetLevel(LevelDebug)
	logger.Debug("should log")
	if buf.String() == "" {
		t.Errorf("Debug should log at Debug level")
	}

	// Verify GetLevel
	if logger.GetLevel() != LevelDebug {
		t.Errorf("GetLevel() = %v, want %v", logger.GetLevel(), LevelDebug)
	}
}

func TestStandardLogger_Formatting(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStandardLogger(LevelInfo, buf)

	logger.Info("test %s %d", "message", 42)

	output := buf.String()
	if !strings.Contains(output, "test message 42") {
		t.Errorf("Expected formatted message, got: %s", output)
	}
}

func TestNoOpLogger(t *testing.T) {
	logger := NewNoOpLogger()

	// Should not panic
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
	logger.SetLevel(LevelDebug)

	if logger.GetLevel() != LevelNone {
		t.Errorf("NoOpLogger.GetLevel() = %v, want %v", logger.GetLevel(), LevelNone)
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelNone, "NONE"},
		{Level(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("Level.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultLogger(t *testing.T) {
	// Save original default logger
	original := GetDefault()
	defer SetDefault(original)

	// Create a custom logger
	buf := &bytes.Buffer{}
	customLogger := NewStandardLogger(LevelDebug, buf)
	SetDefault(customLogger)

	// Test that default logger is used
	Debug("test debug")
	if buf.String() == "" {
		t.Errorf("Expected debug message in custom logger")
	}

	// Test that GetDefault returns the custom logger
	if GetDefault() != customLogger {
		t.Errorf("GetDefault() did not return custom logger")
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Save original default logger
	original := GetDefault()
	defer SetDefault(original)

	buf := &bytes.Buffer{}
	customLogger := NewStandardLogger(LevelDebug, buf)
	SetDefault(customLogger)

	// Test global functions
	Debug("debug %s", "msg")
	Info("info %s", "msg")
	Warn("warn %s", "msg")
	Error("error %s", "msg")

	output := buf.String()
	if !strings.Contains(output, "DEBUG") {
		t.Errorf("Expected DEBUG in output")
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected INFO in output")
	}
	if !strings.Contains(output, "WARN") {
		t.Errorf("Expected WARN in output")
	}
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Expected ERROR in output")
	}
}

func TestSetLevel_Global(t *testing.T) {
	// Save original default logger
	original := GetDefault()
	defer SetDefault(original)

	buf := &bytes.Buffer{}
	customLogger := NewStandardLogger(LevelInfo, buf)
	SetDefault(customLogger)

	// Test SetLevel global function
	SetLevel(LevelWarn)
	if GetLevel() != LevelWarn {
		t.Errorf("SetLevel() did not update level")
	}

	// Verify Info doesn't log at Warn level
	Info("should not log")
	if buf.String() != "" {
		t.Errorf("Info should not log at Warn level")
	}
}

func TestConcurrentAccess(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStandardLogger(LevelInfo, buf)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			logger.Info("message %d", id)
			logger.SetLevel(LevelDebug)
			logger.Debug("debug %d", id)
			_ = logger.GetLevel()
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have some output
	if buf.String() == "" {
		t.Errorf("Expected some log output from concurrent access")
	}
}
