package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
)

type tpl struct {
	ApiName string
}

type validator struct {
	Required  bool
	Min       string
	Max       string
	ParamName string
	Enum      []string
	Def       string
	Type      string
}

type ApiShape struct {
	Url        string
	Auth       bool
	Method     string
	ParamsName string
	Params     map[string]validator
}

var funcMap = template.FuncMap{
	"join":     strings.Join,
	"contains": contains,
	"lower":    strings.ToLower,
	"toInt":    strconv.Atoi,
}

var wrapperTpl, _ = template.Must(template.New("wrapperTpl"), nil).Funcs(funcMap).Parse(`
{{range $key, $val := .}}
	{{ range $methodKey, $item := $val }}
	 /* {{$item.Method}} */
		func (h *{{$key}}) wrapper{{$methodKey}}(w http.ResponseWriter, r *http.Request) (interface{}, *ApiError) {
			var rawParams map[string]string
			{{ if ne $item.Method "" }}
			if (r.Method != "{{$item.Method}}") {
					return nil, &ApiError{http.StatusNotAcceptable, errors.New("bad method")}
			}
			{{end}}
			
			{{if .Auth}}
				var authValue string
				auth := r.Header.Get("X-Auth")
				if len(auth) > 0 {
					authValue = string(r.Header.Get("X-Auth"))
				}
			if authValue != "100500" {
				return nil, &ApiError{http.StatusForbidden, errors.New("unauthorized")}
			}
			{{end}}

			if (r.Method == "POST") {
				body, err := io.ReadAll(r.Body)
				if (err != nil) {
					return nil, &ApiError{http.StatusInternalServerError, err}
					}
					parsedBody, err := url.ParseQuery(string(body))
					if (err != nil) {
						return nil, &ApiError{http.StatusInternalServerError, err}
						}
						rawParams = unpackValues(parsedBody)
						} else {
							 query := r.URL.Query()
							rawParams = unpackValues(query)
							}
							/* Validate Params */
							var validatedParams {{$item.ParamsName}}
							var rawValue string
							var ok bool
							{{ range $paramKey, $paramVal := $item.Params}}
							var {{$paramKey}}Param {{$paramVal.Type}}
							{{if eq $paramVal.ParamName ""}}
							rawValue, ok = rawParams["{{lower $paramKey}}"]
							{{else}}
							/* {{$paramVal.ParamName}} */
							rawValue, ok = rawParams["{{$paramVal.ParamName}}"]
							{{end}}
							
							{{if eq $paramVal.Required true}}
							/* Required */
							if !ok {
							return nil, &ApiError{http.StatusBadRequest, errors.New("{{lower $paramKey}} must be not empty")}
							}
							{{end}}

							if ok {
								{{if eq $paramVal.Type "string"}}
								{{$paramKey}}Param = rawValue
								{{else}}
								var err error
								{{$paramKey}}Param, err = strconv.Atoi(rawValue)
								if (err != nil) {
									return nil, &ApiError{http.StatusBadRequest, errors.New("{{lower $paramKey}} must be {{$paramVal.Type}}")}
								}
								{{end}}
								}

								/* Default */
								{{if ne $paramVal.Def ""}}
								{{if eq $paramVal.Type "string"}}
								if {{$paramKey}}Param == "" {
									{{$paramKey}}Param = "{{$paramVal.Def}}"
								}
								{{else}}
								if {{$paramKey}}Param == 0 {
									{{$paramKey}}Param = {{$paramVal.Def}}
								}
								{{end}}
								
								{{end}}
								
								{{ if ne $paramVal.Min ""}}
								/* Min */
								{{$paramKey}}Min := {{$paramVal.Min}}
								{{if eq $paramVal.Type "string"}}
								if len({{$paramKey}}Param) < {{$paramKey}}Min {
									return nil, &ApiError{http.StatusBadRequest, errors.New("{{lower $paramKey}} len must be >= {{$paramVal.Min}}")}
									}
									{{else}}
									if {{$paramKey}}Param < {{$paramKey}}Min {
									return nil, &ApiError{http.StatusBadRequest, errors.New("{{lower $paramKey}} must be >= {{$paramVal.Min}}")}
								}
								{{end}}
								{{end}}
								{{ if ne $paramVal.Max ""}}
								/* Max */
								{{$paramKey}}Max := {{$paramVal.Max}}
								{{if eq $paramVal.Type "string"}}
								if len({{$paramKey}}Param) > {{$paramKey}}Max {
									return nil, &ApiError{http.StatusBadRequest, errors.New("{{lower $paramKey}} len must be <= {{$paramVal.Max}}")}
								}
								{{else}}
								if {{$paramKey}}Param > {{$paramKey}}Max {
									return nil, &ApiError{http.StatusBadRequest, errors.New("{{lower $paramKey}} must be <= {{$paramVal.Max}}")}
								}
								{{end}}
								{{end}}
														{{ $length := len $paramVal.Enum }} {{ if ne $length 0 }}
							/* Enum */
							 
							{{$paramKey}}Options := []string {"{{join $paramVal.Enum "\",\""}}"}
							if !slices.Contains({{$paramKey}}Options, {{$paramKey}}Param) {
								return nil, &ApiError{http.StatusBadRequest, errors.New("{{lower $paramKey}} must be one of [{{join $paramVal.Enum ", "}}]")}
							}
							{{end}}
								
								validatedParams.{{$paramKey}} = {{$paramKey}}Param
							{{ end }}
			ctx := context.Background()
			res, err := h.{{$methodKey}}(ctx, validatedParams)

			if err != nil {
				if apiErr, ok := err.(ApiError); ok {
					return nil, &ApiError{apiErr.HTTPStatus, err}
				}
					return nil, &ApiError{http.StatusInternalServerError, err}
			}
			return res, nil
		}
	{{end}}
{{end}}
`)

