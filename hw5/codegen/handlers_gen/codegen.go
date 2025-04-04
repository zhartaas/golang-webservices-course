package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

type tpl struct {
	FuncName string
}

var (
	unknownMethodGetQuery = `
	if r.Method == http.MethodPost {
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		query = string(bytes)
	} else {
		query = r.URL.RawQuery
	}
`
	postMethodGetQuery = `
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	query = string(bytes)
`
	authCheck = `if r.Header.Get("X-Auth") != "100500" {
		responseJson, _ := json.Marshal(Response{Error: "unauthorized"})
		w.WriteHeader(http.StatusForbidden)
		w.Write(responseJson)
		return
	}
`
	unpackAndCheck = `unpackErr := params.Unpack(query)
	if unpackErr != nil {
		responseJson, _ := json.Marshal(Response{Error: unpackErr.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseJson)
		return
	}`
	sendResponseTpl = template.Must(template.New("sendResponseTpl").Parse(`
	responseRaw, err := srv.{{.FuncName}}(r.Context(), *params)

	if err != nil {
		switch apiErr := err.(type) {
		case ApiError:
			w.WriteHeader(apiErr.HTTPStatus)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		responseJson, _ := json.Marshal(Response{Error: err.Error()})

		w.Write(responseJson)
		return
	}

	response := Response{Response: responseRaw}
	responseJson, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseJson)
`))
	postMethodCheck = `	if r.Method != http.MethodPost {
		responseJson, _ := json.Marshal(Response{Error: "bad method"})
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(responseJson)
		return
	}`
)

