package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type Plugin struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type License struct {
	LicenseName string `json:"license_name"`
	LicenseText string `json:"license_text"`
	LicenseFile string `json:"license_file"`
}

type Copyright struct {
	CopyrightText string `json:"copyright_text"`
	CopyrightFile string `json:"copyright_file"`
}

type Config struct {
	License   `yaml:",inline"`
	Copyright `yaml:",inline"`
	Build     []BuildInfo `json:"build"`
}

type Vars struct {
	VarsValues map[string]string `json:"vars_values"`
	VarsFiles  []string          `json:"vars_files"`
}

type BuildInfo struct {
	License          `yaml:",inline"`
	Copyright        `yaml:",inline"`
	Kind             string `json:"kind"`
	Spec             string `json:"spec"`
	Vars             `yaml:",inline"`
	Output           string `json:"output"`
	Language         string `json:"language"`
	Templates        string `json:"templates"`
	UserAgent        string `json:"user_agent"`
	Version          string `json:"version"`
	APIServicePrefix string `json:"api_service_prefix"`
	APIServiceSuffix string `json:"api_service_suffix"`
}

type Params struct {
	Kind string
	Name string
}

type Operation struct {
	RequestBody    *Params `json:"request_body"`
	ResponseBody   *Params
	QueryParams    []Params
	RequestHeaders []Params
	PathParams     []Params
}

type Service struct {
	Operations []Operation
}

func main() {
	fmt.Println(SplitReverse("example.com", ".", nil))
	fmt.Println(SplitReverse("example.co.mz", ".", func(s string) string {
		return string(unicode.ToUpper(rune(s[0]))) + s[1:]
	}))

	pathDirs := strings.Split(os.Getenv("PATH"), ":")

	params := make(map[string][]Plugin)

	for _, dir := range pathDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			base := filepath.Base(path)

			prefix := "openapi-gen-"
			if strings.HasPrefix(base, prefix) {
				name := strings.TrimLeft(base, prefix)
				parts := strings.SplitN(name, "-", 2)

				if _, ok := params[parts[0]]; !ok {
					params[parts[0]] = make([]Plugin, 0)
				}

				plugin := Plugin{
					Name: parts[1],
					Kind: parts[0],
					Path: path,
				}

				params[parts[0]] = append(params[parts[0]], plugin)
			}

			return nil
		})
		if err != nil {
			fmt.Printf("walk error [%v]\n", err)
		}
	}

	if err := cmdRoot(params).Execute(); err != nil {
		panic(err)
	}
}

func cmdBuild(plugins map[string][]Plugin) *cobra.Command {
	newCmd := &cobra.Command{
		Use: "build",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	newCmd.Flags().StringP("kind", "k", "", "kind")
	newCmd.Flags().StringP("language", "l", "", "language")
	newCmd.Flags().StringP("spec", "s", "", "spec")
	newCmd.Flags().StringArrayP("var", "R", []string{}, "var")
	newCmd.Flags().StringP("vars-file", "F", "", "var file")
	newCmd.Flags().StringP("templates", "T", "", "templates")
	newCmd.Flags().StringP("output", "o", "", "Output dir")
	newCmd.Flags().StringP("except", "E", "", "except")
	newCmd.Flags().StringP("only", "O", "", "Only")
	newCmd.Flags().StringP("skip", "S", "", "skip")
	newCmd.Flags().StringP("from-file", "f", "", "from file")

	return newCmd
}

func cmdRoot(kind map[string][]Plugin) *cobra.Command {
	newCmd := &cobra.Command{
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := openapi3.NewLoader().LoadFromFile(args[0])
			if err != nil {
				return err
			}

			if err := doc.Validate(context.Background()); err != nil {
				return err
			}

			tags := make(map[string]struct{})

			if len(doc.Tags) > 0 {
				tags[doc.Tags[0].Name] = struct{}{}
			}

			for path, value := range doc.Paths {
				for method, op := range value.Operations() {
					if ext, ok := op.Extensions["x-operation-name"]; ok {
						fmt.Println("found the x-operation-name extentions")

						raw, ok := ext.(json.RawMessage)
						if !ok {
							panic("no json.RawMessage")
						}

						var opName string
						if err := json.Unmarshal([]byte(raw), &opName); err != nil {
							panic(err)
						}

						fmt.Println("Extension:", opName)
					}

					if len(op.Tags) > 0 {
						tags[op.Tags[0]] = struct{}{}
					}

					fmt.Println(toKebab(op.OperationID), "->", method, " ", path)
					fmt.Println(toSnake(op.OperationID), "->", method, " ", path)
					fmt.Println(toCamel(op.OperationID), "->", method, " ", path)
					fmt.Println(toPascal(op.OperationID), "->", method, " ", path)
					fmt.Println(toFlag(op.OperationID), "->", method, " ", path)
					fmt.Println(toEnum(op.OperationID), "->", method, " ", path)

					for ext, v := range op.Extensions {
						fmt.Println(ext, v)
					}

					for _, tag := range op.Tags {
						fmt.Println(tag)
					}

					fmt.Println()
				}
			}

			for tag := range tags {
				fmt.Println(toPascal(tag))
			}

			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:  "list",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, e := range kind {
				for _, lang := range e {

					fmt.Println(lang)
				}
			}

			return nil
		},
	}

	listCmd.Flags().Bool("cli", false, "list cli generators")
	listCmd.Flags().Bool("lib", false, "list lib templates")
	listCmd.Flags().Bool("all", false, "list all generators")

	newCmd.AddCommand(listCmd)

	newCmd.AddCommand(cmdBuild(kind))

	return newCmd
}

