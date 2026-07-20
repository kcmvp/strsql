package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSchemaUsesSharedEntityTypeCache(t *testing.T) {
	dir := t.TempDir()

	err := GenerateSchema(dir, "models", []ModelInfo{
		{
			Name: "Product",
			Fields: []FieldInfo{
				{Name: "ID", Column: "id", Index: 0},
				{Name: "Price", Column: "price", Index: 1},
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
	if strings.Count(content, "reflect.TypeOf(*new(Product))") > 1 {
		t.Fatalf("generated code still repeats Product reflect.Type lookup:\n%s", content)
	}
	if !strings.Contains(content, `var _ProductType = typeOf[Product]()`) {
		t.Fatalf("generated code missing shared Product type cache:\n%s", content)
	}
	if strings.Contains(content, `.Field(`) {
		t.Fatalf("generated code still uses index-based field lookup:\n%s", content)
	}
	if !strings.Contains(content, `fieldType(_ProductType, "ID")`) {
		t.Fatalf("generated code missing name-based field type lookup for Product.ID:\n%s", content)
	}
	if !strings.Contains(content, `fieldType(_ProductType, "Price")`) {
		t.Fatalf("generated code missing name-based field type lookup for Product.Price:\n%s", content)
	}
}
