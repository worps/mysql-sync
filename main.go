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

var source = flag.String("source", "", "sync from, eg: test@(10.10.0.1:3306)/my_online_db_name\nwhen it is not empty,[-conf] while ignore")
var dest = flag.String("dest", "", "sync to, eg: test@(127.0.0.1:3306)/my_local_db_name")
var tables = flag.String("tables", "", "tables to sync\neg : product_base,order_*")
var tablesIgnore = flag.String("tables_ignore", "", "tables ignore sync\neg : product_base,order_*,*_bak*")
var tablesCompareData = flag.String("tables_compare_data", "", "tables to compare data\neg : product_base,order_*,*_bak*")
var singleSchemaChange = flag.Bool("single_schema_change", false, "single schema changes ddl command a single schema change")

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

func main() {
	flag.Parse()
	if len(*source) == 0 {
		cfg = internal.LoadConfig(*configPath)
	} else {
		cfg = new(internal.Config)
		cfg.SourceDSN = *source
		cfg.DestDSN = *dest
	}
	cfg.Sync = *sync
	cfg.Drop = *drop
	cfg.SingleSchemaChange = *singleSchemaChange

	cfg.SetTables(strings.Split(*tables, ","))
	cfg.SetTablesIgnore(strings.Split(*tablesIgnore, ","))
	cfg.SetTablesCompareData(strings.Split(*tablesCompareData, ","))

	defer (func() {
		if re := recover(); re != nil {
			log.Println(re)
			bf := make([]byte, 4096)
			n := runtime.Stack(bf, false)
			log.Fatalln("panic:", string(bf[:n]))
		}
	})()

	internal.PrintDbName(cfg.SourceDSN)

	cfg.Check()
	internal.CheckDiffData(cfg)
	internal.CheckAlterProcedure(cfg)
	internal.CheckSchemaDiff(cfg)
}
