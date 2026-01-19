// Package grpckit provides custom marshalers for various content types.
package grpckit

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// bufferPool provides reusable byte buffers to reduce GC pressure.
// Buffers are reset before being returned to the pool.
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// getBuffer retrieves a buffer from the pool.
func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

// putBuffer returns a buffer to the pool after resetting it.
// Very large buffers (>64KB) are not returned to prevent memory leaks.
func putBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 64*1024 {
		// Don't pool very large buffers to prevent memory issues
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}

// titleCase capitalizes the first letter of a string.
// This is a simple replacement for the deprecated strings.Title.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// buildMarshalerOptions converts the marshaler configuration to ServeMuxOptions.
func buildMarshalerOptions(cfg *serverConfig) []runtime.ServeMuxOption {
	var opts []runtime.ServeMuxOption

	// Apply JSON options if set
	if cfg.jsonOptions != nil {
		jsonMarshaler := &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   cfg.jsonOptions.UseProtoNames,
				EmitUnpopulated: cfg.jsonOptions.EmitUnpopulated,
				Indent:          cfg.jsonOptions.Indent,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: cfg.jsonOptions.DiscardUnknown,
			},
		}
		opts = append(opts, runtime.WithMarshalerOption("application/json", jsonMarshaler))
		opts = append(opts, runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonMarshaler))
	}

	// Apply custom marshalers
	for mimeType, marshaler := range cfg.marshalers {
		opts = append(opts, runtime.WithMarshalerOption(mimeType, marshaler))
	}

	// Append any additional gateway options
	opts = append(opts, cfg.gatewayOptions...)

	return opts
}

// ============================================================================
// Form URL-Encoded Marshaler
// ============================================================================

// FormMarshaler handles application/x-www-form-urlencoded content.
// It parses form data into proto messages using proto field names.
// For responses, it falls back to JSON output since forms are typically input-only.
//
// Field mapping:
//   - Uses proto field names (snake_case by default)
//   - Supports nested fields via dot notation: address.street=123
//   - Supports repeated fields via multiple values: tags=a&tags=b
//
// Example request:
//
//	POST /api/v1/users
//	Content-Type: application/x-www-form-urlencoded
//
//	name=John&email=john@example.com&age=30
type FormMarshaler struct {
	runtime.JSONPb // Fallback for output (forms are typically input-only)
}

// ContentType returns the MIME type for form data.
func (f *FormMarshaler) ContentType(_ interface{}) string {
	return "application/x-www-form-urlencoded"
}

// Unmarshal parses form-encoded data into a proto message.
func (f *FormMarshaler) Unmarshal(data []byte, v interface{}) error {
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse form data: %w", err)
	}

	return populateFromValues(values, v)
}

// NewDecoder returns a decoder for streaming form data.
func (f *FormMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return &formDecoder{r: r, marshaler: f}
}

// formDecoder implements runtime.Decoder for form data.
type formDecoder struct {
	r         io.Reader
	marshaler *FormMarshaler
}

func (d *formDecoder) Decode(v interface{}) error {
	data, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}
	return d.marshaler.Unmarshal(data, v)
}

// populateFromValues populates a proto message from URL values.
func populateFromValues(values url.Values, v interface{}) error {
	// Convert to JSON then unmarshal via JSONPb for proper proto handling
	jsonData, err := valuesToJSON(values)
	if err != nil {
		return err
	}

	jsonMarshaler := &runtime.JSONPb{
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}
	return jsonMarshaler.Unmarshal(jsonData, v)
}

// valuesToJSON converts URL values to JSON bytes.
// Supports nested fields via dot notation and repeated fields.
func valuesToJSON(values url.Values) ([]byte, error) {
	result := make(map[string]interface{})

	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}

		// Handle nested keys (e.g., "address.street" -> {"address": {"street": ...}})
		parts := strings.Split(key, ".")
		current := result

		for i, part := range parts {
			if i == len(parts)-1 {
				// Last part - set the value
				if len(vals) == 1 {
					current[part] = inferType(vals[0])
				} else {
					// Multiple values = array
					arr := make([]interface{}, len(vals))
					for j, v := range vals {
						arr[j] = inferType(v)
					}
					current[part] = arr
				}
			} else {
				// Intermediate part - create nested map if needed
				if _, ok := current[part]; !ok {
					current[part] = make(map[string]interface{})
				}
				if nested, ok := current[part].(map[string]interface{}); ok {
					current = nested
				}
			}
		}
	}

	return marshalJSON(result)
}

