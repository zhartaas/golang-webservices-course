package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

type HTTPResponse struct {
	Error    string      `json:"error,omitempty"`
	Response interface{} `json:"response,omitempty"`
	//Response map[string]interface{} `json:"response,omitempty"`
}

type Table struct {
	tableName string
	fields    []Field
}

type Field struct {
	Field      *string
	Type       *string
	Collation  *string
	Null       *string
	Key        *string
	Default    *string
	Extra      *string
	Privileges *string
	Comment    *string
}

type DbExplorer struct {
	db     *sql.DB
	tables []*Table
}

func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	srv := &DbExplorer{db: db}
	err := srv.getTables()
	if err != nil {
		return nil, err
	}
	return srv, nil
}

func (srv *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		srv.list(w, r)
	case http.MethodPut:
		srv.create(w, r)
	case http.MethodPost:
		srv.update(w, r)
	case http.MethodDelete:
		srv.delete(w, r)
	}
}

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
func (srv *DbExplorer) getTables() error {
	rows, err := srv.db.Query("SHOW TABLES")
	if err != nil {
		return err
	}
	defer rows.Close()

	tables := make([]*Table, 0, 0)
	for rows.Next() {
		table := &Table{}
		rows.Scan(&table.tableName)
		tables = append(tables, table)
		fmt.Println(table.tableName)

	}

	for _, table := range tables {

		sqlQuery := "SHOW FULL COLUMNS FROM " + table.tableName + ";"
		rows, err := srv.db.Query(sqlQuery)

		if err != nil {
			return err
		}
		defer rows.Close()

		fields := make([]Field, 0, 0)
		for rows.Next() {

			field := Field{}
			err = rows.Scan(&field.Field, &field.Type, &field.Collation, &field.Null, &field.Key,
				&field.Default, &field.Extra, &field.Privileges, &field.Comment)
			if err != nil {
				return err
			}
			fields = append(fields, field)
		}
		table.fields = fields

	}

	srv.tables = tables

	return nil
}

func (srv *DbExplorer) tableExists(tableName string) bool {
	return slices.Contains(srv.getTableList(), tableName)
}

func (srv *DbExplorer) getTableList() []string {
	tableList := []string{}

	for _, table := range srv.tables {
		tableList = append(tableList, table.tableName)
	}
	return tableList
}

func (srv *DbExplorer) getPrimaryKey(tablename string) string {
	ts, _ := srv.getTableAndFields(tablename)
	for _, f := range ts.fields {
		if *f.Key == "PRI" {
			return *f.Field
		}
	}
	return ""
}

func (srv *DbExplorer) getById(w http.ResponseWriter, r *http.Request, tableName, id string) {
	primaryKeySQL := srv.getPrimaryKey(tableName)
	sqlQuery := "SELECT * FROM " + tableName + " WHERE " + primaryKeySQL + " = ?"

	rows, err := srv.db.Query(sqlQuery, id)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	defer rows.Close()
	rowExist := rows.Next()

	if !rowExist {
		responseJson, _ := json.Marshal(HTTPResponse{Error: "record not found"})
		w.WriteHeader(http.StatusNotFound)
		w.Write(responseJson)
		return
	}
	cols, err := rows.Columns()
	//fmt.Println(cols)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	result := map[string]interface{}{}
	columns := make([]interface{}, len(cols))
	columnsPointer := make([]interface{}, len(cols))

	for i, _ := range cols {
		columnsPointer[i] = &columns[i]
	}
	rows.Scan(columnsPointer...)

	for i, col := range cols {
		switch val := (*columnsPointer[i].(*interface{})).(type) {
		case []byte:
			//fmt.Printf("%+v %T\n", col, val)
			result[col] = string(val)
		default:
			//fmt.Printf("%+v %T\n", col, val)
			result[col] = val
		}
	}

	response := &HTTPResponse{
		Response: struct {
			Record map[string]interface{} `json:"record"`
		}{
			Record: result,
		},
	}
	responseJson, _ := json.Marshal(response)
	//fmt.Println(string(responseJson))
	w.WriteHeader(http.StatusOK)
	w.Write(responseJson)

}

func (srv *DbExplorer) getTableAndFields(tableName string) (Table, []string) {
	table := Table{}
	for _, t := range srv.tables {
		if t.tableName == tableName {
			table = *t
		}
	}
	fields := make([]string, len(table.fields))
	for i, f := range table.fields {
		fields[i] = *f.Field
	}
	return table, fields
}

