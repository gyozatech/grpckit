package grpckit

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormMarshaler_ContentType(t *testing.T) {
	m := &FormMarshaler{}
	ct := m.ContentType(nil)

	if ct != "application/x-www-form-urlencoded" {
		t.Errorf("expected application/x-www-form-urlencoded, got %s", ct)
	}
}

func TestInferType(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		{"true", true},
		{"false", false},
		{"123", int64(123)},
		{"-456", int64(-456)},
		{"3.14", float64(3.14)},
		{"hello", "hello"},
		{"", ""},
		{"0", int64(0)},
		{"1", int64(1)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := inferType(tt.input)
			if result != tt.expected {
				t.Errorf("inferType(%q) = %v (%T), want %v (%T)", tt.input, result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestValuesToJSON(t *testing.T) {
	tests := []struct {
		name     string
		values   map[string][]string
		contains []string
	}{
		{
			name: "simple values",
			values: map[string][]string{
				"name": {"John"},
				"age":  {"30"},
			},
			contains: []string{`"name":"John"`, `"age":30`},
		},
		{
			name: "boolean value",
			values: map[string][]string{
				"active": {"true"},
			},
			contains: []string{`"active":true`},
		},
		{
			name: "array values",
			values: map[string][]string{
				"tags": {"a", "b", "c"},
			},
			contains: []string{`"tags":["a","b","c"]`},
		},
		{
			name: "nested values",
			values: map[string][]string{
				"address.street": {"123 Main St"},
			},
			contains: []string{`"address":{`, `"street":"123 Main St"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := valuesToJSON(tt.values)
			if err != nil {
				t.Fatalf("valuesToJSON failed: %v", err)
			}

			resultStr := string(result)
			for _, expected := range tt.contains {
				if !strings.Contains(resultStr, expected) {
					t.Errorf("expected result to contain %q, got %s", expected, resultStr)
				}
			}
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"null", nil, "null"},
		{"true", true, "true"},
		{"false", false, "false"},
		{"int", int64(42), "42"},
		{"float", float64(3.14), "3.14"},
		{"string", "hello", `"hello"`},
		{"array", []any{"a", "b"}, `["a","b"]`},
		{"object", map[string]any{"key": "value"}, `{"key":"value"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := marshalJSON(tt.input)
			if err != nil {
				t.Fatalf("marshalJSON failed: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(result))
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer

	// Test nested structure
	input := map[string]any{
		"name": "test",
		"count": int64(5),
		"items": []any{"a", "b"},
	}

	err := writeJSON(&buf, input)
	if err != nil {
		t.Fatalf("writeJSON failed: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, `"name":"test"`) {
		t.Error("expected name field in output")
	}
}

func TestWriteJSON_UnsupportedType(t *testing.T) {
	var buf bytes.Buffer

	// Unsupported type should error
	err := writeJSON(&buf, struct{ Name string }{Name: "test"})
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestXMLMarshaler_ContentType(t *testing.T) {
	m := &XMLMarshaler{}
	ct := m.ContentType(nil)

	if ct != "application/xml" {
		t.Errorf("expected application/xml, got %s", ct)
	}
}

func TestXMLMarshaler_Marshal(t *testing.T) {
	m := &XMLMarshaler{}

	type TestStruct struct {
		Name string `xml:"name"`
		Age  int    `xml:"age"`
	}

	input := TestStruct{Name: "John", Age: 30}
	result, err := m.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if !strings.Contains(string(result), "<name>John</name>") {
		t.Error("expected name element in XML output")
	}
}

func TestXMLMarshaler_MarshalIndent(t *testing.T) {
	m := &XMLMarshaler{Indent: "  "}

	type TestStruct struct {
		Name string `xml:"name"`
	}

	input := TestStruct{Name: "Test"}
	result, err := m.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if !strings.Contains(string(result), "\n") {
		t.Error("expected indented XML output")
	}
}

func TestXMLMarshaler_Unmarshal(t *testing.T) {
	m := &XMLMarshaler{}

	type TestStruct struct {
		Name string `xml:"name"`
	}

	input := []byte(`<TestStruct><name>John</name></TestStruct>`)
	var output TestStruct

	err := m.Unmarshal(input, &output)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if output.Name != "John" {
		t.Errorf("expected name John, got %s", output.Name)
	}
}

func TestBinaryMarshaler_ContentType(t *testing.T) {
	m := &BinaryMarshaler{}
	ct := m.ContentType(nil)

	if ct != "application/octet-stream" {
		t.Errorf("expected application/octet-stream, got %s", ct)
	}
}

func TestBinaryMarshaler_MarshalBytes(t *testing.T) {
	m := &BinaryMarshaler{}

	input := []byte("raw binary data")
	result, err := m.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if string(result) != string(input) {
		t.Errorf("expected %s, got %s", string(input), string(result))
	}
}

func TestBinaryMarshaler_UnsupportedType(t *testing.T) {
	m := &BinaryMarshaler{}

	_, err := m.Marshal("not bytes or proto")
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestMultipartMarshaler_ContentType(t *testing.T) {
	m := &MultipartMarshaler{}
	ct := m.ContentType(nil)

	if ct != "multipart/form-data" {
		t.Errorf("expected multipart/form-data, got %s", ct)
	}
}

func TestMultipartMarshaler_Unmarshal_Error(t *testing.T) {
	m := &MultipartMarshaler{}

	// Direct Unmarshal should error
	err := m.Unmarshal([]byte("test"), nil)
	if err == nil {
		t.Error("expected error for direct Unmarshal")
	}
}

func TestDetectBoundary(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CRLF line ending",
			input:    "--boundary123\r\nContent-Type: text/plain\r\n",
			expected: "boundary123",
		},
		{
			name:     "LF line ending",
			input:    "--boundary456\nContent-Type: text/plain\n",
			expected: "boundary456",
		},
		{
			name:     "no boundary",
			input:    "just some data",
			expected: "",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectBoundary([]byte(tt.input))
			if result != tt.expected {
				t.Errorf("detectBoundary() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTextMarshaler_ContentType(t *testing.T) {
	m := &TextMarshaler{}
	ct := m.ContentType(nil)

	if ct != "text/plain" {
		t.Errorf("expected text/plain, got %s", ct)
	}
}

func TestTextMarshaler_MarshalString(t *testing.T) {
	m := &TextMarshaler{}

	result, err := m.Marshal("hello world")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if string(result) != "hello world" {
		t.Errorf("expected 'hello world', got %s", string(result))
	}
}

func TestTextMarshaler_UnsupportedType(t *testing.T) {
	m := &TextMarshaler{}

	_, err := m.Marshal(123)
	if err == nil {
		t.Error("expected error for non-string, non-proto type")
	}
}

func TestWithFormURLEncodedSupport(t *testing.T) {
	cfg := newServerConfig()
	opt := WithFormURLEncodedSupport()
	opt(cfg)

	if _, ok := cfg.marshalers["application/x-www-form-urlencoded"]; !ok {
		t.Error("expected form marshaler to be registered")
	}
}

func TestWithXMLSupport(t *testing.T) {
	cfg := newServerConfig()
	opt := WithXMLSupport()
	opt(cfg)

	if _, ok := cfg.marshalers["application/xml"]; !ok {
		t.Error("expected XML marshaler to be registered")
	}
}

func TestWithXMLSupportIndented(t *testing.T) {
	cfg := newServerConfig()
	opt := WithXMLSupportIndented("  ")
	opt(cfg)

	marshaler, ok := cfg.marshalers["application/xml"]
	if !ok {
		t.Fatal("expected XML marshaler to be registered")
	}

	xmlMarshaler, ok := marshaler.(*XMLMarshaler)
	if !ok {
		t.Fatal("expected XMLMarshaler type")
	}

	if xmlMarshaler.Indent != "  " {
		t.Errorf("expected indent '  ', got %q", xmlMarshaler.Indent)
	}
}

func TestWithBinarySupport(t *testing.T) {
	cfg := newServerConfig()
	opt := WithBinarySupport()
	opt(cfg)

	if _, ok := cfg.marshalers["application/octet-stream"]; !ok {
		t.Error("expected binary marshaler to be registered")
	}
}

func TestWithMultipartSupport(t *testing.T) {
	cfg := newServerConfig()
	opt := WithMultipartSupport()
	opt(cfg)

	if _, ok := cfg.marshalers["multipart/form-data"]; !ok {
		t.Error("expected multipart marshaler to be registered")
	}
}

func TestWithMultipartSupportWithMaxMemory(t *testing.T) {
	cfg := newServerConfig()
	opt := WithMultipartSupportWithMaxMemory(64 << 20) // 64MB
	opt(cfg)

	marshaler, ok := cfg.marshalers["multipart/form-data"]
	if !ok {
		t.Fatal("expected multipart marshaler to be registered")
	}

	multipartMarshaler, ok := marshaler.(*MultipartMarshaler)
	if !ok {
		t.Fatal("expected MultipartMarshaler type")
	}

	if multipartMarshaler.MaxMemory != 64<<20 {
		t.Errorf("expected MaxMemory 64MB, got %d", multipartMarshaler.MaxMemory)
	}
}

func TestWithTextSupport(t *testing.T) {
	cfg := newServerConfig()
	opt := WithTextSupport()
	opt(cfg)

	if _, ok := cfg.marshalers["text/plain"]; !ok {
		t.Error("expected text marshaler to be registered")
	}
}

func TestWithTextSupportFields(t *testing.T) {
	cfg := newServerConfig()
	opt := WithTextSupportFields("content", "response")
	opt(cfg)

	marshaler, ok := cfg.marshalers["text/plain"]
	if !ok {
		t.Fatal("expected text marshaler to be registered")
	}

	textMarshaler, ok := marshaler.(*TextMarshaler)
	if !ok {
		t.Fatal("expected TextMarshaler type")
	}

	if textMarshaler.InputField != "content" {
		t.Errorf("expected InputField 'content', got %q", textMarshaler.InputField)
	}

	if textMarshaler.OutputField != "response" {
		t.Errorf("expected OutputField 'response', got %q", textMarshaler.OutputField)
	}
}

func TestBuildMarshalerOptions(t *testing.T) {
	cfg := newServerConfig()

	// Add JSON options
	cfg.jsonOptions = &JSONOptions{
		UseProtoNames:   true,
		EmitUnpopulated: true,
	}

	// Add custom marshaler
	cfg.marshalers["application/xml"] = &XMLMarshaler{}

	opts := buildMarshalerOptions(cfg)

	// Should have options for JSON and XML
	if len(opts) < 2 {
		t.Errorf("expected at least 2 options, got %d", len(opts))
	}
}

func TestBuildMarshalerOptions_NoOptions(t *testing.T) {
	cfg := newServerConfig()

	opts := buildMarshalerOptions(cfg)

	// With no custom options, should have no options
	if len(opts) != 0 {
		t.Errorf("expected 0 options for empty config, got %d", len(opts))
	}
}
