package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSchemaUsesFieldByName(t *testing.T) {
	dir := t.TempDir()

	err := GenerateSchema(dir, "models", []ModelInfo{
		{
			Name: "Product",
			Fields: []FieldInfo{
				{Name: "ID", Column: "id"},
				{Name: "Price", Column: "price"},
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateSchema() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "strsql_gen.go"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(got)
	if strings.Contains(content, ".Field(") {
		t.Fatalf("generated code still uses index-based field lookup:\n%s", content)
	}
	if !strings.Contains(content, `FieldByName("ID")`) {
		t.Fatalf("generated code missing FieldByName lookup for ID:\n%s", content)
	}
	if !strings.Contains(content, `panic("strsql_gen: Product.ID not found")`) {
		t.Fatalf("generated code missing panic guard for Product.ID:\n%s", content)
	}
}
