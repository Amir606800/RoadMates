package v1

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIContract(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(filepath.Join("..", "..", "..", "api", "openapi", "v1.yaml"))
	if err != nil {
		t.Fatalf("could not load OpenAPI contract: %v", err)
	}
	if err = doc.Validate(context.Background()); err != nil {
		t.Fatalf("invalid OpenAPI contract: %v", err)
	}
	for schemaName, dto := range map[string]any{
		"RegisterRequest": RegisterRequest{}, "LoginRequest": LoginRequest{}, "Session": SessionResponse{}, "User": UserResponse{},
		"DriverApplicationRequest": DriverApplicationRequest{}, "DriverApplication": DriverApplicationResponse{}, "AdminApplicationDetail": AdminApplicationDetail{},
		"ReviewApplicationRequest": ReviewApplicationRequest{}, "ApplicationList": ApplicationListResponse{}, "District": DistrictResponse{}, "PublishRideRequest": PublishRideRequest{}, "Ride": RideResponse{},
	} {
		schema := doc.Components.Schemas[schemaName]
		if schema == nil || schema.Value == nil {
			t.Fatalf("schema missing for %s", schemaName)
		}
		var fields func(reflect.Type)
		seen := make(map[string]bool)
		fields = func(typ reflect.Type) {
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				if field.Anonymous {
					fields(field.Type)
					continue
				}
				name := strings.Split(field.Tag.Get("json"), ",")[0]
				if name != "" && name != "-" {
					seen[name] = true
				}
			}
		}
		fields(reflect.TypeOf(dto))
		if len(seen) != len(schema.Value.Properties) {
			t.Fatalf("%s schema field count differs from DTO", schemaName)
		}
		for field := range seen {
			if schema.Value.Properties[field] == nil {
				t.Fatalf("%s.%s missing from OpenAPI", schemaName, field)
			}
		}
	}
	// Detect newly mounted routes that were not added to the contract.
	for source, prefix := range map[string]string{"router.go": "/api/v1", filepath.Join("..", "..", "router", "router.go"): ""} {
		file, err := parser.ParseFile(token.NewFileSet(), source, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) < 1 {
				return true
			}
			fn, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (fn.Sel.Name != "Handle" && fn.Sel.Name != "HandleFunc") {
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			path, _ := strconv.Unquote(literal.Value)
			if path == "/" {
				return true
			}
			path = strings.ReplaceAll(prefix+path, "{$}", "")
			if doc.Paths.Value(path) == nil {
				t.Errorf("mounted route %s missing from OpenAPI", path)
			}
			return true
		})
	}
	text, err := os.ReadFile(filepath.Join("..", "..", "apperrors", "errors.go"))
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{"INTERNAL_ERROR": true}
	for _, match := range regexp.MustCompile(`\d{3},\s*"([A-Z_]+)"`).FindAllStringSubmatch(string(text), -1) {
		codes[match[1]] = true
	}
	listed := doc.Components.Schemas["Error"].Value.Properties["code"].Value.Enum
	if len(codes) != len(listed) {
		t.Fatal("OpenAPI error codes differ from implementation")
	}
	for _, code := range listed {
		if !codes[code.(string)] {
			t.Fatalf("undocumented/obsolete error code %s", code)
		}
	}
}
