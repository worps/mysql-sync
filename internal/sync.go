package internal

import (
	"fmt"
	"log"
	"strings"
)

// SchemaSync 配置文件
type SchemaSync struct {
	Config   *Config
	SourceDb *MyDb
	DestDb   *MyDb
}

// NewSchemaSync 对一个配置进行同步
func NewSchemaSync(config *Config) *SchemaSync {
	s := new(SchemaSync)
	s.Config = config
	s.SourceDb = NewMyDb(config.SourceDSN, "source")
	s.DestDb = NewMyDb(config.DestDSN, "dest")
	return s
}

// GetNewTableNames 获取所有新增加的表名
func (sc *SchemaSync) GetNewTableNames() []string {
	sourceTables := sc.SourceDb.GetTableNames()
	destTables := sc.DestDb.GetTableNames()

	var newTables []string

	for _, name := range sourceTables {
		if !inStringSlice(name, destTables) {
			newTables = append(newTables, name)
		}
	}
	return newTables
}

// 合并源数据库和目标数据库的表名
func (sc *SchemaSync) GetTableNames() []string {
	sourceTables := sc.SourceDb.GetTableNames()
	destTables := sc.DestDb.GetTableNames()
	var tables []string
	tables = append(tables, destTables...)
	for _, name := range sourceTables {
		if !inStringSlice(name, tables) {
			tables = append(tables, name)
		}
	}
	return tables
}

// RemoveTableSchemaConfig 删除表创建引擎信息，编码信息，分区信息，已修复同步表结构遇到分区表异常退出问题，
// 对于分区表，只会同步字段，索引，主键，外键的变更
func RemoveTableSchemaConfig(schema string) string {
	return strings.Split(schema, "ENGINE")[0]
}

func (sc *SchemaSync) getAlterDataByTable(table string, cfg *Config) *TableAlterData {
	sSchema := sc.SourceDb.GetTableSchema(table)
	dSchema := sc.DestDb.GetTableSchema(table)
	return sc.getAlterDataBySchema(table, sSchema, dSchema, cfg)
}

func (sc *SchemaSync) getAlterDataBySchema(table string, sSchema string, dSchema string, cfg *Config) *TableAlterData {
	alter := new(TableAlterData)
	alter.Table = table
	alter.Type = alterTypeNo
	alter.SchemaDiff = newSchemaDiff(table, RemoveTableSchemaConfig(sSchema), RemoveTableSchemaConfig(dSchema))

	if sSchema == dSchema {
		return alter
	}
	if len(sSchema) == 0 {
		alter.Type = alterTypeDropTable
		alter.Comment = "源数据库不存在，删除目标数据库多余的表"
		alter.SQL = append(alter.SQL, fmt.Sprintf("drop table `%s`;", table))
		return alter
	}
	if len(dSchema) == 0 {
		alter.Type = alterTypeCreate
		alter.Comment = "目标数据库不存在，创建"
		alter.SQL = append(alter.SQL, fmtTableCreateSQL(sSchema)+";")
		return alter
	}

	diffLines := sc.getSchemaDiff(alter)
	if len(diffLines) == 0 {
		return alter
	}
	alter.Type = alterTypeAlter
	if cfg.SingleSchemaChange {
		for _, line := range diffLines {
			ns := fmt.Sprintf("ALTER TABLE `%s`\n%s;", table, line)
			alter.SQL = append(alter.SQL, ns)
		}
	} else {
		ns := fmt.Sprintf("ALTER TABLE `%s`\n%s;", table, strings.Join(diffLines, ",\n"))
		alter.SQL = append(alter.SQL, ns)
	}

	return alter
}

