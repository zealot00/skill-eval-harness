package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"
)

type EvaluationCapability struct{}

var evaluationLog []LogEntry
var evaluationTrace []TraceEntry
var logMu sync.Mutex

type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

type LogEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	Level     LogLevel       `json:"level"`
	Runtime   string         `json:"runtime"`
	CaseID    string         `json:"case_id,omitempty"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

type TraceEvent string

const (
	TraceEventStart       TraceEvent = "start"
	TraceEventWorkdir     TraceEvent = "workdir_created"
	TraceEventInputPrep   TraceEvent = "input_prepared"
	TraceEventExec        TraceEvent = "exec"
	TraceEventExecOutput  TraceEvent = "exec_output"
	TraceEventParseOutput TraceEvent = "parse_output"
	TraceEventValidate    TraceEvent = "validate"
	TraceEventComplete    TraceEvent = "complete"
	TraceEventError       TraceEvent = "error"
)

type TraceEntry struct {
	Timestamp  time.Time  `json:"timestamp"`
	Runtime    string     `json:"runtime"`
	CaseID     string     `json:"case_id,omitempty"`
	Event      TraceEvent `json:"event"`
	SpanID     string     `json:"span_id"`
	ParentID   string     `json:"parent_id,omitempty"`
	DurationMS int64      `json:"duration_ms,omitempty"`
	Error      string     `json:"error,omitempty"`
}

var spanCounter int64
var spanMu sync.Mutex

func nextSpanID() string {
	spanMu.Lock()
	defer spanMu.Unlock()
	spanCounter++
	return fmt.Sprintf("span-%d", spanCounter)
}

func Log(runtime, caseID string, level LogLevel, message string, fields map[string]any) {
	logMu.Lock()
	defer logMu.Unlock()
	evaluationLog = append(evaluationLog, LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Runtime:   runtime,
		CaseID:    caseID,
		Message:   message,
		Fields:    fields,
	})
}

func Trace(runtime, caseID string, event TraceEvent, parentID string, durationMS int64, err error) {
	logMu.Lock()
	defer logMu.Unlock()
	spanID := nextSpanID()
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	evaluationTrace = append(evaluationTrace, TraceEntry{
		Timestamp:  time.Now().UTC(),
		Runtime:    runtime,
		CaseID:     caseID,
		Event:      event,
		SpanID:     spanID,
		ParentID:   parentID,
		DurationMS: durationMS,
		Error:      errStr,
	})
}

func GetEvaluationLog() []LogEntry {
	logMu.Lock()
	defer logMu.Unlock()
	result := make([]LogEntry, len(evaluationLog))
	copy(result, evaluationLog)
	return result
}

func GetEvaluationTrace() []TraceEntry {
	logMu.Lock()
	defer logMu.Unlock()
	result := make([]TraceEntry, len(evaluationTrace))
	copy(result, evaluationTrace)
	return result
}

func ClearEvaluationLog() {
	logMu.Lock()
	defer logMu.Unlock()
	evaluationLog = nil
}

func ClearEvaluationTrace() {
	logMu.Lock()
	defer logMu.Unlock()
	evaluationTrace = nil
}

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type SchemaValidator struct{}

func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{}
}

func (v *SchemaValidator) ValidateOutput(output map[string]any, schema map[string]any) ValidationResult {
	result := ValidationResult{Valid: true}

	requiredFields, ok := schema["required"].([]any)
	if ok {
		for _, field := range requiredFields {
			fieldStr, ok := field.(string)
			if !ok {
				continue
			}
			if _, exists := output[fieldStr]; !exists {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("required field %q is missing", fieldStr))
			}
		}
	}

	if properties, ok := schema["properties"].(map[string]any); ok {
		for fieldName, fieldSchema := range properties {
			fieldValue, exists := output[fieldName]
			if !exists {
				continue
			}
			if err := v.validateField(fieldName, fieldValue, fieldSchema); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, err.Error())
			}
		}
	}

	return result
}

func (v *SchemaValidator) validateField(name string, value any, schema any) error {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return nil
	}

	if typeStr, ok := schemaMap["type"].(string); ok {
		switch typeStr {
		case "string":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("field %q must be string, got %T", name, value)
			}
		case "number", "integer":
			if !isNumeric(value) {
				return fmt.Errorf("field %q must be numeric, got %T", name, value)
			}
		case "boolean":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("field %q must be boolean, got %T", name, value)
			}
		case "array":
			if _, ok := value.([]any); !ok {
				return fmt.Errorf("field %q must be array, got %T", name, value)
			}
		case "object":
			if _, ok := value.(map[string]any); !ok {
				return fmt.Errorf("field %q must be object, got %T", name, value)
			}
		}
	}

	if enum, ok := schemaMap["enum"].([]any); ok {
		found := false
		valStr := fmt.Sprintf("%v", value)
		for _, e := range enum {
			if fmt.Sprintf("%v", e) == valStr {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("field %q value %v not in enum", name, value)
		}
	}

	numVal := toFloat64(value)

	if minVal := toFloat64(schemaMap["minimum"]); minVal != nil {
		if numVal != nil && *numVal < *minVal {
			return fmt.Errorf("field %q value %v is less than minimum %v", name, *numVal, *minVal)
		}
	}

	if maxVal := toFloat64(schemaMap["maximum"]); maxVal != nil {
		if numVal != nil && *numVal > *maxVal {
			return fmt.Errorf("field %q value %v exceeds maximum %v", name, *numVal, *maxVal)
		}
	}

	if minLen := toFloat64(schemaMap["minLength"]); minLen != nil {
		if str, ok := value.(string); ok {
			if len(str) < int(*minLen) {
				return fmt.Errorf("field %q string length %d is less than minLength %v", name, len(str), int(*minLen))
			}
		}
	}

	if maxLen := toFloat64(schemaMap["maxLength"]); maxLen != nil {
		if str, ok := value.(string); ok {
			if len(str) > int(*maxLen) {
				return fmt.Errorf("field %q string length %d exceeds maxLength %v", name, len(str), int(*maxLen))
			}
		}
	}

	if pattern, ok := schemaMap["pattern"].(string); ok {
		if str, ok := value.(string); ok {
			matched, err := regexpMatch(pattern, str)
			if err != nil {
				return fmt.Errorf("field %q invalid pattern %q", name, pattern)
			}
			if !matched {
				return fmt.Errorf("field %q value %q does not match pattern", name, str)
			}
		}
	}

	return nil
}

func isNumeric(v any) bool {
	switch v.(type) {
	case float64, float32, int, int64, int32, int8, int16, uint, uint64, uint32:
		return true
	}
	return false
}

func toFloat64(v any) *float64 {
	switch val := v.(type) {
	case float64:
		f := float64(val)
		return &f
	case float32:
		f := float64(val)
		return &f
	case int:
		f := float64(val)
		return &f
	case int64:
		f := float64(val)
		return &f
	case int32:
		f := float64(val)
		return &f
	case int8:
		f := float64(val)
		return &f
	case int16:
		f := float64(val)
		return &f
	case uint:
		f := float64(val)
		return &f
	case uint64:
		f := float64(val)
		return &f
	case uint32:
		f := float64(val)
		return &f
	}
	return nil
}

var regexCache = make(map[string]*regexpWrapper)
var regexMu sync.Mutex

type regexpWrapper struct {
	pattern *regexp.Regexp
}

func regexpMatch(pattern, str string) (bool, error) {
	regexMu.Lock()
	defer regexMu.Unlock()

	cached, ok := regexCache[pattern]
	if !ok {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, err
		}
		cached = &regexpWrapper{pattern: re}
		regexCache[pattern] = cached
	}

	return cached.pattern.MatchString(str), nil
}

type DeterministicValidator struct{}

func NewDeterministicValidator() *DeterministicValidator {
	return &DeterministicValidator{}
}

func (v *DeterministicValidator) ValidateDeterministic(output map[string]any, expected map[string]any) ValidationResult {
	result := ValidationResult{Valid: true}

	deterministicFields, ok := expected["_deterministic"].([]any)
	if !ok {
		return result
	}

	for _, field := range deterministicFields {
		fieldName, ok := field.(string)
		if !ok {
			continue
		}

		outputVal, exists := output[fieldName]
		if !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("deterministic field %q not found in output", fieldName))
			continue
		}

		expectedVal, exists := expected[fieldName]
		if !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("deterministic field %q not found in expected", fieldName))
			continue
		}

		if !DeepEqual(outputVal, expectedVal) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("deterministic field %q: output %v != expected %v", fieldName, outputVal, expectedVal))
		}
	}

	return result
}

func DeepEqual(a, b any) bool {
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return a == b
	}
	return string(aJSON) == string(bJSON)
}

type IsolationMonitor struct {
	activeProcesses map[int]struct{}
	mu              sync.Mutex
}

func NewIsolationMonitor() *IsolationMonitor {
	return &IsolationMonitor{
		activeProcesses: make(map[int]struct{}),
	}
}

func (m *IsolationMonitor) TrackProcess(pid int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeProcesses[pid] = struct{}{}
}

func (m *IsolationMonitor) UntrackProcess(pid int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.activeProcesses, pid)
}

func (m *IsolationMonitor) ActiveProcessCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.activeProcesses)
}

func (m *IsolationMonitor) KillAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for pid := range m.activeProcesses {
		processKill(pid)
	}
	m.activeProcesses = make(map[int]struct{})
}

func processKill(pid int) {}

func NewDetachedContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return ctx, cancel
}

type ExecutionMetrics struct {
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	DurationMS   int64     `json:"duration_ms"`
	MemoryUsedKB int64     `json:"memory_used_kb,omitempty"`
	CPUPercent   float64   `json:"cpu_percent,omitempty"`
	ExitCode     int       `json:"exit_code,omitempty"`
	Signal       string    `json:"signal,omitempty"`
}

func MeasureExecution(ctx context.Context, fn func() error) (ExecutionMetrics, error) {
	metrics := ExecutionMetrics{
		StartTime: time.Now(),
	}
	err := fn()
	metrics.EndTime = time.Now()
	metrics.DurationMS = metrics.EndTime.Sub(metrics.StartTime).Milliseconds()
	return metrics, err
}
