# mysql-schema-sync

MySQL Schema 自动同步工具  
功能强大，用于命令行方式进行两个数据库之间的结构同步。同时支持对表进行表数据比较。

支持功能：  

1. 同步**新表**  
2. 同步**字段** 变动：新增、修改  
3. 同步**索引** 变动：新增、修改
4. 同步**存储过程**
4. 支持**预览**（只对比不同步变动）  
5. 对比**数据差异**
6. 支持屏蔽更新**表、字段、索引、外键**  
7. 支持本地比线上额外多一些表、字段、索引、外键

## 配置示例(conf.json):  

```
{
      // 同步源
      "source":"test:test@(127.0.0.1:3306)/test_0",
      // 待同步到的数据库
      "dest":"test:test@(127.0.0.1:3306)/test_1",
      // 同步时忽略的字段和索引，键名为表名
      "alter_ignore":{
        "tb1*":{
            "column":["aaa","a*"],
            "index":["aa"],
            "foreign":[]
        }
      },
      //  要检查的表，默认所有表，支持通配符
      "tables":[],
      // 要忽略的表，支持通配符
      "tables_ignore": [],
      // 要进行数据比较的表，会将内容存在差异的表名以注释的形式输出，注意查看
      "tables_compare_data":["sys_*"]
}
```

### 直接进行同步

```shell
go run main.go -conf conf.json -sync
```

### 生成变更sql

```shell
go run main.go -drop -conf conf.json 2>/dev/null >db_alter.sql

```

### 运行参数说明

```shell
go run main.go [-conf] [-dest] [-source] [-sync] [-drop]
```

说明：

```shell
go run main.go -help  
  -conf string
        配置文件名称
  -dest string
        待同步的数据库 eg: test@(10.10.0.1:3306)/test_1
        该项不为空时，忽略读入 -conf参数项
  -drop
        是否对本地多出的字段和索引进行删除 默认否
  -source string
        mysql 同步源,eg test@(127.0.0.1:3306)/test_0
  -sync
        是否将修改同步到数据库中去，默认否
  -tables string
        待检查同步的数据库表，为空则是全部
        eg : product_base,order_*
  -single_schema_change
        生成 SQL DDL 语言每条命令是否只会进行单个修改操作，默认否
```
