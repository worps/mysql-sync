package internal

import (
	"encoding/json"
	"log"
	"strings"
)

// Config  config struct
type Config struct {
	// SourceDSN 同步的源头
	SourceDSN string `json:"source"`
	SourceSSH string `json:"source_ssh"`

	// DestDSN 将被同步
	DestDSN string `json:"dest"`
	DestSSH string `json:"dest_ssh"`

	ConfigPath string

	// 要同步的数据库
	Schemas []string `json:"schemas"`

	// Tables 同步表的白名单，若为空，则同步全库
	Tables []string `json:"tables"`

	// TablesIgnore 不同步的表
	TablesIgnore []string `json:"tables_ignore"`

	TablesCompareData []string `json:"tables_compare_data"`

	// Sync 是否真正的执行同步操作
	Sync bool

	// Drop 若目标数据库表比源头多了字段、索引，是否删除
	Drop bool

	// SingleSchemaChange 生成sql ddl语言每条命令只会进行单个修改操作
	SingleSchemaChange bool `json:"single_schema_change"`
}

func (cfg *Config) String() string {
	ds, _ := json.MarshalIndent(cfg, "  ", "  ")
	return string(ds)
}

// CheckMatchTables check table is match
func (cfg *Config) CheckMatchTables(name string) bool {
	// 若没有指定表，则意味对全库进行同步
	if len(cfg.Tables) == 0 {
		return true
	}
	for _, tableName := range cfg.Tables {
		if simpleMatch(tableName, name, "CheckMatchTables") {
			return true
		}
	}
	return false
}

func (cfg *Config) SetTables(tables []string) {
	for _, name := range tables {
		name = strings.TrimSpace(name)
		if len(name) > 0 {
			cfg.Tables = append(cfg.Tables, name)
		}
	}
}

// SetTablesIgnore 设置忽略
func (cfg *Config) SetTablesIgnore(tables []string) {
	for _, name := range tables {
		name = strings.TrimSpace(name)
		if len(name) > 0 {
			cfg.TablesIgnore = append(cfg.TablesIgnore, name)
		}
	}
}

// set tables to compare data，* is supported
func (cfg *Config) SetTablesCompareData(tables []string) {
	for _, name := range tables {
		name = strings.TrimSpace(name)
		if len(name) > 0 {
			cfg.TablesCompareData = append(cfg.TablesCompareData, name)
		}
	}
}

// CheckMatchIgnoreTables check TablesCompareData is match
func (cfg *Config) CheckMatchCompareDataTables(name string) bool {
	if len(cfg.TablesCompareData) == 0 {
		return false
	}
	for _, tableName := range cfg.TablesCompareData {
		if simpleMatch(tableName, name, "CheckMatchCompareDataTables") {
			return true
		}
	}
	return false
}

// CheckMatchIgnoreTables check table_Ignore is match
func (cfg *Config) CheckMatchIgnoreTables(name string) bool {
	if len(cfg.TablesIgnore) == 0 {
		return false
	}
	for _, tableName := range cfg.TablesIgnore {
		if simpleMatch(tableName, name, "CheckMatchTables") {
			return true
		}
	}
	return false
}

// Check check config
func (cfg *Config) Check() {
	if len(cfg.SourceDSN) == 0 {
		log.Fatal("source DSN is empty")
	}
	if len(cfg.DestDSN) == 0 {
		log.Fatal("dest DSN is empty")
	}
	if len(cfg.Schemas) <= 0 {
		log.Fatal("Schemas is empty")
	}
}

// LoadConfig load config file
func LoadConfig(confPath string) *Config {
	var cfg *Config
	err := loadJSONFile(confPath, &cfg)
	if err != nil {
		log.Fatalln("load json conf:", confPath, "failed:", err)
	}
	cfg.ConfigPath = confPath
	return cfg
}
