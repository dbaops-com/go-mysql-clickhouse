# go-mysql-clickhouse

从MySQL实时解析数据写入clickhouse，以达到实时分析的目的

## 特性

1、解析MySQL binlog，改写UPDATE、DELETE为INSERT，并附带上原始变更的类型（SQLType）写入到Clickhouse

2、保留MySQL binlog的元数据信息

    ```
    BinlogTime：binlog数据生成时间
    ServerId  ：当前伪装的Server Id
    ParseTime ：解析每行数据的时间戳
    BinlogXid ：binlog的Xid，因为结构原因，与binlog文件中Xid上下错了一行，不影响分析
    BinlogFile：当前数据所处的binlog文件
    BinlogPos ：当前数据所处的binlog文件的position位置，方便数据查找
    ```
获取最新数据的SQL:

    ```
    SELECT 
        argMax(id,         ParseTime) as ids,
        argMax(k,          ParseTime) as ks,
        argMax(c,          ParseTime) as cs,
        argMax(pad,        ParseTime) as pads,
        argMax(BinlogTime, ParseTime) as BinlogTimes,
        argMax(SQLType,    ParseTime) as SQLTypes,
        argMax(ServerId,   ParseTime) as SQLTypes
    FROM sbtest
    WHERE SQLType IN ('insert','update')
    GROUP BY id
    ORDER BY id DESC LIMIT 100;
    ```
    
## 部署

0、创建工作目录

0.1、mkdir -p /data/gomyck

1、使用supervisor来管理go-mysql-clickhouse的启动和日志

1.1、apt-get install python-setuptools

1.2、sudo easy_install supervisor

