package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

func seed(db *sql.DB) {
	qs := []string{
		`DROP TABLE IF EXISTS items;`,

		`CREATE TABLE items (
  id int(11) NOT NULL AUTO_INCREMENT,
  title varchar(255) NOT NULL,
  description text NOT NULL,
  updated varchar(255) DEFAULT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;`,

		`INSERT INTO items (id, title, description, updated) VALUES
(1,	'database/sql',	'Рассказать про базы данных',	'rvasily'),
(2,	'memcache',	'Рассказать про мемкеш с примером использования',	NULL);`,

		`DROP TABLE IF EXISTS users;`,

		`CREATE TABLE users (
			user_id int(11) NOT NULL AUTO_INCREMENT,
  login varchar(255) NOT NULL,
  password varchar(255) NOT NULL,
  email varchar(255) NOT NULL,
  info text NOT NULL,
  updated varchar(255) DEFAULT NULL,
  PRIMARY KEY (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;`,

		`INSERT INTO users (user_id, login, password, email, info, updated) VALUES
(1,	'rvasily',	'love',	'rvasily@example.com',	'none',	NULL);`,
	}

	for _, q := range qs {
		_, err := db.Exec(q)
		if err != nil {
			panic(err)
		}
	}
}

type ColumnMeta struct {
	Name     string
	Type     string
	Nullable string
	Key      string
	Extra    string
	GoType   string
}

type resultValue map[string]interface{}

type Response struct {
	statusCode int
	response   interface{}
	Err        error `json:"error"`
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	dbTables, err := getTablesStructures(db)
	ctx := context.Background()
	ctx = context.WithValue(ctx, "tables", dbTables)
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rootHandler(ctx, w, r, db)
	}), nil
}

