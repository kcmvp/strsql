package main

import (
	"log"

	"github.com/kcmvp/strsql/internal/generator"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "strsql",
	Short: "strsql is a type-safe SQL builder and ORM generator",
	Long: `strsql generates strictly typed SQL mapping code for your Go structs.
It parses your code, extracts struct tags, and generates type-safe builders.`,
}

var tagName string

var genCmd = &cobra.Command{
	Use:   "gen [dir]",
	Short: "Generate schema mapping code for structs in the specified directory",
	Long: `Parse Go files in the given directory (default is current directory), 
find all structs, and generate schema_gen.go which contains type-safe mappings.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		models, pkgName, err := generator.ParseModels(dir, tagName)
		if err != nil {
			log.Fatalf("Parse failed: %v", err)
		}

		if len(models) == 0 {
			log.Println("No suitable models found")
			return
		}

		if err := generator.GenerateSchema(dir, pkgName, models); err != nil {
			log.Fatalf("Failed to generate code: %v", err)
		}

		log.Printf("Successfully generated strsql_gen.go in %s\n", dir)
	},
}

func init() {
	genCmd.Flags().StringVarP(&tagName, "tag", "t", "db", "The struct tag to use for column names")
	rootCmd.AddCommand(genCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