func (sc *SchemaSync) getSchemaDiff(alter *TableAlterData) []string {
	sourceMyS := alter.SchemaDiff.Source
	destMyS := alter.SchemaDiff.Dest
	table := alter.Table
	var beforeFieldName string
	var alterLines []string
	var fieldCount int = 0
	// 比对字段
	for el := sourceMyS.Fields.Front(); el != nil; el = el.Next() {
		if sc.Config.IsIgnoreField(table, el.Key.(string)) {
			continue
		}
		var alterSQL string
		if destDt, has := destMyS.Fields.Get(el.Key); has {
			if el.Value != destDt {
				alterSQL = fmt.Sprintf("CHANGE `%s` %s", el.Key, el.Value)
			}
			beforeFieldName = el.Key.(string)
		} else {
			if len(beforeFieldName) == 0 {
				if fieldCount == 0 {
					alterSQL = "ADD " + el.Value.(string) + " FIRST"
				} else {
					alterSQL = "ADD " + el.Value.(string)
				}
			} else {
				alterSQL = fmt.Sprintf("ADD %s AFTER `%s`", el.Value.(string), beforeFieldName)
			}
			beforeFieldName = el.Key.(string)
		}

		if len(alterSQL) != 0 {
			alterLines = append(alterLines, alterSQL)
		}
		fieldCount++
	}

	// 源库已经删除的字段
	if sc.Config.Drop {
		for _, name := range destMyS.Fields.Keys() {
			if sc.Config.IsIgnoreField(table, name.(string)) {
				continue
			}
			if _, has := sourceMyS.Fields.Get(name); !has {
				alterSQL := fmt.Sprintf("drop `%s`", name)
				alterLines = append(alterLines, alterSQL)
			}
		}
	}

	// 多余的字段暂不删除

	// 比对索引
	for indexName, idx := range sourceMyS.IndexAll {
		if sc.Config.IsIgnoreIndex(table, indexName) {
			continue
		}
		dIdx, has := destMyS.IndexAll[indexName]
		var alterSQLs []string
		if has {
			if idx.SQL != dIdx.SQL {
				alterSQLs = append(alterSQLs, idx.alterAddSQL(true)...)
			}
		} else {
			alterSQLs = append(alterSQLs, idx.alterAddSQL(false)...)
		}
		if len(alterSQLs) > 0 {
			alterLines = append(alterLines, alterSQLs...)
		}
	}

	// drop index
	if sc.Config.Drop {
		for indexName, dIdx := range destMyS.IndexAll {
			if sc.Config.IsIgnoreIndex(table, indexName) {
				continue
			}
			var dropSQL string
			if _, has := sourceMyS.IndexAll[indexName]; !has {
				dropSQL = dIdx.alterDropSQL()
			}

			if len(dropSQL) != 0 {
				alterLines = append(alterLines, dropSQL)
			}
		}
	}

	// 比对外键
	for foreignName, idx := range sourceMyS.ForeignAll {
		if sc.Config.IsIgnoreForeignKey(table, foreignName) {
			continue
		}
		dIdx, has := destMyS.ForeignAll[foreignName]
		var alterSQLs []string
		if has {
			if idx.SQL != dIdx.SQL {
				alterSQLs = append(alterSQLs, idx.alterAddSQL(true)...)
			}
		} else {
			alterSQLs = append(alterSQLs, idx.alterAddSQL(false)...)
		}
		if len(alterSQLs) > 0 {
			alterLines = append(alterLines, alterSQLs...)
		}
	}

	// drop 外键
	if sc.Config.Drop {
		for foreignName, dIdx := range destMyS.ForeignAll {
			if sc.Config.IsIgnoreForeignKey(table, foreignName) {
				continue
			}
			var dropSQL string
			if _, has := sourceMyS.ForeignAll[foreignName]; !has {
				dropSQL = dIdx.alterDropSQL()
			}
			if len(dropSQL) != 0 {
				alterLines = append(alterLines, dropSQL)
			}
		}
	}

	return alterLines
}

// SyncSQL4Dest sync schema change
func (sc *SchemaSync) SyncSQL4Dest(sqlStr string, sqls []string) error {
	sqlStr = strings.TrimSpace(sqlStr)
	if len(sqlStr) == 0 {
		return nil
	}
	t := newMyTimer()
	ret, err := sc.DestDb.Query(sqlStr)

	defer func() {
		if ret != nil {
			err := ret.Close()
			if err != nil {
				log.Println("close ret error:", err)
				return
			}
		}
	}()

	// how to enable allowMultiQueries?
	if err != nil && len(sqls) > 1 {
		log.Println("exec_mut_query failed, err=", err, ",now exec SQLs foreach")
		tx, errTx := sc.DestDb.Db.Begin()
		if errTx == nil {
			for _, sql := range sqls {
				ret, err = tx.Query(sql)
				if err != nil {
					log.Println("error query_one:[", sql, "]", err)
					break
				}
			}
			if err == nil {
				err = tx.Commit()
			} else {
				_ = tx.Rollback()
			}
		}
	}
	t.stop()
	if err != nil {
		log.Println("EXEC_SQL_FAILED:", err)
		return err
	}
	_, err = ret.Columns()
	return err
}