func (srv *DbExplorer) list(w http.ResponseWriter, r *http.Request) {
	// removing first "/"
	params := strings.Split(r.URL.Path[1:], "/")
	tableName := params[0]

	if len(tableName) != 0 && !srv.tableExists(tableName) {
		responseJson, _ := json.Marshal(HTTPResponse{Error: "unknown table"})
		w.WriteHeader(http.StatusNotFound)
		w.Write(responseJson)
		return
	}
	if len(tableName) == 0 {
		response := HTTPResponse{
			Response: &struct {
				Tables []string `json:"tables"`
			}{
				Tables: srv.getTableList(),
			},
		}

		responseJson, err := json.Marshal(response)
		if err != nil {
			panic(err)
		}
		//fmt.Println(string(responseJson))
		w.WriteHeader(http.StatusOK)
		w.Write(responseJson)
		return
	} else if len(params) > 1 {
		srv.getById(w, r, tableName, params[1])
		return
	}
	paramQueries := r.URL.Query()

	args := []any{}

	// protect from sql injection

	sqlQuery := "SELECT * FROM " + tableName + " LIMIT ? OFFSET ?"

	limit, _ := strconv.Atoi(paramQueries.Get("limit"))
	if limit == 0 {
		limit = 5
	}
	offset, _ := strconv.Atoi(paramQueries.Get("offset"))

	args = append(args, limit, offset)
	rows, err := srv.db.Query(sqlQuery, args...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	result := []map[string]interface{}{}
	for rows.Next() {

		columns := make([]interface{}, len(cols))
		columnsPointer := make([]interface{}, len(cols))

		for i, _ := range cols {
			columnsPointer[i] = &columns[i]
		}
		rows.Scan(columnsPointer...)

		m := make(map[string]interface{})
		for i, col := range cols {
			switch val := (*columnsPointer[i].(*interface{})).(type) {
			case []byte:
				//fmt.Printf("%+v %T\n", col, val)
				m[col] = string(val)
			default:
				//fmt.Printf("%+v %T\n", col, val)
				m[col] = val
			}
		}
		result = append(result, m)
	}

	response := HTTPResponse{
		Response: &struct {
			Records []map[string]interface{} `json:"records,omitempty"`
		}{
			Records: result,
		},
	}

	resultJson, _ := json.Marshal(response)
	//fmt.Println(string(resultJson))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resultJson)
}

func (srv *DbExplorer) create(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(r.URL.Path[1:], "/")
	tableName := params[0]
	if len(tableName) != 0 && !srv.tableExists(tableName) {
		responseJson, _ := json.Marshal(HTTPResponse{Error: "unknown table"})
		w.WriteHeader(http.StatusNotFound)
		w.Write(responseJson)
		return
	}

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	insertObject := map[string]interface{}{}
	err = json.Unmarshal(bytes, &insertObject)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	table, _ := srv.getTableAndFields(tableName)

	cols := []string{}
	args := []any{}
	for _, f := range table.fields {
		if *f.Extra == "auto_increment" {
			continue
		}
		cols = append(cols, *f.Field)
		arg, ok := insertObject[*f.Field]
		if !ok && *f.Null == "NO" {
			arg = ""
		}
		args = append(args, arg)
	}

	// "INSERT INTO items (`title`, `description`) VALUES (?, ?)"
	attributes := "(" + strings.Join(cols, ", ") + ")"
	argsAmount := "(" + strings.Repeat("?, ", len(cols)-1) + "?)"
	sqlQuery := `INSERT INTO ` + tableName + " " + attributes + " VALUES " + argsAmount

	execRes, err := srv.db.Exec(sqlQuery, args...)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	id, err := execRes.LastInsertId()
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	pk := srv.getPrimaryKey(tableName)
	res := map[string]interface{}{
		pk: id,
	}
	response := HTTPResponse{
		Response: res,
	}

	responseJson, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJson)
}

func checkTypes(field Field, val interface{}) bool {
	if *field.Key == "PRI" {
		return false
	}

	t := *field.Type
	switch val.(type) {
	case float64:
		return t == "int"
	case string:
		return strings.HasPrefix(t, "varchar") || t == "text"
	case nil:
		return *field.Null == "YES"
	default:
		fmt.Printf("Unknown type %T\n", val)
		return false
	}
}

func (srv *DbExplorer) update(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(r.URL.Path[1:], "/")
	tableName := params[0]

	if len(tableName) != 0 && !srv.tableExists(tableName) {
		responseJson, _ := json.Marshal(HTTPResponse{Error: "unknown table"})
		w.WriteHeader(http.StatusNotFound)
		w.Write(responseJson)
		return
	}

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	updateObject := map[string]interface{}{}
	err = json.Unmarshal(bytes, &updateObject)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	table, fields := srv.getTableAndFields(tableName)

	cols := []string{}
	args := []any{}
	for key, val := range updateObject {
		if i := slices.Index(fields, key); i != -1 {
			if !checkTypes(table.fields[i], val) {
				responseJson, _ := json.Marshal(HTTPResponse{Error: ("field " + key + " have invalid type")})
				w.WriteHeader(http.StatusBadRequest)
				w.Write(responseJson)
				return
			}
		}
		cols = append(cols, key)
		args = append(args, val)
	}
	args = append(args, params[1])

	primaryKeySQL := srv.getPrimaryKey(tableName)
	sqlQuery := `UPDATE ` + tableName + " SET " + strings.Join(cols, "=?,") + "=? WHERE " + primaryKeySQL + "=?"
	sqlRes, err := srv.db.Exec(sqlQuery, args...)
	if err != nil {
		fmt.Println("SQL ERR", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	rowsAffected, err := sqlRes.RowsAffected()
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	response := HTTPResponse{Response: struct {
		Updated int `json:"updated"`
	}{
		Updated: int(rowsAffected),
	},
	}

	responseJson, _ := json.Marshal(response)
	w.WriteHeader(http.StatusOK)
	w.Write(responseJson)
}

func (srv *DbExplorer) delete(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(r.URL.Path[1:], "/")
	tableName := params[0]
	id := params[1]
	if len(tableName) != 0 && !srv.tableExists(tableName) {
		responseJson, _ := json.Marshal(HTTPResponse{Error: "unknown table"})
		w.WriteHeader(http.StatusNotFound)
		w.Write(responseJson)
		return
	}

	primaryKeySQL := srv.getPrimaryKey(tableName)
	sqlQuery := "DELETE FROM " + tableName + " WHERE " + primaryKeySQL + " =?"
	sqlRes, err := srv.db.Exec(sqlQuery, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	rowsAffected, _ := sqlRes.RowsAffected()

	response := HTTPResponse{Response: struct {
		Deleted int `json:"deleted"`
	}{
		Deleted: int(rowsAffected),
	}}

	responseJson, _ := json.Marshal(response)

	w.WriteHeader(http.StatusOK)
	w.Write(responseJson)
}