[详见：安装supervisor](https://cloudwafer.com/blog/how-to-install-and-configure-supervisor-on-ubuntu-16-04/)

注意：默认的supervisor配置文件存放到/etc/supervisor/conf.d/目录下

2、生成配置文件

2.1、部署MySQL账号

    ```
    CREATE USER 'chenxinglong'@'%' IDENTIFIED BY 'Cxl.123456';
    CREATE USER 'chenxinglong'@'127.0.0.1' IDENTIFIED BY 'Cxl.123456';
    CREATE USER 'chenxinglong'@'localhost' IDENTIFIED BY 'Cxl.123456';
    GRANT SELECT, REPLICATION CLIENT, REPLICATION SLAVE ON *.* TO 'chenxinglong'@'%';
    GRANT SELECT, REPLICATION CLIENT, REPLICATION SLAVE ON *.* TO 'chenxinglong'@'127.0.0.1';
    GRANT SELECT, REPLICATION CLIENT, REPLICATION SLAVE ON *.* TO 'chenxinglong'@'localhost';
    FLUSH PRIVILEGES;
    ```
    
2.2、部署Clickhouse账号

    ```
    tcp://127.0.0.1:9000?debug=false
    ```
    
2.3、编写账号文件

vim gomyckInit.cnf

    ```
    #实例名称编号     文件类型(新建/附加) 实例ID       主机地址            主机端口  主机用户       主机密码     源库名称   源表名称     目标数据库
    #InstanceName    FileType           ServerId    HOST               PORT      USER          PASS        dbname    tbname       dstname
    Class_01         NewFile            10001       class1.dbaops.com  3306      chenxinglong  Cxl.123456  class_1    gmc_user    ods.gmc_user_group
    Class_01         Append             10001       class1.dbaops.com  3306      chenxinglong  Cxl.123456  class_1    gmc_class   ods.gmc_class_group
    Class_02         NewFile            10002       class2.dbaops.com  3306      chenxinglong  Cxl.123456  class_2    gmc_user    ods.gmc_user_group
    Class_02         Append             10002       class2.dbaops.com  3306      chenxinglong  Cxl.123456  class_2    gmc_class   ods.gmc_class_group
    Class_03         NewFile            10003       class3.dbaops.com  3306      chenxinglong  Cxl.123456  class_3    gmc_user    ods.gmc_user_group
    Class_03         Append             10003       class3.dbaops.com  3306      chenxinglong  Cxl.123456  class_3    gmc_class   ods.gmc_class_group
    Class_04         NewFile            10004       class4.dbaops.com  3306      chenxinglong  Cxl.123456  class_4    gmc_user    ods.gmc_user_group
    Class_04         Append             10004       class4.dbaops.com  3306      chenxinglong  Cxl.123456  class_4    gmc_class   ods.gmc_class_group
    ``
    
以上配置就从4个实例中class1、class2、class3、class4，把gmc_user和gmc_class表分别汇总到ods.gmc_user_group和ods.gmc_class_group表

2.4、生成配置文件
```
shell>> vim gomyckInit.sh 修改：
        workDir='/data/gomyck'                      # 工作目录
        ckAddr='tcp://127.0.0.1:9000?debug=false'   # clickhouse连接地址

shell>> bash gomyckInit.sh                          # 会生成对应的配置文件

shell>> mv ./conf/gomyck-* /etc/supervisor/conf.d/  # 复制supervisor配置文件到目标目录

shell>> mv ./conf/*.cnf    /data/gomyck             # 移动gomyck配置文件到目标目录

shell>> mv ./conf/*.gtid   /data/gomyck             # 移动gtid文件到目标目录
```

2.5、完成配置

## 启动
    ```
    supervisorctl reload
    supervisorctl status
    supervisorctl start all
    ```
    
## 使用场景

0、样例数据
```
// binlog元数据信息
SQL> select id,k,SQLType,BinlogTime,ServerId,ParseTime,BinlogXid,BinlogFile,BinlogPos from sbtest_group order by ParseTime desc limit 10;

┌────id─┬─────k─┬─SQLType─┬─BinlogTime──────────┬─ServerId─┬───────────ParseTime─┬─BinlogXid─┬─BinlogFile───────┬─BinlogPos─┐
│ 50042 │ 46928 │ insert  │ 2020-07-04 10:51:20 │ 10001    │ 1593831084326306330 │ 290284617 │ mysql-bin.000717 │ 795189141 │
│ 50042 │ 50074 │ delete  │ 2020-07-04 10:51:20 │ 10001    │ 1593831084326299224 │ 290284617 │ mysql-bin.000717 │ 795189141 │
│ 50112 │ 60509 │ update  │ 2020-07-04 10:51:20 │ 10001    │ 1593831084326291951 │ 290284617 │ mysql-bin.000717 │ 795189141 │
│ 50483 │ 56540 │ update  │ 2020-07-04 10:51:20 │ 10001    │ 1593831084326280424 │ 290284617 │ mysql-bin.000717 │ 795189141 │
│ 50105 │ 50081 │ insert  │ 2020-07-04 10:51:20 │ 10001    │ 1593831084326273171 │ 290284467 │ mysql-bin.000717 │ 795187443 │
│ 50105 │ 49877 │ delete  │ 2020-07-04 10:51:20 │ 10001    │ 1593831084326265413 │ 290284467 │ mysql-bin.000717 │ 795187443 │
│ 50175 │ 49903 │ update  │ 2020-07-04 10:51:20 │ 10001    │ 1593831084326258313 │ 290284467 │ mysql-bin.000717 │ 795187443 │
│ 50023 │ 50306 │ update  │ 2020-07-04 10:51:20 │ 10001    │ 1593831084326249000 │ 290284467 │ mysql-bin.000717 │ 795187443 │
│ 50226 │ 50480 │ insert  │ 2020-07-04 10:51:20 │ 10001    │ 1593831084326241758 │ 290284546 │ mysql-bin.000717 │ 795185745 │
│ 50226 │ 46207 │ delete  │ 2020-07-04 10:51:20 │ 10001    │ 1593831084326232398 │ 290284546 │ mysql-bin.000717 │ 795185745 │
└───────┴───────┴─────────┴─────────────────────┴──────────┴─────────────────────┴───────────┴──────────────────┴───────────┘
10 rows in set. Elapsed: 0.119 sec. Processed 1.58 million rows, 224.60 MB (13.26 million rows/s., 1.88 GB/s.)

// 查询最大的事务
SQL> select BinlogXid,count(*) c from sbtest_group group by BinlogXid order by c desc limit 100;
┌─BinlogXid─┬─c─┐
│ 283880837 │ 4 │
│ 284301837 │ 4 │
│ 282831800 │ 4 │
│ 286678522 │ 4 │
│ 290189486 │ 4 │
│ 284861368 │ 4 │
│ 283856989 │ 4 │
└───────────┴───┘
100 rows in set. Elapsed: 0.139 sec. Processed 1.58 million rows, 28.47 MB (11.42 million rows/s., 205.48 MB/s.)

// 检查同一行数据重复写
SQL> select id, count(*) c from sbtest_group group by id order by c desc limit 10;
┌────id─┬────c─┐
│ 50313 │ 1735 │
│ 50412 │ 1731 │
│ 49901 │ 1729 │
│ 50024 │ 1717 │
│ 49964 │ 1714 │
│ 49961 │ 1713 │
│ 50013 │ 1711 │
│ 50332 │ 1705 │
│ 49800 │ 1705 │
│ 49815 │ 1703 │
└───────┴──────┘

10 rows in set. Elapsed: 0.032 sec. Processed 1.58 million rows, 12.65 MB (48.90 million rows/s., 391.18 MB/s.)
```
1、分库分表的数据查询聚合

2、binlog分析

3、TPS监控

QQ群：192815465