type HandlerParams struct {
	URL    string `json:"url"`
	Name   string
	Auth   bool   `json:"auth"`
	Method string `json:"method"`
}

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out)
	fmt.Fprintln(out, `import (`)
	fmt.Fprintln(out, `"encoding/json"
	"errors"
	"io" 
	"net/http"
	"net/url"
	"strconv"
)
`)
	fmt.Fprintln(out, "type Data interface{}")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "type Response struct {")
	fmt.Fprintln(out, "Error string `json:\"error\"`")
	fmt.Fprintln(out, "Response Data `json:\"response,omitempty\"`")
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)
	handlers := make(map[string][]HandlerParams)

	for _, f := range node.Decls {
		switch g := f.(type) {
		case *ast.GenDecl:
			fmt.Printf("GenDecl %v\n", g)
			//fmt.Println(g.Specs, len(g.Specs))
			for _, spec := range g.Specs {
				currType, ok := spec.(*ast.TypeSpec)
				if !ok {
					fmt.Printf("SKIP %#T is not ast.TypeSpec\n", spec)
					continue
				}

				currStruct, ok := currType.Type.(*ast.StructType)
				if !ok {
					fmt.Printf("SKIP %#T is not ast.StructType\n", currStruct)
					continue
				}

				needCodegen := true
				for _, field := range currStruct.Fields.List {
					if field.Tag != nil {
						tagString := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
						tagsRaw, ok := tagString.Lookup("apivalidator")
						if !ok {
							fmt.Printf("SKIP %#T doesnt contain apivalidator\n", field)
							continue
						}
						tags := strings.Split(tagsRaw, ",")
						if needCodegen {
							fmt.Printf("\tgenerating Unpack method\n")
							fmt.Fprintln(out, "func (in *"+currType.Name.Name+") Unpack(query string) error {")
							fmt.Fprintln(out, "values, err := url.ParseQuery(query)")
							fmt.Fprintln(out, "if err != nil {")
							fmt.Fprintln(out, "return err ")
							fmt.Fprintln(out, "}")
							fmt.Fprintln(out)
							needCodegen = false
						}

						key := ""
						defaultValue := ""
						for _, tag := range tags {
							if strings.HasPrefix(tag, "paramname=") {
								key = strings.TrimPrefix(tag, "paramname=")
								break
							}
							if strings.HasPrefix(tag, "default=") {
								defaultValue = strings.TrimPrefix(tag, "default=")
							}
						}

						fieldName := field.Names[0].Name
						fieldType := field.Type.(*ast.Ident).Name

						varName := strings.ToLower(fieldName)

						fmt.Println("\t generating code for field %s.%s\n", currType.Name.Name, fieldName)
						if key == "" {
							key = varName
						}
						if fieldType == "int" {
							varName += "Raw"
						}
						fmt.Fprintln(out, varName+" := values.Get(\""+key+"\")")
						if defaultValue != "" {
							fmt.Fprintln(out, "if "+varName+" == \"\" {")
							fmt.Fprintln(out, varName+" = \""+defaultValue+"\"")
							fmt.Fprintln(out, "}")
							fmt.Fprintln(out)
						}
						if fieldType == "int" {
							varName = strings.TrimSuffix(varName, "Raw")
							fmt.Fprintln(out, varName+", err := strconv.Atoi("+varName+"Raw)")
							fmt.Fprintln(out, "if err != nil { return errors.New(\""+strings.TrimSuffix(varName, "Raw")+" must be int\") }")

						}

						for _, tag := range tags {
							if strings.HasPrefix(tag, "required") {
								fmt.Fprintln(out, "if len("+varName+") == 0 { ")
								fmt.Fprintln(out, "return errors.New(\""+varName+" must be not empty\")")
								fmt.Fprintln(out, "}")
								fmt.Fprintln(out)
							}
							if strings.HasPrefix(tag, "enum=") {
								enums := strings.Split(strings.TrimPrefix(tag, "enum="), "|")
								fmt.Fprintln(out, "if "+varName+" != \""+strings.Join(enums, ("\" && "+varName+" != \""))+"\" {")
								fmt.Fprintln(out, "return errors.New(\""+varName+" must be one of ["+strings.Join(enums, ", ")+"]\")")
								fmt.Fprintln(out, "}")
							}
							if strings.HasPrefix(tag, "min=") {
								min := strings.TrimPrefix(tag, "min=")
								_, err := strconv.Atoi(min)
								if err != nil {
									fmt.Println("bad tag " + currType.Name.Name + " " + tagsRaw)
									return
								}
								if fieldType == "string" {
									fmt.Fprintln(out, "if len("+varName+") < "+min+"{ return errors.New(\""+varName+" len must be >= "+min+"\") }")
								} else {
									fmt.Fprintln(out, "if "+varName+" < "+min+"{ return errors.New(\""+varName+" must be >= "+min+"\") }")
								}
							}
							if strings.HasPrefix(tag, "max=") {
								max := strings.TrimPrefix(tag, "max=")
								_, err := strconv.Atoi(max)
								if err != nil {
									fmt.Println("bad tag " + currType.Name.Name + " " + tagsRaw)
									return
								}
								if fieldType == "string" {
									fmt.Fprintln(out, "if len("+varName+") > "+max+"{ return errors.New(\""+varName+" len must be <= "+max+"\")")
								} else {
									fmt.Fprintln(out, "if "+varName+" > "+max+"{ return errors.New(\""+varName+" must be <= "+max+"\") }")
								}
							}
						}
						fmt.Fprintln(out, "in."+fieldName+" = "+varName)
					}
				}
				if !needCodegen {

					fmt.Fprintln(out, "return nil")
					fmt.Fprintln(out, "}")
					fmt.Fprintln(out)
				}
			}
		case *ast.FuncDecl:
			fmt.Printf("FuncDecl %v\n", g)
			needCodegen := false
			handlerParams := &HandlerParams{}

			if g.Doc == nil {
				fmt.Printf("SKIP func %#v doesnt have comments\n", g.Name.Name)
				continue
			}
			for _, comment := range g.Doc.List {
				needCodegen = needCodegen || strings.HasPrefix(comment.Text, "// apigen:api")
				err := json.Unmarshal([]byte(strings.TrimPrefix(comment.Text, "// apigen:api ")), handlerParams)
				if err != nil {
					log.Fatal(err)
				}
			}
			if !needCodegen {
				fmt.Printf("SKIP struct %#v doesnt have apigen mark\n", g.Name.Name)
				continue
			}

			fmt.Printf("\t generating " + g.Name.Name + " handler\n")
			// func name
			fmt.Println(g.Name.Name)
			// method variable
			fmt.Printf("1%+v\n", g.Recv.List[0].Names[0].Name)
			// struct name method
			fmt.Printf("2%+v\n %#T\n", g.Recv.List[0].Type, g.Recv.List[0].Type)

			structNameMethod := g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
			handlerParams.Name = g.Name.Name
			handlers[structNameMethod] = append(handlers[structNameMethod], *handlerParams)

			// parameters and return parameters
			fmt.Printf("3%+v\n", g.Type.Params.List[1])
			fmt.Printf("4%+v\n", g.Type.Params.List[1].Type)
			fmt.Printf("5%+v\n", g.Type.Results.List[1].Type)
			// block code
			fmt.Printf("6%+v\n %#T\n", g.Body.List[0], g.Body.List[0])

			fmt.Println(handlerParams)

			fmt.Fprintln(out, "func ("+g.Recv.List[0].Names[0].Name+" *"+structNameMethod+") handler"+g.Name.Name+
				"(w http.ResponseWriter, r *http.Request) {")
			fmt.Fprintln(out, "query := \"\"")
			if handlerParams.Method == "" {
				fmt.Fprintln(out, unknownMethodGetQuery)
			} else if handlerParams.Method == http.MethodPost {
				fmt.Fprintln(out, postMethodCheck)
				fmt.Fprintln(out, postMethodGetQuery)
			}
			if handlerParams.Auth == true {
				fmt.Fprintln(out, authCheck)
			}
			paramVariableName := ""
			for _, methodParams := range g.Type.Params.List {
				switch t := methodParams.Type.(type) {
				case *ast.SelectorExpr:
					if t.Sel.Name == "Context" {
						continue
					}
					paramVariableName = t.X.(*ast.Ident).Name + "." + t.Sel.Name
				case *ast.Ident:
					paramVariableName = t.Name
				}
				fmt.Println(paramVariableName)
			}
			fmt.Fprintln(out, "params := &"+paramVariableName+"{}")
			fmt.Fprintln(out, unpackAndCheck)
			sendResponseTpl.Execute(out, tpl{g.Name.Name})
			fmt.Fprintln(out, "}")

		default:
			fmt.Printf("SKIP %#T is not *ast.GenDecl\n", f)
		}
	}
	fmt.Println(handlers)
	// adding ServeHTTP()

	for structName, handlerArray := range handlers {
		fmt.Fprintln(out, "func (srv *"+structName+") ServeHTTP(w http.ResponseWriter, r *http.Request) {")
		fmt.Fprintln(out, "switch r.URL.Path {")
		for _, handler := range handlerArray {
			fmt.Fprintln(out, "case \""+handler.URL+"\":")
			fmt.Fprintln(out, "srv.handler"+handler.Name+"(w,r)")
		}
		fmt.Fprintln(out, "default:")
		fmt.Fprintln(out, "responseJson, _ := json.Marshal(Response{Error: \"unknown method\"})")
		fmt.Fprintln(out, "w.WriteHeader(http.StatusNotFound)")
		fmt.Fprintln(out, "w.Write(responseJson)")
		fmt.Fprintln(out, "}")
		fmt.Fprintln(out, "}")

	}
}
