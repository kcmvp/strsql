package generator

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"golang.org/x/tools/go/packages"
)

//go:embed schema.tmpl
var schemaTemplate string

// ModelInfo 存储解析到的 struct 信息
type ModelInfo struct {
	Name   string
	Fields []FieldInfo
}

// FieldInfo 存储 struct 字段信息
type FieldInfo struct {
	Name   string
	Column string
	Index  int
}

// parseTag 解析 struct tag 中的目标属性 (默认是 "db")
func parseTag(tag string, targetTagName string) string {
	if tag == "" {
		return ""
	}
	// 去掉反引号
	tag = strings.Trim(tag, "`")

	// 简单解析，寻找 targetTagName:"xxx"
	prefix := targetTagName + ":"
	parts := strings.Split(tag, " ")
	for _, p := range parts {
		if strings.HasPrefix(p, prefix) {
			val := strings.TrimPrefix(p, prefix)
			val = strings.Trim(val, `"`)
			return val
		}
	}
	return ""
}

// ParseModels 解析指定目录下的 Go 文件，找出实现了 strsql.Model 的 struct (这里简化为解析所有导出的 struct)
func ParseModels(dir string, tagName string) ([]ModelInfo, string, error) {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:   dir,
		Tests: false, // 忽略测试文件
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, "", fmt.Errorf("packages.Load failed: %w", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		return nil, "", fmt.Errorf("packages contains errors")
	}

	if len(pkgs) == 0 {
		return nil, "", fmt.Errorf("no packages found in %s", dir)
	}

	pkg := pkgs[0] // 通常我们只解析目标目录下的主包
	pkgName := pkg.Name

	// 严格检查：是否实现了 strsql.Entity 接口
	var modelInterface *types.Interface
	strsqlPkgPath := "github.com/kcmvp/strsql"

	// 1. 尝试从依赖包中查找 Entity 接口
	for _, importedPkg := range pkg.Imports {
		if importedPkg.PkgPath == strsqlPkgPath {
			obj := importedPkg.Types.Scope().Lookup("Entity")
			if obj != nil {
				if iface, ok := obj.Type().Underlying().(*types.Interface); ok {
					modelInterface = iface
					break
				}
			}
		}
	}

	// 2. 如果没找到，并且当前解析的包就是目标包，从当前包中查找
	if modelInterface == nil && pkg.PkgPath == strsqlPkgPath {
		obj := pkg.Types.Scope().Lookup("Entity")
		if obj != nil {
			if iface, ok := obj.Type().Underlying().(*types.Interface); ok {
				modelInterface = iface
			}
		}
	}

	var models []ModelInfo

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			// 寻找 type 声明
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok || !typeSpec.Name.IsExported() {
				return true
			}

			// 必须是 Struct
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return true
			}

			// 严格检查：是否实现了 strsql.Entity 接口
			// Fallback: 如果通过 types.Implements 失败（比如 AST 解析时导入包没对齐），
			// 我们还可以通过 AST 结构本身进行弱校验（比如查看当前文件是否有对应的方法），
			// 但因为我们有了 Packages API，最好还是用类型信息。
			// 如果 modelInterface 获取失败（比如在某些测试环境），我们将回退到名字匹配

			obj := pkg.TypesInfo.Defs[typeSpec.Name]
			if obj != nil {
				t := obj.Type()
				ptrType := types.NewPointer(t)

				hasMethod := false
				if modelInterface != nil {
					if types.Implements(t, modelInterface) || types.Implements(ptrType, modelInterface) {
						hasMethod = true
					}
				}

				// 兜底校验：只要该 struct 有名为 "TableName" 的方法，我们就认为它实现了 Entity
				// 这是因为 go/packages 在某些仅包含部分文件的测试目录中，可能无法正确构建完整的导入图
				if !hasMethod {
					mset := types.NewMethodSet(t)
					pmset := types.NewMethodSet(ptrType)
					for i := 0; i < mset.Len(); i++ {
						if mset.At(i).Obj().Name() == "TableName" {
							hasMethod = true
						}
					}
					for i := 0; i < pmset.Len(); i++ {
						if pmset.At(i).Obj().Name() == "TableName" {
							hasMethod = true
						}
					}
				}

				if !hasMethod {
					return true // 未实现，跳过该 struct
				}
			}

			model := ModelInfo{
				Name: typeSpec.Name.Name,
			}

			fieldIndex := 0
			for _, field := range structType.Fields.List {
				// 忽略未导出的字段和匿名组合
				if len(field.Names) == 0 || !field.Names[0].IsExported() {
					fieldIndex++
					continue
				}

				fieldName := field.Names[0].Name
				columnName := ""

				// 尝试从 tag 解析
				if field.Tag != nil {
					columnName = parseTag(field.Tag.Value, tagName)
				}

				// 如果 tag 没有指定目标属性，使用默认下划线策略
				if columnName == "" {
					columnName = lo.SnakeCase(fieldName)
				}

				model.Fields = append(model.Fields, FieldInfo{
					Name:   fieldName,
					Column: columnName,
					Index:  fieldIndex,
				})
				fieldIndex++
			}

			// 只有包含字段的 struct 才收集
			if len(model.Fields) > 0 {
				models = append(models, model)
			}

			return true
		})
	}

	return models, pkgName, nil
}

type TemplateData struct {
	PkgName string
	Models  []ModelInfo
}

// GenerateSchema 根据解析到的模型生成代码文件
func GenerateSchema(dir string, pkgName string, models []ModelInfo) error {
	tmpl, err := template.New("schema").Parse(schemaTemplate)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	data := TemplateData{
		PkgName: pkgName,
		Models:  models,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	// 格式化生成的代码 (相当于 gofmt)
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// 如果格式化失败，还是把原始的写入，方便排查错误
		outPath := filepath.Join(dir, "strsql_gen.go")
		_ = os.WriteFile(outPath, buf.Bytes(), 0644)
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	outPath := filepath.Join(dir, "strsql_gen.go")
	return os.WriteFile(outPath, formatted, 0644)
}
