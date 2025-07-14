package main

import (
	"flag"
	"fmt"
	"log"
	"mysql-sync/internal"
	"os"
	"runtime"
	"strings"
)

var configPath = flag.String("conf", "./conf.json", "json config file path")
var sync = flag.Bool("sync", false, "sync schema changes to dest's db\non default, only show difference")
var drop = flag.Bool("drop", false, "drop fields,index,foreign key only on dest's table")
var singleSchemaChange = flag.Bool("single_schema_change", false, "single schema changes ddl command a single schema change")

var sql2compare = flag.String("sql_check", "", "sql to compare result on both dsn")
var sqlFile = flag.String("sql_file", "", "sql file path")

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate)
	df := flag.Usage
	flag.Usage = func() {
		df()
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "mysql schema sync tools "+internal.Version)
		fmt.Fprint(os.Stderr, internal.AppURL+"\n\n")
	}
}

var cfg *internal.Config

// 对比两个dsn下的数据库
func compareDSN() {
	cfg.Sync = *sync
	cfg.Drop = *drop
	cfg.SingleSchemaChange = *singleSchemaChange
	cfg.Check()

	syncInstance := internal.NewSchemaSync(cfg)

	for _, _dbname := range cfg.Schemas {
		syncInstance.UseDb(_dbname)
		syncInstance.CheckDiffData(cfg)
		internal.CheckAlterProcedure(cfg)
		internal.CheckSchemaDiff(cfg)
	}
}

// 向目标库导入sql
func importSQL(file string) {
	sqls, err := os.ReadFile(file)
	if err != nil {
		log.Fatal("read sql file failed: ", err)
	}
	sqlStr := string(sqls)

	sqlArr := strings.Split(sqlStr, ";\n")
	sc := internal.NewSchemaSync(cfg)
	err = sc.SyncSQL4Dest(sqlStr, sqlArr)
	if err != nil {
		log.Fatalf("execute failed, error: %v\n", err)
	}
	log.Println("execute_all_sql_done, none error")
}

// 对比sql在两个dsn执行的结果
func compareSQL() {
	if len(*sql2compare) <= 0 {
		log.Fatalln("param `sql_check` is necessary")
	}
	internal.CompareSqlResult(cfg, *sql2compare)
}

func main() {
	flag.Parse()
	cfg = internal.LoadConfig(*configPath)

	defer (func() {
		if re := recover(); re != nil {
			log.Println(re)
			bf := make([]byte, 4096)
			n := runtime.Stack(bf, false)
			log.Fatalln("panic:", string(bf[:n]))
		}
	})()

	if len(*sql2compare) > 0 {
		// 对比sql的执行结果
		compareSQL()
	} else if len(*sqlFile) > 0 {
		// 在目标库执行sql文件
		importSQL(*sqlFile)
	} else {
		// 对比或同步两个数据库
		compareDSN()
	}
}
