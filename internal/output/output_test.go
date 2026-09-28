package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type sampleVM struct {
	ID         string `json:"id"`
	NameLabel  string `json:"name_label"`
	PowerState string `json:"power_state"`
	Memory     struct {
		Size int64 `json:"size"`
	} `json:"memory"`
}

func sampleData() []sampleVM {
	return []sampleVM{
		{ID: "id-1", NameLabel: "web-01", PowerState: "Running"},
		{ID: "id-2", NameLabel: "db-01", PowerState: "Halted"},
	}
}

func TestParseFormat(t *testing.T) {
	for _, valid := range Formats {
		if _, err := ParseFormat(valid); err != nil {
			t.Errorf("ParseFormat(%q): unexpected error %v", valid, err)
		}
	}
	if _, err := ParseFormat("xml"); err == nil {
		t.Error("ParseFormat(xml): expected an error")
	}
}

func TestValidateQuery(t *testing.T) {
	if err := ValidateQuery(""); err != nil {
		t.Errorf("ValidateQuery(\"\"): unexpected error %v", err)
	}
	if err := ValidateQuery("[].name_label"); err != nil {
		t.Errorf("ValidateQuery(valid): unexpected error %v", err)
	}
	if err := ValidateQuery("bad["); err == nil {
		t.Error("ValidateQuery(invalid): expected an error")
	}
}

func TestQueryEmptyReturnsNothing(t *testing.T) {
	result, err := Query("", sampleData())
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}
}

func TestQueryProjectsFields(t *testing.T) {
	result, err := Query("[].name_label", sampleData())
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	var names []string
	for _, item := range result.Value.([]any) {
		names = append(names, item.(string))
	}
	if len(names) != 2 || names[0] != "web-01" || names[1] != "db-01" {
		t.Fatalf("unexpected projection: %#v", names)
	}
}

func TestQueryFilter(t *testing.T) {
	result, err := Query("[?power_state==`Running`].name_label", sampleData())
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	names, ok := result.Value.([]any)
	if !ok || len(names) != 1 || names[0] != "web-01" {
		t.Fatalf("unexpected filter result: %#v", result.Value)
	}
}

func TestRenderTable(t *testing.T) {
	var buf bytes.Buffer
	err := RenderTable(&buf, []string{"ID", "NAME"}, [][]string{{"a", "web-01"}, {"b", "db-01"}})
	if err != nil {
		t.Fatalf("RenderTable: %v", err)
	}
	out := buf.String()
	for _, expected := range []string{"ID", "NAME", "a", "web-01", "db-01"} {
		if !strings.Contains(out, expected) {
			t.Errorf("table output missing %q:\n%s", expected, out)
		}
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines (header, separator, 2 rows), got %d:\n%s", len(lines), out)
	}
	// The second column must start at the same offset in the header and in
	// every data row.
	col1 := strings.Index(lines[0], "NAME")
	if col1 < 0 {
		t.Fatalf("header missing NAME:\n%s", out)
	}
	if got := strings.Index(lines[2], "web-01"); got != col1 {
		t.Errorf("row 1 misaligned: header=%d row=%d\n%s", col1, got, out)
	}
	if got := strings.Index(lines[3], "db-01"); got != col1 {
		t.Errorf("row 2 misaligned: header=%d row=%d\n%s", col1, got, out)
	}
}

func TestRenderTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderTable(&buf, []string{"ID"}, nil); err != nil {
		t.Fatalf("RenderTable(empty): %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no output for an empty table, got %q", buf.String())
	}
}

func TestRenderNoQueryTable(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, FormatTable, Table{
		Headers: []string{"ID", "NAME"},
		Rows:    [][]string{{"a", "web-01"}},
	}, nil, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(buf.String(), "web-01") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestRenderNoQueryJSON(t *testing.T) {
	var buf bytes.Buffer
	raw, err := Normalize(sampleData())
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if err := Render(&buf, FormatJSON, Table{}, raw, nil); err != nil {
		t.Fatalf("Render: %v", err)
	}

	var decoded []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if len(decoded) != 2 || decoded[0]["name_label"] != "web-01" {
		t.Fatalf("unexpected JSON: %s", buf.String())
	}
}

func TestRenderNoQueryYAML(t *testing.T) {
	var buf bytes.Buffer
	raw, err := Normalize(sampleData())
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if err := Render(&buf, FormatYAML, Table{}, raw, nil); err != nil {
		t.Fatalf("Render: %v", err)
	}

	var decoded []map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid YAML: %v\n%s", err, buf.String())
	}
	if len(decoded) != 2 || decoded[0]["name_label"] != "web-01" {
		t.Fatalf("unexpected YAML: %s", buf.String())
	}
}

func TestRenderQueryResultJSON(t *testing.T) {
	var buf bytes.Buffer
	query, err := Query("[].name_label", sampleData())
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if err := Render(&buf, FormatJSON, Table{}, nil, query); err != nil {
		t.Fatalf("Render: %v", err)
	}

	var names []string
	if err := json.Unmarshal(buf.Bytes(), &names); err != nil {
		t.Fatalf("query result is not a JSON array: %v\n%s", err, buf.String())
	}
	if len(names) != 2 {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestRenderQueryResultTable(t *testing.T) {
	var buf bytes.Buffer
	query, err := Query("[].name_label", sampleData())
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if err := Render(&buf, FormatTable, Table{}, nil, query); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "web-01") || !strings.Contains(out, "db-01") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRenderTextFormat(t *testing.T) {
	var buf bytes.Buffer
	raw, err := Normalize(sampleData())
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if err := Render(&buf, FormatText, Table{}, raw, nil); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	for _, expected := range []string{"name_label", "web-01", "power_state"} {
		if !strings.Contains(out, expected) {
			t.Errorf("text output missing %q:\n%s", expected, out)
		}
	}
}

func TestNormalizeSDKLikeStruct(t *testing.T) {
	normalized, err := Normalize(sampleData())
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	list, ok := normalized.([]any)
	if !ok || len(list) != 2 {
		t.Fatalf("unexpected normalized shape: %#v", normalized)
	}
	first, ok := list[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %#v", list[0])
	}
	if first["name_label"] != "web-01" {
		t.Fatalf("unexpected value: %#v", first)
	}
}