// inferType attempts to infer the Go type from a string value.
func inferType(s string) interface{} {
	// Try boolean
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// Try integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		// Only use float if it has a decimal point
		if strings.Contains(s, ".") {
			return f
		}
	}

	// Default to string
	return s
}

// marshalJSON is a simple JSON marshaler to avoid import cycles.
// Uses buffer pooling to reduce GC pressure.
func marshalJSON(v interface{}) ([]byte, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	if err := writeJSON(buf, v); err != nil {
		return nil, err
	}

	// Must copy since buffer will be reused
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

func writeJSON(w io.Writer, v interface{}) error {
	switch val := v.(type) {
	case nil:
		_, err := w.Write([]byte("null"))
		return err
	case bool:
		if val {
			_, err := w.Write([]byte("true"))
			return err
		}
		_, err := w.Write([]byte("false"))
		return err
	case int64:
		_, err := fmt.Fprintf(w, "%d", val)
		return err
	case float64:
		_, err := fmt.Fprintf(w, "%g", val)
		return err
	case string:
		_, err := fmt.Fprintf(w, "%q", val)
		return err
	case []interface{}:
		if _, err := w.Write([]byte("[")); err != nil {
			return err
		}
		for i, item := range val {
			if i > 0 {
				if _, err := w.Write([]byte(",")); err != nil {
					return err
				}
			}
			if err := writeJSON(w, item); err != nil {
				return err
			}
		}
		_, err := w.Write([]byte("]"))
		return err
	case map[string]interface{}:
		if _, err := w.Write([]byte("{")); err != nil {
			return err
		}
		first := true
		for k, item := range val {
			if !first {
				if _, err := w.Write([]byte(",")); err != nil {
					return err
				}
			}
			first = false
			if _, err := fmt.Fprintf(w, "%q:", k); err != nil {
				return err
			}
			if err := writeJSON(w, item); err != nil {
				return err
			}
		}
		_, err := w.Write([]byte("}"))
		return err
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// ============================================================================
// XML Marshaler
// ============================================================================

// XMLMarshaler handles application/xml content.
// It uses Go's encoding/xml package for serialization.
//
// Example request:
//
//	POST /api/v1/items
//	Content-Type: application/xml
//
//	<CreateItemRequest>
//	  <name>Widget</name>
//	  <price>9.99</price>
//	</CreateItemRequest>
//
// Note: Proto messages need xml struct tags for proper field mapping.
// By default, uses the proto field names.
type XMLMarshaler struct {
	// Indent sets indentation for pretty printing (empty = compact)
	Indent string
}

// ContentType returns the MIME type for XML.
func (x *XMLMarshaler) ContentType(_ interface{}) string {
	return "application/xml"
}

// Marshal serializes a proto message to XML bytes.
func (x *XMLMarshaler) Marshal(v interface{}) ([]byte, error) {
	if x.Indent != "" {
		return xml.MarshalIndent(v, "", x.Indent)
	}
	return xml.Marshal(v)
}

// Unmarshal parses XML bytes into a proto message.
func (x *XMLMarshaler) Unmarshal(data []byte, v interface{}) error {
	return xml.Unmarshal(data, v)
}

// NewDecoder returns an XML decoder for streaming.
func (x *XMLMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return xml.NewDecoder(r)
}

// NewEncoder returns an XML encoder for streaming.
func (x *XMLMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	enc := xml.NewEncoder(w)
	if x.Indent != "" {
		enc.Indent("", x.Indent)
	}
	return enc
}

// ============================================================================
// Binary/Octet-Stream Marshaler
// ============================================================================

// BinaryMarshaler handles application/octet-stream content for raw binary data.
// It works with google.api.HttpBody proto messages for raw byte handling,
// or any proto message that has a bytes field.
//
// For messages with a 'data' bytes field, it reads/writes the raw bytes directly.
// For other messages, it falls back to proto binary encoding.
//
// Example proto definition for binary endpoints:
//
//	import "google/api/httpbody.proto";
//
//	rpc DownloadFile(DownloadRequest) returns (google.api.HttpBody) {
//	  option (google.api.http) = { get: "/api/v1/files/{id}" };
//	}
type BinaryMarshaler struct {
	// FallbackMarshaler is used for non-binary responses (default: JSONPb)
	FallbackMarshaler runtime.Marshaler
}

// ContentType returns the MIME type for binary data.
func (b *BinaryMarshaler) ContentType(_ interface{}) string {
	return "application/octet-stream"
}

// Marshal serializes a message to binary.
func (b *BinaryMarshaler) Marshal(v interface{}) ([]byte, error) {
	// Check if it's a proto message with Data field
	if msg, ok := v.(proto.Message); ok {
		// Try to get bytes from a 'data' field via reflection
		rv := reflect.ValueOf(msg).Elem()
		if dataField := rv.FieldByName("Data"); dataField.IsValid() && dataField.Kind() == reflect.Slice && dataField.Type().Elem().Kind() == reflect.Uint8 {
			return dataField.Bytes(), nil
		}
		// Fall back to proto binary encoding
		return proto.Marshal(msg)
	}

	// Try type assertion for bytes
	if data, ok := v.([]byte); ok {
		return data, nil
	}

	return nil, errors.New("binary marshaler: unsupported type")
}

// Unmarshal parses binary data into a message.
func (b *BinaryMarshaler) Unmarshal(data []byte, v interface{}) error {
	// Check if it's a proto message
	if msg, ok := v.(proto.Message); ok {
		// Try to set bytes to a 'data' field via reflection
		rv := reflect.ValueOf(msg).Elem()
		if dataField := rv.FieldByName("Data"); dataField.IsValid() && dataField.CanSet() && dataField.Kind() == reflect.Slice && dataField.Type().Elem().Kind() == reflect.Uint8 {
			dataField.SetBytes(data)
			return nil
		}
		// Fall back to proto binary decoding
		return proto.Unmarshal(data, msg)
	}

	return errors.New("binary marshaler: unsupported type")
}

// NewDecoder returns a decoder for binary data.
func (b *BinaryMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return &binaryDecoder{r: r, marshaler: b}
}

// NewEncoder returns an encoder for binary data.
func (b *BinaryMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return &binaryEncoder{w: w, marshaler: b}
}

type binaryDecoder struct {
	r         io.Reader
	marshaler *BinaryMarshaler
}

func (d *binaryDecoder) Decode(v interface{}) error {
	data, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}
	return d.marshaler.Unmarshal(data, v)
}

type binaryEncoder struct {
	w         io.Writer
	marshaler *BinaryMarshaler
}

func (e *binaryEncoder) Encode(v interface{}) error {
	data, err := e.marshaler.Marshal(v)
	if err != nil {
		return err
	}
	_, err = e.w.Write(data)
	return err
}

// ============================================================================
// Multipart Form Data Marshaler
// ============================================================================

// MultipartMarshaler handles multipart/form-data content for file uploads.
// It parses multipart forms and maps fields/files to proto message fields.
//
// Field mapping:
//   - Form fields map to proto fields by name
//   - File uploads are stored in bytes fields with "_data" suffix
//   - File metadata (filename, content-type) stored in corresponding string fields
//
// Example proto definition:
//
//	message UploadRequest {
//	  string name = 1;
//	  bytes file_data = 2;      // File contents
//	  string file_name = 3;     // Original filename
//	  string file_type = 4;     // Content-Type
//	}
type MultipartMarshaler struct {
	// MaxMemory limits memory usage for parsing (default: 32MB)
	MaxMemory int64

	// Fallback for non-multipart responses
	runtime.JSONPb
}

// ContentType returns the MIME type for multipart form data.
func (m *MultipartMarshaler) ContentType(_ interface{}) string {
	return "multipart/form-data"
}

// Unmarshal is not directly used for multipart; use NewDecoder instead.
func (m *MultipartMarshaler) Unmarshal(data []byte, v interface{}) error {
	return errors.New("multipart marshaler: use NewDecoder for multipart data")
}

// NewDecoder returns a decoder for multipart form data.
func (m *MultipartMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	maxMem := m.MaxMemory
	if maxMem == 0 {
		maxMem = 32 << 20 // 32MB default
	}
	return &multipartDecoder{r: r, maxMemory: maxMem}
}

type multipartDecoder struct {
	r         io.Reader
	maxMemory int64
}

func (d *multipartDecoder) Decode(v interface{}) error {
	// Read all data to get boundary from content-type
	data, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}

	// Try to detect boundary from data
	boundary := detectBoundary(data)
	if boundary == "" {
		return errors.New("multipart decoder: could not detect boundary")
	}

	reader := multipart.NewReader(bytes.NewReader(data), boundary)

	form, err := reader.ReadForm(d.maxMemory)
	if err != nil {
		return fmt.Errorf("multipart decoder: failed to parse form: %w", err)
	}
	defer func() { _ = form.RemoveAll() }()

	return populateFromMultipart(form, v)
}

// detectBoundary attempts to detect the multipart boundary from the data.
func detectBoundary(data []byte) string {
	// Look for the first line which should be --boundary
	idx := bytes.Index(data, []byte("\r\n"))
	if idx == -1 {
		idx = bytes.Index(data, []byte("\n"))
	}
	if idx == -1 {
		return ""
	}

	line := string(data[:idx])
	if strings.HasPrefix(line, "--") {
		return strings.TrimPrefix(line, "--")
	}
	return ""
}

// populateFromMultipart populates a proto message from multipart form data.
func populateFromMultipart(form *multipart.Form, v interface{}) error {
	result := make(map[string]interface{})

	// Process regular fields
	for key, vals := range form.Value {
		if len(vals) == 1 {
			result[key] = inferType(vals[0])
		} else if len(vals) > 1 {
			arr := make([]interface{}, len(vals))
			for i, v := range vals {
				arr[i] = inferType(v)
			}
			result[key] = arr
		}
	}

	// Process file fields
	for key, files := range form.File {
		if len(files) == 0 {
			continue
		}
		fh := files[0] // Take first file for each field

		f, err := fh.Open()
		if err != nil {
			return fmt.Errorf("multipart: failed to open file %s: %w", key, err)
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("multipart: failed to read file %s: %w", key, err)
		}

		// Store file data as base64 for JSON conversion
		// The key_data field will contain the bytes
		result[key+"_data"] = data
		result[key+"_name"] = fh.Filename

		// Extract content type
		if ct := fh.Header.Get("Content-Type"); ct != "" {
			result[key+"_type"] = ct
		}
	}

	// Convert to JSON then unmarshal
	jsonData, err := marshalJSONWithBytes(result)
	if err != nil {
		return err
	}

	jsonMarshaler := &runtime.JSONPb{
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}
	return jsonMarshaler.Unmarshal(jsonData, v)
}

// marshalJSONWithBytes marshals a map to JSON, encoding []byte as base64.
// Uses buffer pooling to reduce GC pressure.
func marshalJSONWithBytes(v map[string]interface{}) ([]byte, error) {
	// Convert []byte to base64 strings for JSON encoding
	converted := make(map[string]interface{})
	for k, val := range v {
		if data, ok := val.([]byte); ok {
			// Encode bytes as base64 for proto's bytes fields
			converted[k] = data
		} else {
			converted[k] = val
		}
	}

	// Use encoding/json for proper handling with pooled buffer
	buf := getBuffer()
	defer putBuffer(buf)

	if err := writeJSONWithBytes(buf, converted); err != nil {
		return nil, err
	}

	// Must copy since buffer will be reused
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

func writeJSONWithBytes(w io.Writer, v interface{}) error {
	switch val := v.(type) {
	case nil:
		_, err := w.Write([]byte("null"))
		return err
	case bool:
		if val {
			_, err := w.Write([]byte("true"))
			return err
		}
		_, err := w.Write([]byte("false"))
		return err
	case int64:
		_, err := fmt.Fprintf(w, "%d", val)
		return err
	case float64:
		_, err := fmt.Fprintf(w, "%g", val)
		return err
	case string:
		_, err := fmt.Fprintf(w, "%q", val)
		return err
	case []byte:
		// Encode as base64 string for proto bytes fields
		_, err := fmt.Fprintf(w, "%q", string(val))
		return err
	case []interface{}:
		if _, err := w.Write([]byte("[")); err != nil {
			return err
		}
		for i, item := range val {
			if i > 0 {
				if _, err := w.Write([]byte(",")); err != nil {
					return err
				}
			}
			if err := writeJSONWithBytes(w, item); err != nil {
				return err
			}
		}
		_, err := w.Write([]byte("]"))
		return err
	case map[string]interface{}:
		if _, err := w.Write([]byte("{")); err != nil {
			return err
		}
		first := true
		for k, item := range val {
			if !first {
				if _, err := w.Write([]byte(",")); err != nil {
					return err
				}
			}
			first = false
			if _, err := fmt.Fprintf(w, "%q:", k); err != nil {
				return err
			}
			if err := writeJSONWithBytes(w, item); err != nil {
				return err
			}
		}
		_, err := w.Write([]byte("}"))
		return err
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// ============================================================================
// Convenience Option Functions
// ============================================================================

// WithFormURLEncodedSupport enables application/x-www-form-urlencoded request parsing.
// This allows your endpoints to accept HTML form submissions.
//
// Form fields are mapped to proto fields by name using snake_case.
// Nested fields use dot notation: address.street=123
// Repeated fields use multiple values: tags=a&tags=b
//
// Example:
//
//	grpckit.Run(
//	    grpckit.WithGRPCService(...),
//	    grpckit.WithRESTService(...),
//	    grpckit.WithFormURLEncodedSupport(),
//	)
//
// Then send requests like:
//
//	curl -X POST http://localhost:8080/api/v1/users \
//	  -H "Content-Type: application/x-www-form-urlencoded" \
//	  -d "name=John&email=john@example.com"
func WithFormURLEncodedSupport() Option {
	return WithMarshaler("application/x-www-form-urlencoded", &FormMarshaler{})
}

// WithXMLSupport enables application/xml request and response support.
// XML elements are mapped to proto fields by name.
//
// Example:
//
//	grpckit.Run(
//	    grpckit.WithGRPCService(...),
//	    grpckit.WithRESTService(...),
//	    grpckit.WithXMLSupport(),
//	)
//
// Then send requests like:
//
//	curl -X POST http://localhost:8080/api/v1/items \
//	  -H "Content-Type: application/xml" \
//	  -d '<CreateItemRequest><name>Widget</name></CreateItemRequest>'
func WithXMLSupport() Option {
	return WithMarshaler("application/xml", &XMLMarshaler{})
}

// WithXMLSupportIndented enables XML support with pretty-printed output.
// The indent parameter specifies the indentation string (e.g., "  " for 2 spaces).
func WithXMLSupportIndented(indent string) Option {
	return WithMarshaler("application/xml", &XMLMarshaler{Indent: indent})
}

// WithBinarySupport enables application/octet-stream for raw binary data.
// Use this for file downloads or endpoints that work with raw bytes.
//
// Your proto should use bytes fields or google.api.HttpBody for binary data.
//
// Example:
//
//	grpckit.Run(
//	    grpckit.WithGRPCService(...),
//	    grpckit.WithRESTService(...),
//	    grpckit.WithBinarySupport(),
//	)
func WithBinarySupport() Option {
	return WithMarshaler("application/octet-stream", &BinaryMarshaler{})
}

// WithMultipartSupport enables multipart/form-data for file uploads.
// The default max memory for buffering is 32MB.
//
// Proto field mapping for files:
//   - {field}_data: bytes field containing file contents
//   - {field}_name: string field containing original filename
//   - {field}_type: string field containing Content-Type
//
// Example proto:
//
//	message UploadRequest {
//	  string description = 1;
//	  bytes file_data = 2;
//	  string file_name = 3;
//	  string file_type = 4;
//	}
//
// Example usage:
//
//	grpckit.Run(
//	    grpckit.WithGRPCService(...),
//	    grpckit.WithRESTService(...),
//	    grpckit.WithMultipartSupport(),
//	)
//
// Then upload files:
//
//	curl -X POST http://localhost:8080/api/v1/upload \
//	  -F "description=My file" \
//	  -F "file=@document.pdf"
func WithMultipartSupport() Option {
	return WithMultipartSupportWithMaxMemory(32 << 20) // 32MB default
}

// WithMultipartSupportWithMaxMemory enables multipart/form-data with custom memory limit.
// The maxMemory parameter controls how much memory is used for buffering file uploads.
// Files larger than maxMemory are stored in temporary files.
func WithMultipartSupportWithMaxMemory(maxMemory int64) Option {
	return WithMarshaler("multipart/form-data", &MultipartMarshaler{
		MaxMemory: maxMemory,
	})
}

// ============================================================================
// Text Marshaler (Bonus)
// ============================================================================

// TextMarshaler handles text/plain content.
// It maps plain text to a string field in the proto message.
// For responses, it extracts a string field (defaults to "text" or "message").
type TextMarshaler struct {
	// InputField is the proto field name to populate with text input (default: "text")
	InputField string

	// OutputField is the proto field name to use for text output (default: "text", then "message")
	OutputField string
}

// ContentType returns the MIME type for plain text.
func (t *TextMarshaler) ContentType(_ interface{}) string {
	return "text/plain"
}

// Marshal extracts a string field from the message.
func (t *TextMarshaler) Marshal(v interface{}) ([]byte, error) {
	if msg, ok := v.(proto.Message); ok {
		rv := reflect.ValueOf(msg).Elem()

		// Try output field
		outputField := t.OutputField
		if outputField == "" {
			outputField = "text"
		}

		if field := rv.FieldByName(titleCase(outputField)); field.IsValid() && field.Kind() == reflect.String {
			return []byte(field.String()), nil
		}

		// Try "message" as fallback
		if field := rv.FieldByName("Message"); field.IsValid() && field.Kind() == reflect.String {
			return []byte(field.String()), nil
		}
	}

	// Try direct string
	if s, ok := v.(string); ok {
		return []byte(s), nil
	}

	return nil, errors.New("text marshaler: no string field found")
}

// Unmarshal sets the text content to a string field in the message.
func (t *TextMarshaler) Unmarshal(data []byte, v interface{}) error {
	if msg, ok := v.(proto.Message); ok {
		rv := reflect.ValueOf(msg).Elem()

		inputField := t.InputField
		if inputField == "" {
			inputField = "text"
		}

		if field := rv.FieldByName(titleCase(inputField)); field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
			field.SetString(string(data))
			return nil
		}

		// Try "message" as fallback
		if field := rv.FieldByName("Message"); field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
			field.SetString(string(data))
			return nil
		}
	}

	return errors.New("text marshaler: no string field found to set")
}

// NewDecoder returns a decoder for text data.
func (t *TextMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return &textDecoder{r: r, marshaler: t}
}

// NewEncoder returns an encoder for text data.
func (t *TextMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return &textEncoder{w: w, marshaler: t}
}

type textDecoder struct {
	r         io.Reader
	marshaler *TextMarshaler
}

func (d *textDecoder) Decode(v interface{}) error {
	data, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}
	return d.marshaler.Unmarshal(data, v)
}

type textEncoder struct {
	w         io.Writer
	marshaler *TextMarshaler
}

func (e *textEncoder) Encode(v interface{}) error {
	data, err := e.marshaler.Marshal(v)
	if err != nil {
		return err
	}
	_, err = e.w.Write(data)
	return err
}

// WithTextSupport enables text/plain request and response support.
// Text content is mapped to a "text" field in the proto message.
//
// Example:
//
//	grpckit.Run(
//	    grpckit.WithGRPCService(...),
//	    grpckit.WithRESTService(...),
//	    grpckit.WithTextSupport(),
//	)
func WithTextSupport() Option {
	return WithMarshaler("text/plain", &TextMarshaler{})
}

// WithTextSupportFields enables text/plain support with custom field names.
// Use inputField for the proto field to populate on requests,
// and outputField for the field to use on responses.
func WithTextSupportFields(inputField, outputField string) Option {
	return WithMarshaler("text/plain", &TextMarshaler{
		InputField:  inputField,
		OutputField: outputField,
	})
}

// Suppress unused import warning for mime package
var _ = mime.ParseMediaType
