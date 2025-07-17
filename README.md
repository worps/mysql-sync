# mysql-sync

MySQL Schema 自动同步工具  
功能强大，用于命令行方式进行两个数据库之间的结构同步。同时支持对表进行表数据比较。

支持功能：  

1. 同步**新表**  
2. 同步**字段** 变动：新增、修改  
3. 同步**索引** 变动：新增、修改
4. 同步**存储过程**
4. 支持**预览**（只对比不同步变动）  
5. 对比**数据差异**
6. 对比同一条sql语句在两个数据库中的执行结果
7. 根据sql文件向目标库导入sql

### 配置示例(conf.json):  

```
{
      // 同步源
      "source":"test:test@127.0.0.1:3306",
      //（可选）同步源在内网，支持ssh通道连接mysql，支持通过私钥连接ssh
      "source_ssh":"root:passwd123@14.xx.xx.xx:22",
      // 目标源
      "dest":"test:test@127.0.0.1:3308",
      //（可选）目标源在内网，支持ssh通道连接mysql，支持通过私钥连接ssh
      "dest_ssh":"root@14.xx.xx.xx:22/data/default.key",
      // 要处理的数据库名
      "schemas": ["game_config_db"],
      // 要检查的表，默认所有表，支持通配符
      "tables":[],
      // 要忽略的表，支持通配符
      "tables_ignore": [],
      // 要进行数据比较的表，会将内容存在差异的表名以注释的形式输出，注意查看
      "tables_compare_data":["sys_*"]
}
```
### 编译
```shell
go build -tags netgo -ldflags '-w -s -extldflags "-static"' -o sync.exe .\main.go
```

### 直接进行同步

```shell
sync.exe -conf conf.json -sync
```

### 生成变更sql

```shell
sync.exe -drop -conf conf.json >db_alter.sql
```

### 对比查询结果
```shell
sync.exe -conf conf.json -sql_check "select count(1) as cc from game_main_db.club"
```

### 导入sql文件到目标库
sync.exe -conf conf.json -sql_file ./data.sql

### 运行参数说明

```shell
sync.exe -help  
      -conf
            配置文件名称
      -drop
            是否对本地多出的字段和索引进行删除 默认否
      -sync
            是否将修改同步到数据库中去，默认否
      -sql_check
            检查sql语句在两个库的执行结果
      -sql_file
            导入sql文件到目标库
```
