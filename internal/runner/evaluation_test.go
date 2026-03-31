package runner

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestSchemaValidator_ValidateOutput(t *testing.T) {
	t.Parallel()

	v := NewSchemaValidator()

	tests := []struct {
		name    string
		output  map[string]any
		schema  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name:   "valid output with all required fields",
			output: map[string]any{"name": "test", "age": 25, "active": true},
			schema: map[string]any{
				"required": []any{"name", "age"},
				"properties": map[string]any{
					"name":   map[string]any{"type": "string"},
					"age":    map[string]any{"type": "integer"},
					"active": map[string]any{"type": "boolean"},
				},
			},
			wantErr: false,
		},
		{
			name:   "missing required field",
			output: map[string]any{"name": "test"},
			schema: map[string]any{
				"required": []any{"name", "age"},
			},
			wantErr: true,
			errMsg:  "required field \"age\" is missing",
		},
		{
			name:   "wrong field type string",
			output: map[string]any{"name": 123},
			schema: map[string]any{
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
			wantErr: true,
			errMsg:  "field \"name\" must be string",
		},
		{
			name:   "wrong field type number",
			output: map[string]any{"age": "twenty"},
			schema: map[string]any{
				"properties": map[string]any{
					"age": map[string]any{"type": "number"},
				},
			},
			wantErr: true,
			errMsg:  "field \"age\" must be numeric",
		},
		{
			name:   "value below minimum",
			output: map[string]any{"score": 5},
			schema: map[string]any{
				"properties": map[string]any{
					"score": map[string]any{"type": "number", "minimum": 10},
				},
			},
			wantErr: true,
			errMsg:  "field \"score\" value 5 is less than minimum 10",
		},
		{
			name:   "value exceeds maximum",
			output: map[string]any{"score": 150},
			schema: map[string]any{
				"properties": map[string]any{
					"score": map[string]any{"type": "number", "maximum": 100},
				},
			},
			wantErr: true,
			errMsg:  "field \"score\" value 150 exceeds maximum 100",
		},
		{
			name:   "string too short",
			output: map[string]any{"code": "ab"},
			schema: map[string]any{
				"properties": map[string]any{
					"code": map[string]any{"type": "string", "minLength": 3},
				},
			},
			wantErr: true,
			errMsg:  "field \"code\" string length 2 is less than minLength 3",
		},
		{
			name:   "string too long",
			output: map[string]any{"code": "abcdef"},
			schema: map[string]any{
				"properties": map[string]any{
					"code": map[string]any{"type": "string", "maxLength": 5},
				},
			},
			wantErr: true,
			errMsg:  "field \"code\" string length 6 exceeds maxLength 5",
		},
		{
			name:   "pattern mismatch",
			output: map[string]any{"email": "not-an-email"},
			schema: map[string]any{
				"properties": map[string]any{
					"email": map[string]any{"type": "string", "pattern": `^[a-z]+@[a-z]+\.[a-z]+$`},
				},
			},
			wantErr: true,
			errMsg:  "field \"email\" value \"not-an-email\" does not match pattern",
		},
		{
			name:   "pattern match",
			output: map[string]any{"email": "test@example.com"},
			schema: map[string]any{
				"properties": map[string]any{
					"email": map[string]any{"type": "string", "pattern": `^[a-z]+@[a-z]+\.[a-z]+$`},
				},
			},
			wantErr: false,
		},
		{
			name:   "enum mismatch",
			output: map[string]any{"status": "unknown"},
			schema: map[string]any{
				"properties": map[string]any{
					"status": map[string]any{"type": "string", "enum": []any{"active", "inactive"}},
				},
			},
			wantErr: true,
			errMsg:  "field \"status\" value unknown not in enum",
		},
		{
			name:   "enum match",
			output: map[string]any{"status": "active"},
			schema: map[string]any{
				"properties": map[string]any{
					"status": map[string]any{"type": "string", "enum": []any{"active", "inactive"}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := v.ValidateOutput(tt.output, tt.schema)

			if tt.wantErr {
				if result.Valid {
					t.Errorf("ValidateOutput() valid = true, want false")
				}
				if len(result.Errors) == 0 {
					t.Errorf("ValidateOutput() errors = [], want error containing %q", tt.errMsg)
				} else if !containsString(result.Errors[0], tt.errMsg) {
					t.Errorf("ValidateOutput() errors[0] = %q, want to contain %q", result.Errors[0], tt.errMsg)
				}
			} else {
				if !result.Valid {
					t.Errorf("ValidateOutput() valid = false, want true; errors = %v", result.Errors)
				}
			}
		})
	}
}

func TestDeterministicValidator_ValidateDeterministic(t *testing.T) {
	t.Parallel()

	v := NewDeterministicValidator()

	tests := []struct {
		name     string
		output   map[string]any
		expected map[string]any
		wantErr  bool
	}{
		{
			name:     "deterministic fields match",
			output:   map[string]any{"id": "123", "timestamp": "2024-01-01T00:00:00Z"},
			expected: map[string]any{"_deterministic": []any{"id", "timestamp"}, "id": "123", "timestamp": "2024-01-01T00:00:00Z"},
			wantErr:  false,
		},
		{
			name:     "deterministic fields mismatch",
			output:   map[string]any{"id": "123", "timestamp": "2024-01-02T00:00:00Z"},
			expected: map[string]any{"_deterministic": []any{"id", "timestamp"}, "id": "123", "timestamp": "2024-01-01T00:00:00Z"},
			wantErr:  true,
		},
		{
			name:     "no deterministic fields",
			output:   map[string]any{"id": "123"},
			expected: map[string]any{"id": "456"},
			wantErr:  false,
		},
		{
			name:     "deterministic field missing in output",
			output:   map[string]any{"id": "123"},
			expected: map[string]any{"_deterministic": []any{"timestamp"}, "timestamp": "2024-01-01T00:00:00Z"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := v.ValidateDeterministic(tt.output, tt.expected)

			if tt.wantErr && result.Valid {
				t.Errorf("ValidateDeterministic() valid = true, want false")
			}
			if !tt.wantErr && !result.Valid {
				t.Errorf("ValidateDeterministic() valid = false, want true; errors = %v", result.Errors)
			}
		})
	}
}

func TestDeepEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    any
		b    any
		want bool
	}{
		{
			name: "equal strings",
			a:    "hello",
			b:    "hello",
			want: true,
		},
		{
			name: "unequal strings",
			a:    "hello",
			b:    "world",
			want: false,
		},
		{
			name: "equal numbers",
			a:    42,
			b:    42,
			want: true,
		},
		{
			name: "equal maps",
			a:    map[string]any{"a": 1, "b": 2},
			b:    map[string]any{"a": 1, "b": 2},
			want: true,
		},
		{
			name: "unequal maps",
			a:    map[string]any{"a": 1, "b": 2},
			b:    map[string]any{"a": 1, "b": 3},
			want: false,
		},
		{
			name: "equal arrays",
			a:    []any{1, 2, 3},
			b:    []any{1, 2, 3},
			want: true,
		},
		{
			name: "unequal arrays",
			a:    []any{1, 2, 3},
			b:    []any{1, 2, 4},
			want: false,
		},
		{
			name: "mixed types same value",
			a:    float64(42),
			b:    int64(42),
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := DeepEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("DeepEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestLogAndTrace(t *testing.T) {
	t.Parallel()

	ClearEvaluationLog()
	ClearEvaluationTrace()

	Log("test-runtime", "case-1", LogLevelInfo, "test message", map[string]any{"key": "value"})
	Trace("test-runtime", "case-1", TraceEventStart, "", 0, nil)

	logs := GetEvaluationLog()
	if len(logs) != 1 {
		t.Errorf("GetEvaluationLog() len = %d, want 1", len(logs))
	}
	if logs[0].Message != "test message" {
		t.Errorf("GetEvaluationLog()[0].Message = %q, want %q", logs[0].Message, "test message")
	}
	if logs[0].Level != LogLevelInfo {
		t.Errorf("GetEvaluationLog()[0].Level = %v, want %v", logs[0].Level, LogLevelInfo)
	}

	traces := GetEvaluationTrace()
	if len(traces) != 1 {
		t.Errorf("GetEvaluationTrace() len = %d, want 1", len(traces))
	}
	if traces[0].Event != TraceEventStart {
		t.Errorf("GetEvaluationTrace()[0].Event = %v, want %v", traces[0].Event, TraceEventStart)
	}

	ClearEvaluationLog()
	ClearEvaluationTrace()

	logs = GetEvaluationLog()
	if len(logs) != 0 {
		t.Errorf("After clear, GetEvaluationLog() len = %d, want 0", len(logs))
	}
}

func TestIsolationMonitor(t *testing.T) {
	t.Parallel()

	m := NewIsolationMonitor()

	if m.ActiveProcessCount() != 0 {
		t.Errorf("ActiveProcessCount() = %d, want 0", m.ActiveProcessCount())
	}

	m.TrackProcess(1234)
	m.TrackProcess(5678)

	if m.ActiveProcessCount() != 2 {
		t.Errorf("ActiveProcessCount() = %d, want 2", m.ActiveProcessCount())
	}

	m.UntrackProcess(1234)

	if m.ActiveProcessCount() != 1 {
		t.Errorf("After UntrackProcess, ActiveProcessCount() = %d, want 1", m.ActiveProcessCount())
	}

	m.activeProcesses = make(map[int]struct{})
	if m.ActiveProcessCount() != 0 {
		t.Errorf("After reset, ActiveProcessCount() = %d, want 0", m.ActiveProcessCount())
	}
}

func TestValidationResult_CombinesErrors(t *testing.T) {
	t.Parallel()

	v := NewSchemaValidator()

	output := map[string]any{
		"name": 123,
		"age":  "twenty",
		"code": "ab",
	}
	schema := map[string]any{
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
			"code": map[string]any{"type": "string", "minLength": 3},
		},
	}

	result := v.ValidateOutput(output, schema)

	if result.Valid {
		t.Errorf("ValidateOutput() valid = true, want false")
	}
	if len(result.Errors) != 3 {
		t.Errorf("ValidateOutput() errors count = %d, want 3", len(result.Errors))
	}
}

func TestToFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		want  *float64
	}{
		{"float64", float64(1.5), floatPtr(1.5)},
		{"float32", float32(2.5), floatPtr(2.5)},
		{"int", int(10), floatPtr(10)},
		{"int64", int64(20), floatPtr(20)},
		{"int32", int32(30), floatPtr(30)},
		{"int8", int8(40), floatPtr(40)},
		{"int16", int16(50), floatPtr(50)},
		{"uint", uint(60), floatPtr(60)},
		{"uint64", uint64(70), floatPtr(70)},
		{"uint32", uint32(80), floatPtr(80)},
		{"string", "invalid", nil},
		{"bool", true, nil},
		{"nil", nil, nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toFloat64(tt.value)
			if tt.want == nil {
				if got != nil {
					t.Errorf("toFloat64(%v) = %v, want nil", tt.value, *got)
				}
			} else {
				if got == nil {
					t.Errorf("toFloat64(%v) = nil, want %v", tt.value, *tt.want)
				} else if *got != *tt.want {
					t.Errorf("toFloat64(%v) = %v, want %v", tt.value, *got, *tt.want)
				}
			}
		})
	}
}