// check data change
func CheckDiffData(cfg *Config) {
	if len(cfg.TablesCompareData) == 0 {
		fmt.Println("# Tables to CompareData is empty")
		return
	}

	sc := NewSchemaSync(cfg)
	allTables := sc.SourceDb.GetTableNames()
	dstTables := sc.DestDb.GetTableNames()

	dataDiffTables := []string{}

	for _, table := range allTables {
		if !cfg.CheckMatchCompareDataTables(table) {
			continue
		}
		// 目标库没有此表
		if !inStringSlice(table, dstTables) {
			dataDiffTables = append(dataDiffTables, table)
			continue
		}

		// 查询第一个表
		rows1, err := sc.SourceDb.Query(fmt.Sprintf("CHECKSUM TABLE `%s`", table))
		if err != nil {
			log.Fatal("failed to fetch line data", err)
		}
		defer rows1.Close()

		rows2, err := sc.DestDb.Query(fmt.Sprintf("CHECKSUM TABLE `%s`", table))
		if err != nil {
			log.Fatal("failed to fetch line data", err)
		}
		defer rows2.Close()

		// 比较两个表的数据
		var tab1, tab2 string
		var c1, c2 int64

		rows1.Next()
		rows2.Next()
		rows1.Scan(&tab1, &c1)
		rows2.Scan(&tab2, &c2)

		if c1 != c2 {
			dataDiffTables = append(dataDiffTables, table)
			continue
		}
	}

	if len(dataDiffTables) == 0 {
		fmt.Println("# no data of tables in difference")
	} else {
		fmt.Println("# !!!!! data diff tables: ", strings.Join(dataDiffTables, ", "))
	}
}

// check procedures
func CheckAlterProcedure(cfg *Config) {
	sc := NewSchemaSync(cfg)
	allProcedures := sc.SourceDb.GetProcedureNames()

	for _, proceName := range allProcedures {
		srcProcedureStr := sc.SourceDb.GetProcedureSchema(proceName)
		dstProcedureStr := sc.DestDb.GetProcedureSchema(proceName)

		if srcProcedureStr != dstProcedureStr {
			// 不支持直接执行语句 DELIMITER $$
			sqlDrop := fmt.Sprintf("DROP PROCEDURE IF EXISTS `%s`", proceName)

			// 输出
			fmt.Printf("DELIMITER $$\n%s$$\n%s$$\nDELIMITER ;\n\n", sqlDrop, srcProcedureStr)

			// 直接执行同步
			if sc.Config.Sync {
				if _, err := sc.DestDb.Db.Exec(sqlDrop); err != nil {
					log.Fatalln("exec procedure failed", sqlDrop, err)
				}
				if _, err := sc.DestDb.Db.Exec(srcProcedureStr); err != nil {
					log.Fatalln("exec procedure failed", srcProcedureStr, err)
				}
			}
		}
	}
}

// CheckSchemaDiff 执行最终的 diff
func CheckSchemaDiff(cfg *Config) {
	scs := newStatics(cfg)
	defer func() {
		scs.timer.stop()
	}()

	sc := NewSchemaSync(cfg)
	newTables := sc.GetTableNames()
	changedTables := make(map[string][]*TableAlterData)

	for _, table := range newTables {
		if !cfg.CheckMatchTables(table) {
			continue
		}

		if cfg.CheckMatchIgnoreTables(table) {
			continue
		}

		sd := sc.getAlterDataByTable(table, cfg)

		if sd.Type == alterTypeNo {
			continue
		}

		if sd.Type == alterTypeDropTable {
			continue
		}

		fmt.Println(sd)
		fmt.Println("")
		relationTables := sd.SchemaDiff.RelationTables()
		// fmt.Println("relationTables:",table,relationTables)

		// 将所有有外键关联的单独放
		groupKey := "multi"
		if len(relationTables) == 0 {
			groupKey = "single_" + table
		}
		if _, has := changedTables[groupKey]; !has {
			changedTables[groupKey] = make([]*TableAlterData, 0)
		}
		changedTables[groupKey] = append(changedTables[groupKey], sd)
	}

	var countSuccess int
	var countFailed int
	canRunTypePref := "single"

	// 先执行单个表的
runSync:
	for typeName, sds := range changedTables {
		if !strings.HasPrefix(typeName, canRunTypePref) {
			continue
		}
		var sqls []string
		var sts []*tableStatics
		for _, sd := range sds {
			for index := range sd.SQL {
				sql := strings.TrimRight(sd.SQL[index], ";")
				sqls = append(sqls, sql)

				st := scs.newTableStatics(sd.Table, sd, index)
				sts = append(sts, st)
			}
		}

		sql := strings.Join(sqls, ";\n") + ";"
		var ret error

		if sc.Config.Sync {
			ret = sc.SyncSQL4Dest(sql, sqls)
			if ret == nil {
				countSuccess++
			} else {
				countFailed++
			}
		}
		for _, st := range sts {
			st.alterRet = ret
			st.schemaAfter = sc.DestDb.GetTableSchema(st.table)
			st.timer.stop()
		}
	} // end for

	// 最后再执行多个表的 alter
	if canRunTypePref == "single" {
		canRunTypePref = "multi"
		goto runSync
	}

	if sc.Config.Sync {
		log.Println("execute_all_sql_done, success_total:", countSuccess, "failed_total:", countFailed)
	}
}
