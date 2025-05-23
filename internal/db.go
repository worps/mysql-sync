package internal

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"

	_ "github.com/go-sql-driver/mysql" // mysql driver
)

// MyDb db struct
type MyDb struct {
	Db     *sql.DB
	dbType string
}

// NewMyDb parse dsn
func NewMyDb(dsn string, dbType string) *MyDb {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(fmt.Sprintf("connected to db [%s] failed,err=%s", dsn, err))
	}
	return &MyDb{
		Db:     db,
		dbType: dbType,
	}
}

// print dbName
func PrintDbName(dsn string) {
	db, err := sql.Open("mysql", dsn)
	if err == nil {
		rs, err := db.Query("SELECT DATABASE()")
		if err == nil {
			rs.Next()
			dbname := ""
			rs.Scan(&dbname)
			fmt.Printf("------------------------ db %s -------------------------\n", dbname)
		}
	}
}

// GetTableNames table names
func (db *MyDb) GetTableNames() []string {
	rs, err := db.Query("show table status")
	if err != nil {
		panic("show tables failed:" + err.Error())
	}
	defer rs.Close()

	var tables []string
	columns, _ := rs.Columns()
	for rs.Next() {
		var values = make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rs.Scan(valuePtrs...); err != nil {
			panic("show tables failed when scan," + err.Error())
		}
		var valObj = make(map[string]any)
		for i, col := range columns {
			var v any
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			valObj[col] = v
		}
		if valObj["Engine"] != nil {
			tables = append(tables, valObj["Name"].(string))
		}
	}
	return tables
}

// Get procedure names
func (db *MyDb) GetProcedureNames() []string {
	rs, err := db.Query(`SELECT SPECIFIC_NAME
		FROM information_schema.ROUTINES
		WHERE ROUTINE_TYPE = 'PROCEDURE' 
		AND ROUTINE_SCHEMA = DATABASE()`)
	if err != nil {
		panic("show procedure failed:" + err.Error())
	}
	defer rs.Close()

	var procedures []string
	for rs.Next() {
		var vname string
		if err := rs.Scan(&vname); err != nil {
			panic(fmt.Sprintf("get procedure failed, %s", err))
		}
		procedures = append(procedures, vname)
	}
	return procedures
}

// GetTableSchema table schema
func (db *MyDb) GetTableSchema(name string) (schema string) {
	rs, err := db.Query(fmt.Sprintf("show create table `%s`", name))
	if err != nil {
		// 可能表不存在
		return
	}

	defer rs.Close()
	for rs.Next() {
		var vname string
		if err := rs.Scan(&vname, &schema); err != nil {
			panic(fmt.Sprintf("get table %s 's schema failed, %s", name, err))
		}
		// 生成建表语句中可能包含了字段使用的字符集，可能在其他库中不显示使用的字符集而对比出差异。需要替换掉 CHARACTER SET xxxx
		reg, _ := regexp.Compile("CHARACTER SET [a-z0-9_]+ ")
		schema = reg.ReplaceAllString(schema, "")
	}
	return
}

// Get procedure schema
func (db *MyDb) GetProcedureSchema(name string) (schema string) {
	rs, err := db.Query(fmt.Sprintf("show create PROCEDURE `%s`", name))
	if err != nil {
		log.Println(err)
		return
	}
	defer rs.Close()
	for rs.Next() {
		var vname, sqlmode, chars, coll, dbcoll string
		if err := rs.Scan(&vname, &sqlmode, &schema, &chars, &coll, &dbcoll); err != nil {
			panic(fmt.Sprintf("get table %s 's schema failed, %s", name, err))
		}
	}
	return
}

// Query execute sql query
func (db *MyDb) Query(query string, args ...any) (*sql.Rows, error) {
	// log.Println("[SQL]", "["+db.dbType+"]", query, args)
	return db.Db.Query(query, args...)
}