func floatPtr(v float64) *float64 {
	return &v
}

func TestIsolationMonitor_KillAll(t *testing.T) {
	t.Parallel()

	m := NewIsolationMonitor()

	m.TrackProcess(1)
	m.TrackProcess(2)
	m.TrackProcess(3)

	if m.ActiveProcessCount() != 3 {
		t.Errorf("ActiveProcessCount() = %d, want 3", m.ActiveProcessCount())
	}

	m.KillAll()

	if m.ActiveProcessCount() != 0 {
		t.Errorf("After KillAll, ActiveProcessCount() = %d, want 0", m.ActiveProcessCount())
	}
}

func TestNewDetachedContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := NewDetachedContext(1 * time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		t.Errorf("ctx.Done() fired immediately, want timeout")
	default:
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Errorf("ctx.Deadline() returned false, want true")
	}

	now := time.Now()
	if deadline.Before(now) {
		t.Errorf("deadline %v is before now %v", deadline, now)
	}
}

func TestMeasureExecution(t *testing.T) {
	t.Parallel()

	metrics, err := MeasureExecution(context.Background(), func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("MeasureExecution() error = %v, want nil", err)
	}

	if metrics.DurationMS < 10 {
		t.Errorf("DurationMS = %d, want >= 10", metrics.DurationMS)
	}

	if metrics.StartTime.IsZero() {
		t.Errorf("StartTime is zero")
	}

	if metrics.EndTime.IsZero() {
		t.Errorf("EndTime is zero")
	}
}