func toKebab(s string) string {
	buf := new(strings.Builder)

	for i, c := range s {
		if i == 0 {
			buf.WriteRune(c)
			continue
		}

		if unicode.IsUpper(c) && unicode.IsLower(rune(s[i-1])) {
			buf.WriteRune('-')
		} else if unicode.IsDigit(c) && !unicode.IsDigit(rune(s[i-1])) {
			buf.WriteRune('-')
		} else if !unicode.IsDigit(c) && unicode.IsDigit(rune(s[i-1])) {
			buf.WriteRune('-')
		} else if !unicode.IsDigit(c) && !unicode.IsLetter(c) && c != '-' {
			buf.WriteRune('-')
			continue
		}

		buf.WriteRune(c)
	}

	return strings.ToLower(buf.String())
}

func toSnake(s string) string {
	buf := new(strings.Builder)

	for i, c := range s {
		if i == 0 {
			buf.WriteRune(unicode.ToLower(c))
			continue
		}

		if unicode.IsUpper(c) && unicode.IsLower(rune(s[i-1])) {
			buf.WriteRune('_')
		} else if unicode.IsDigit(c) && !unicode.IsDigit(rune(s[i-1])) {
			buf.WriteRune('_')
		} else if !unicode.IsDigit(c) && unicode.IsDigit(rune(s[i-1])) {
			buf.WriteRune('_')
		} else if !unicode.IsDigit(c) && !unicode.IsLetter(c) && c != '_' {
			buf.WriteRune('_')
			continue
		}

		buf.WriteRune(c)
	}

	return strings.ToLower(buf.String())
}

func toFlag(s string) string {
	return "--" + toKebab(s)
}

func toCamel(s string) string {
	buf := new(strings.Builder)

	var changeCase bool
	for i, c := range s {
		if i == 0 {
			buf.WriteRune(c)
			continue
		}

		if !unicode.IsDigit(c) && !unicode.IsLetter(c) {
			changeCase = true
			continue
		}

		if changeCase {
			buf.WriteRune(unicode.ToUpper(c))
			changeCase = false
		} else {
			buf.WriteRune(c)
		}
	}

	return buf.String()
}

func toPascal(s string) string {
	buf := new(strings.Builder)

	var changeCase bool
	for i, c := range s {
		if i == 0 {
			buf.WriteRune(unicode.ToUpper(c))
			continue
		}

		if !unicode.IsDigit(c) && !unicode.IsLetter(c) {
			changeCase = true
			continue
		}

		if changeCase {
			buf.WriteRune(unicode.ToUpper(c))
			changeCase = false
		} else {
			buf.WriteRune(c)
		}
	}

	return buf.String()
}

func SplitReverse(s string, sep string, fn func(string) string) string {
	parts := strings.Split(s, sep)

	reversed := make([]string, 0)
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if fn != nil {
			part = fn(parts[i])
		}

		reversed = append(reversed, part)
	}

	return strings.Join(reversed, ".")
}

func toEnum(s string) string {
	return strings.ToUpper(toSnake(s))
}
