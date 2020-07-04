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

vim createconf.cnf

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
shell>> vim createconf.sh 修改：
        workDir='/data/gomyck'                    # 工作目录
        ckAddr='tcp://127.0.0.1:9000?debug=false' # clickhouse连接地址

shell>> bash createconf.sh                        # 会生成对应的配置文件

shell>> mv ./gomyck-* /etc/supervisor/conf.d/     # 复制supervisor配置文件到目标目录

shell>> mv ./*.cnf /workDir/                      # 移动gomyck配置文件到目标目录

shell>> mv ./*.gtid /workDir/                     # 移动gtid文件到目标目录
```

2.5、完成配置

## 启动
1、进入supervisor：supervisorctl

    ```
    supervisorctl reload
    supervisorctl status
    supervisorctl start all
    ```
    
## 使用场景

1、分库分表的数据查询聚合

2、binlog分析

3、TPS监控

QQ群：192815465