func getColumns(tableName string, db *sql.DB) (map[string]ColumnMeta, error) {
	columns := make(map[string]ColumnMeta)
	query := fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`;", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return columns, err
	}
	defer rows.Close()

	var field, columnType, collation, null, key, extra, privileges, comment sql.NullString
	var defaultValue sql.NullString
	for rows.Next() {
		err := rows.Scan(&field, &columnType, &collation, &null, &key, &defaultValue, &extra, &privileges, &comment)
		if err != nil {
			return columns, err
		}
		columnMeta := ColumnMeta{field.String, columnType.String, null.String, key.String, extra.String, ""}
		columns[field.String] = columnMeta
	}
	return columns, err
}

func findPrimaryKey(cols map[string]ColumnMeta) string {
	for _, col := range cols {
		if col.Key == "PRI" {
			return col.Name
		}
	}
	return ""
}

func getTablesStructures(db *sql.DB) (map[string]map[string]ColumnMeta, error) {
	tableNames, err := getTablesList(db)
	dbStructure := make(map[string]map[string]ColumnMeta)
	if err != nil {
		return dbStructure, err
	}
	for _, name := range tableNames["tables"] {
		columns, err := getColumns(name, db)
		if err != nil {
			return dbStructure, nil
		}
		dbStructure[name] = columns
	}
	return dbStructure, nil
}

func getTablesList(db *sql.DB) (map[string][]string, error) {
	res, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer res.Close()
	var table string
	var tables []string

	for res.Next() {
		res.Scan(&table)
		tables = append(tables, table)
	}
	tablesRes := make(map[string][]string)
	tablesRes["tables"] = tables
	return tablesRes, nil
}

func getTables(_ context.Context, _ *http.Request, db *sql.DB) Response {
	tables, err := getTablesList(db)
	return Response{http.StatusOK, tables, err}
}

func addRow(ctx context.Context, r *http.Request, db *sql.DB) Response {

	var body map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return Response{http.StatusBadRequest, nil, fmt.Errorf("Unabled to parse body json")}
	}

	path, ok := ctx.Value("path").([]string)
	tableName := path[0]
	tables, ok := ctx.Value("tables").(map[string]map[string]ColumnMeta)
	if !ok {
		return Response{http.StatusInternalServerError, nil, fmt.Errorf("failed to parse table columns")}
	}

	tableStructure, ok := tables[tableName]

	if !ok {
		return Response{http.StatusInternalServerError, nil, fmt.Errorf("failed to get table data")}
	}

	queryString := fmt.Sprintf("INSERT INTO %s (", tableName)

	var colsToUpdate []string
	var placeholders []string
	var values []interface{}
	for key, structure := range tableStructure {
		value, ok := body[key]
		if !ok {
			if structure.Nullable == "YES" {
				continue
			} else {
				t := sqlToGoType(structure.Type)
				if t == "string" {
					value = ""
				} else if t == "float64" {
					value = 0
				}
			}
		}
		if strings.Contains(structure.Extra, "auto_increment") {
			continue
		}
		colsToUpdate = append(colsToUpdate, key)
		placeholders = append(placeholders, "?")
		values = append(values, value)
	}
	queryString += strings.Join(colsToUpdate, ", ")
	queryString += ") VALUES "
	queryString += "("
	queryString += strings.Join(placeholders, ", ")
	queryString += ")"

	res, err := db.Exec(queryString, values...)
	if err != nil {
		return Response{http.StatusInternalServerError, nil, fmt.Errorf("Failed update to add a row")}
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Response{http.StatusInternalServerError, nil, fmt.Errorf("Failed to get id of an updated row")}
	}
	idMap := make(map[string]int64)
	primKey := findPrimaryKey(tableStructure)
	idMap[primKey] = id
	return Response{http.StatusOK, idMap, nil}
}

func getRow(ctx context.Context, _ *http.Request, db *sql.DB) Response {
	path, ok := ctx.Value("path").([]string)
	results := make([]resultValue, 0, 0)

	if !ok {
		return Response{http.StatusInternalServerError, results, fmt.Errorf("Error parsing URL")}
	}
	tableName := path[0]
	rowId := path[1]

	tables, ok := ctx.Value("tables").(map[string]map[string]ColumnMeta)
	if !ok {
		return Response{http.StatusInternalServerError, results, fmt.Errorf("Could not fetch row")}
	}
	primaryKey := findPrimaryKey(tables[tableName])

	queryString := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", tableName, primaryKey)
	rows, err := db.Query(queryString, rowId)
	if err != nil {
		return Response{http.StatusInternalServerError, results, fmt.Errorf("Could not fetch row")}
	}
	defer rows.Close()
	fetchedRows, err := getRows(rows)

	if err != nil {
		return Response{http.StatusInternalServerError, results, fmt.Errorf("Could not fetch row")}
	}

	recordsMap := make(map[string]resultValue)

	if len(fetchedRows) == 0 {
		return Response{http.StatusNotFound, nil, fmt.Errorf("record not found")}
	}
	record := fetchedRows[0]
	recordsMap["record"] = record
	return Response{http.StatusOK, recordsMap, nil}
}

func updateRow(ctx context.Context, r *http.Request, db *sql.DB) Response {
	var body map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return Response{http.StatusBadRequest, nil, fmt.Errorf("Unabled to parse body json")}
	}

	tables, ok := ctx.Value("tables").(map[string]map[string]ColumnMeta)
	if !ok {
		return Response{http.StatusInternalServerError, nil, fmt.Errorf("Could not fetch row")}
	}

	path, ok := ctx.Value("path").([]string)
	results := make([]resultValue, 0, 0)

	if !ok {
		return Response{http.StatusInternalServerError, results, fmt.Errorf("Error parsing URL")}
	}
	tableName := path[0]
	rowId := path[1]

	currTable, ok := tables[tableName]
	if !ok {
		return Response{http.StatusBadRequest, nil, fmt.Errorf("Wrong table requested")}
	}

	queryString := fmt.Sprintf("UPDATE %s SET ", tableName)

	var keys []string
	var values []interface{}
	for key, val := range body {

		colMeta, ok := currTable[key]
		if !ok {
			return Response{http.StatusBadRequest, nil, fmt.Errorf("Updating non existing column")}
		}
		if colMeta.Key == "PRI" {
			return Response{http.StatusBadRequest, nil, fmt.Errorf("field %s have invalid type", key)}
		}
		keys = append(keys, key+" = ? ")
		values = append(values, val)
		if val == nil {
			if colMeta.Nullable == "NO" {
				return Response{http.StatusBadRequest, nil, fmt.Errorf("field %s have invalid type", key)}
			}
		} else {
			isValid := validateType(reflect.ValueOf(val).Type(), colMeta.Type)
			if !isValid {
				return Response{http.StatusBadRequest, nil, fmt.Errorf("field %s have invalid type", key)}
			}
		}
	}

	key := findPrimaryKey(currTable)
	queryString += strings.Join(keys, ", ")
	queryString += "WHERE " + key + " = ?"

	values = append(values, rowId)

	res, err := db.Exec(queryString, values...)
	if err != nil {
		return Response{http.StatusInternalServerError, nil, fmt.Errorf("Failed to update row")}
	}
	num, err := res.RowsAffected()
	if err != nil {
		return Response{http.StatusInternalServerError, nil, fmt.Errorf("Failed to update row")}
	}
	responseMap := make(map[string]int64)
	responseMap["updated"] = num
	return Response{http.StatusOK, responseMap, nil}
}

func deleteRow(ctx context.Context, _ *http.Request, db *sql.DB) Response {
	tables, ok := ctx.Value("tables").(map[string]map[string]ColumnMeta)
	if !ok {
		return Response{http.StatusInternalServerError, nil, fmt.Errorf("Could not fetch row")}
	}

	path, ok := ctx.Value("path").([]string)
	results := make([]resultValue, 0, 0)

	if !ok {
		return Response{http.StatusInternalServerError, results, fmt.Errorf("Error parsing URL")}
	}
	tableName := path[0]
	rowId := path[1]

	table, ok := tables[tableName]
	if !ok {
		return Response{http.StatusBadRequest, nil, fmt.Errorf("Table does not exist")}
	}

	primKey := findPrimaryKey(table)
	if primKey == "" {
		return Response{http.StatusBadRequest, nil, fmt.Errorf("Table does not have a primary key")}
	}
	queryString := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", tableName, primKey)
	res, err := db.Exec(queryString, rowId)

	if err != nil {
		return Response{http.StatusInternalServerError, nil, fmt.Errorf("Failed to delete a row")}
	}

	affected, err := res.RowsAffected()

	if err != nil {
		return Response{http.StatusInternalServerError, nil, fmt.Errorf("Failed to find affected rows")}
	}

	resMap := make(map[string]int64)

	resMap["deleted"] = affected
	return Response{http.StatusOK, resMap, nil}
}

func getTable(ctx context.Context, r *http.Request, db *sql.DB) Response {
	path, ok := ctx.Value("path").([]string)
	results := make([]resultValue, 0, 0)
	if !ok {
		return Response{http.StatusInternalServerError, results, fmt.Errorf("Error parsing URL")}
	}
	tableName := path[0]

	var rows *sql.Rows
	var err error

	tables, ok := ctx.Value("tables").(map[string]map[string]ColumnMeta)
	if !ok {
		return Response{http.StatusInternalServerError, results, fmt.Errorf("Could not find the DB")}
	}

	limit := r.URL.Query().Get("limit")
	_, e := strconv.Atoi(limit)
	if limit == "" || e != nil {
		limit = "5"
	}

	offset := r.URL.Query().Get("offset")
	_, e = strconv.Atoi(offset)
	if offset == "" || e != nil {
		offset = "0"
	}

	queryString := fmt.Sprintf("SELECT * FROM %s ", tableName)
	var args []any

	args = append(args, limit)
	queryString += "LIMIT ? "
	if offset != "" {
		args = append(args, offset)
		queryString += "OFFSET ?"
	}

	_, existsTable := tables[tableName]
	if !existsTable {
		return Response{http.StatusNotFound, results, fmt.Errorf("unknown table")}
	}

	rows, err = db.Query(queryString, args...)

	if err != nil {
		return Response{http.StatusInternalServerError, results, err}
	}
	defer rows.Close()

	fetchedRows, err := getRows(rows)
	if err != nil {
		return Response{http.StatusInternalServerError, results, err}
	}
	results = fetchedRows

	if !ok {
		return Response{http.StatusInternalServerError, results, err}
	}
	recordsMap := make(map[string][]resultValue)
	recordsMap["records"] = results
	return Response{http.StatusOK, recordsMap, err}
}

func validateType(dataType reflect.Type, sqlType string) bool {
	dt := dataType.Kind().String()
	switch dt {
	case "string":
		if strings.Contains(sqlType, "varchar") {
			return true
		} else if strings.Contains(sqlType, "text") {
			return true
		}
	case "float64":
		if strings.Contains(sqlType, "int") {
			return true
		}
	}
	return false
}

func sqlToGoType(sqlType string) string {
	if strings.Contains(sqlType, "varchar") {
		return "string"
	}
	if strings.Contains(sqlType, "text") {
		return "string"
	}
	if strings.Contains(sqlType, "int") {
		return "float64"
	}
	return "nil"
}

func rootHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var response Response
	path := filterEmpty(strings.Split(strings.ReplaceAll(r.URL.Path, " ", ""), "/"))
	ctx = context.WithValue(ctx, "path", path)
	if len(path) == 0 && r.Method == "GET" {
		response = getTables(ctx, r, db)
	} else if len(path) == 1 {
		if r.Method == "GET" {
			// GET /$table?limit=5&offset=7
			response = getTable(ctx, r, db)
			// PUT /$table
		} else if r.Method == "PUT" {
			response = addRow(ctx, r, db)
		}
	} else if len(path) == 2 {
		if r.Method == "GET" {
			response = getRow(ctx, r, db)
		} else if r.Method == "POST" {
			response = updateRow(ctx, r, db)
		} else if r.Method == "DELETE" {
			response = deleteRow(ctx, r, db)
		}
	}
	res := make(map[string]interface{})
	if response.Err != nil {
		res["error"] = response.Err.Error()
	} else {
		res["response"] = response.response
	}
	jsonBody, _ := json.Marshal(res)
	w.WriteHeader(response.statusCode)
	w.Write(jsonBody)
}

func filterEmpty(arr []string) []string {
	var newSlice []string
	for i := 0; i < len(arr); i++ {
		if arr[i] == "" {
			continue
		}
		newSlice = append(newSlice, arr[i])
	}
	return newSlice
}

func handleNullable(value interface{}) interface{} {
	switch v := value.(type) {
	case nil:
		return nil
	case sql.NullString:
		if v.Valid {
			return v.String
		}
		return nil
	case sql.NullInt64:
		if v.Valid {
			return v.Int64
		}
		return nil
	case sql.NullBool:
		if v.Valid {
			return v.Bool
		}
		return nil
	case sql.NullTime:
		if v.Valid {
			return v.Time
		}
		return nil
	case sql.NullInt32:
		if v.Valid {
			return v.Int32
		}
		return nil
	case sql.NullInt16:
		if v.Valid {
			return v.Int16
		}
		return nil
	case sql.NullFloat64:
		if v.Valid {
			return v.Float64
		}
		return nil
	case sql.NullByte:
		if v.Valid {
			return v.Byte
		}
		return nil
	default:
		return v
	}
}

func getRows(rows *sql.Rows) ([]resultValue, error) {
	columnTypes, err := rows.ColumnTypes()
	results := make([]resultValue, 0, 0)
	if err != nil {
		return results, nil
	}

	rowValues := make([]reflect.Value, len(columnTypes))
	for i := 0; i < len(columnTypes); i++ {
		// allocate reflect.Value representing a **T value
		rowValues[i] = reflect.New(reflect.PointerTo(columnTypes[i].ScanType()))
	}

	for rows.Next() {
		rowResult := make([]interface{}, len(columnTypes))
		for i := 0; i < len(columnTypes); i++ {
			rowResult[i] = rowValues[i].Interface()
		}

		if err := rows.Scan(rowResult...); err != nil {
			return results, err
		}

		rowMap := make(map[string]interface{})
		// dereference pointers
		for i := 0; i < len(rowValues); i++ {
			if rv := rowValues[i].Elem(); rv.IsNil() {
				rowResult[i] = nil
			} else {
				rowResult[i] = handleNullable(rv.Elem().Interface())
			}

			for i := 0; i < len(columnTypes); i++ {
				columnName := columnTypes[i].Name()
				rowMap[columnName] = rowResult[i]
			}
		}
		results = append(results, rowMap)
	}
	return results, nil
}