func TestMeasureExecution_WithError(t *testing.T) {
	t.Parallel()

	expectedErr := fmt.Errorf("test error")
	metrics, err := MeasureExecution(context.Background(), func() error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("MeasureExecution() error = %v, want %v", err, expectedErr)
	}

	if metrics.DurationMS < 0 {
		t.Errorf("DurationMS = %d, want >= 0", metrics.DurationMS)
	}
}

func TestSchemaValidator_ValidateOutput_AllTypes(t *testing.T) {
	t.Parallel()

	v := NewSchemaValidator()

	output := map[string]any{
		"strField":   "hello",
		"numField":   float64(100),
		"boolField":  true,
		"arrayField": []any{1, 2, 3},
		"objField":   map[string]any{"nested": "value"},
	}
	schema := map[string]any{
		"properties": map[string]any{
			"strField":   map[string]any{"type": "string"},
			"numField":   map[string]any{"type": "number", "minimum": 50, "maximum": 200},
			"boolField":  map[string]any{"type": "boolean"},
			"arrayField": map[string]any{"type": "array"},
			"objField":   map[string]any{"type": "object"},
		},
	}

	result := v.ValidateOutput(output, schema)
	if !result.Valid {
		t.Errorf("ValidateOutput() valid = false, want true; errors = %v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Errorf("ValidateOutput() errors = %v, want none", result.Errors)
	}
}

func TestSchemaValidator_ValidateOutput_IntType(t *testing.T) {
	t.Parallel()

	v := NewSchemaValidator()

	output := map[string]any{"count": 5}
	schema := map[string]any{
		"properties": map[string]any{
			"count": map[string]any{"type": "integer", "minimum": 0, "maximum": 100},
		},
	}

	result := v.ValidateOutput(output, schema)
	if !result.Valid {
		t.Errorf("ValidateOutput() valid = false, want true; errors = %v", result.Errors)
	}
}

func TestSchemaValidator_ValidateOutput_EdgeCases(t *testing.T) {
	t.Parallel()

	v := NewSchemaValidator()

	tests := []struct {
		name    string
		output  map[string]any
		schema  map[string]any
		wantErr bool
	}{
		{
			name:    "empty schema",
			output:  map[string]any{"field": "value"},
			schema:  map[string]any{},
			wantErr: false,
		},
		{
			name:    "no properties",
			output:  map[string]any{"field": "value"},
			schema:  map[string]any{"type": "object"},
			wantErr: false,
		},
		{
			name:    "missing required with no properties",
			output:  map[string]any{},
			schema:  map[string]any{"required": []any{"field"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := v.ValidateOutput(tt.output, tt.schema)
			if tt.wantErr && result.Valid {
				t.Errorf("ValidateOutput() valid = true, want false")
			}
			if !tt.wantErr && !result.Valid {
				t.Errorf("ValidateOutput() valid = false, want true; errors = %v", result.Errors)
			}
		})
	}
}

func TestDeterministicValidator_ValidateDeterministic_Warnings(t *testing.T) {
	t.Parallel()

	v := NewDeterministicValidator()

	output := map[string]any{"id": "123"}
	expected := map[string]any{
		"_deterministic": []any{"timestamp"},
		"timestamp":      "2024-01-01T00:00:00Z",
	}

	result := v.ValidateDeterministic(output, expected)
	if !result.Valid {
		t.Errorf("ValidateDeterministic() valid = false, want true")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("Warnings count = %d, want 1", len(result.Warnings))
	}
}

func TestRegexMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pattern string
		input   string
		want    bool
		wantErr bool
	}{
		{"^hello$", "hello", true, false},
		{"^hello$", "world", false, false},
		{"[0-9]+", "123", true, false},
		{"[0-9]+", "abc", false, false},
		{"*", "test", false, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := regexpMatch(tt.pattern, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("regexpMatch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("regexpMatch(%q, %q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}