var utilTpl, _ = template.Must(template.New("utilTpl"), nil).Parse(`
import (
	"net/http"
	"context"
	"errors"
	"slices"
	"strconv"
	"encoding/json"
	"io"
	"net/url"
)

func unpackValues(valuesMap map[string][]string) map[string]string {
	var unpackedMap = make(map[string]string)
	for key, value := range valuesMap {
		if len(value) > 0 {
			unpackedMap[key] = value[0]
		}
	}
	return unpackedMap
}
`)

var methodsTpl, _ = template.Must(template.New("methodsTpl"), nil).Parse(`
{{range $k, $v := .}}
func (srv *{{$k}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
var response interface{}
var error *ApiError
	switch r.URL.Path {
		{{range $key, $val := $v}}
		case "{{$val.Url}}":
			response, error = srv.wrapper{{$key}}(w,r)
		{{end}}
		default:
			error = &ApiError{http.StatusNotFound, errors.New("unknown method")}
		}
			
			res := make(map[string]interface{})
			if error != nil {
			w.WriteHeader(error.HTTPStatus)
			res["error"] = error.Error()
		} else {
			res["error"] = ""
			res["response"] = response
		}
		jsonRes, _ := json.Marshal(res)
		w.Write(jsonRes)
		}
		{{end}}
`)

func main() {

	if len(os.Args) < 3 {
		log.Fatal("Must provide two arguments: FROM and TO paths")
		return
	}

	fileIn := os.Args[1]
	fileOut, err := os.Create(os.Args[2])

	if err != nil {
		log.Fatal("Error trying to create a file")
		return
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, fileIn, nil, parser.ParseComments)
	fmt.Fprintln(fileOut, `package `+node.Name.Name)
	utilTpl.Execute(fileOut, nil)
	fmt.Fprintln(fileOut, "type ResError struct { Error string `json:\"error\"` }")

	apiMap := parseMethods(node.Decls)

	wrapperTpl.Execute(fileOut, apiMap)
	methodsTpl.Execute(fileOut, apiMap)

	if err != nil {
		log.Fatal(err)
	}
}

func (shape *ApiShape) UnmarshalJSON(text []byte) error {
	type defaults ApiShape
	opts := defaults{}

	if err := json.Unmarshal(text, &opts); err != nil {
		return err
	}
	*shape = ApiShape(opts)
	return nil
}

func parseMethods(decls []ast.Decl) map[string]map[string]ApiShape {
	apiMap := make(map[string]map[string]ApiShape)

	for _, f := range decls {
		fd, ok := f.(*ast.FuncDecl)

		if !ok {
			continue
		}

		comment := fd.Doc.Text()
		if strings.HasPrefix(comment, "apigen:api") == false {
			continue
		}

		apiStructureJSON := strings.Replace(comment, "apigen:api", "", 1)
		var shape ApiShape
		shape.UnmarshalJSON([]byte(apiStructureJSON))

		handlerName := fd.Name.Name

		funcList := fd.Recv.List

		if len(fd.Type.Params.List) > 1 {
			param := types.ExprString(fd.Type.Params.List[1].Type)
			shape.ParamsName = param
			parsedStruct := parseStructs(decls, param)
			shape.Params = parsedStruct
		}

		for _, item := range funcList {
			starExpr, ok := item.Type.(*ast.StarExpr)
			if !ok {
				continue
			}

			apiName := types.ExprString(starExpr.X)
			val, ok := apiMap[apiName]
			if ok {
				val[handlerName] = shape
			} else {
				apiMap[apiName] = make(map[string]ApiShape)
				apiMap[apiName][handlerName] = shape
			}
		}
	}
	return apiMap
}

func parseStructs(decls []ast.Decl, paramName string) map[string]validator {
	vMap := make(map[string]validator)

	for _, d := range decls {
		decl, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range decl.Specs {
			currType, ok := spec.(*ast.TypeSpec)
			if !ok {
				// loop over types only
				continue
			}
			name := currType.Name.Name

			if name != paramName {
				continue
			}

			currStruct, ok := currType.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range currStruct.Fields.List {
				tag := field.Tag.Value
				fieldName := types.ExprString(field.Names[0])
				fieldType := field.Type

				if !strings.Contains(tag, "apivalidator:") {
					// skip fields without validator
					continue
				}
				tag = strings.Replace(tag, "apivalidator:", "", 1)
				tag = strings.ReplaceAll(tag, "\"", "")
				tag = strings.ReplaceAll(tag, "`", "")
				tags := strings.Split(tag, ",")

				validator := validator{}

				validator.Type = types.ExprString(fieldType)
				for _, record := range tags {
					if record == "required" {
						validator.Required = true
						continue
					}
					pair := strings.Split(record, "=")
					key, value := pair[0], pair[1]
					if key == "enum" {
						validator.Enum = strings.Split(value, "|")
					} else if key == "min" {
						validator.Min = value
					} else if key == "max" {
						validator.Max = value
					} else if key == "paramname" {
						validator.ParamName = value
					} else if key == "default" {
						validator.Def = value
					}
				}
				vMap[fieldName] = validator
			}
		}
	}
	return vMap
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
