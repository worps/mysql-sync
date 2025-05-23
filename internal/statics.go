package internal

import (
	"flag"
)

type statics struct {
	timer  *myTimer
	Config *Config
	tables []*tableStatics
}

type tableStatics struct {
	timer       *myTimer
	table       string
	alter       *TableAlterData
	alterRet    error
	schemaAfter string
}

func newStatics(cfg *Config) *statics {
	return &statics{
		timer:  newMyTimer(),
		tables: make([]*tableStatics, 0),
		Config: cfg,
	}
}

func (s *statics) newTableStatics(table string, sd *TableAlterData, index int) *tableStatics {
	ts := &tableStatics{
		timer: newMyTimer(),
		table: table,
		alter: sd,
	}
	if sd.Type == alterTypeNo {
		return ts
	}
	if s.Config.SingleSchemaChange {
		sds := sd.Split()
		nts := &tableStatics{}
		*nts = *ts
		nts.alter = sds[index]
		s.tables = append(s.tables, nts)
	} else {
		s.tables = append(s.tables, ts)
	}
	return ts
}

func init() {
	flag.StringVar(&htmlResultPath, "html", "", "html result file path")
}

var htmlResultPath string
