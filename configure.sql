#SQL1: 配置文件关键信息生成SQL
SET @db_name = 'schema_name';
SET @tb_name = 'table_name';
SET @dstname = 'dst_dbname.dst_tbname';
SELECT 
    CONCAT('SQLType,BinlogTime,ParseTime,ServerId,BinlogXid,BinlogFile,BinlogPos,',group_concat(column_name ORDER BY ORDINAL_POSITION ASC)) AS columnList,
    CONCAT('?,?,?,?,?,?,?,',group_concat(if(length(column_name)>0, '?',''))) AS posString,
    CONCAT('0,0,1,0,0,0,0,',
    GROUP_CONCAT(IF(column_type LIKE '%int%', 1, IF(column_type LIKE '%decimal%', 2, 0)) 
                 ORDER BY ORDINAL_POSITION ASC SEPARATOR ',' )) AS dataString, 
    CONCAT('CREATE TABLE IF NOT EXISTS ', @dstname, ' ( ',
        GROUP_CONCAT(CONCAT(column_name, ' ', 
            IF(column_type LIKE '%int%',    'UInt64',
            IF(column_type LIKE '%decimal%','String',
            IF(column_type LIKE '%char%',   'String',''))))
            ORDER BY ORDINAL_POSITION ASC SEPARATOR ','),
        ', SQLType String, BinlogTime String, ServerId String, ParseTime UInt64, BinlogXid String, BinlogFile String, BinlogPos String)',
        #'ENGINE = ReplicatedMergeTree(\'/clickhouse/tables/{layer}-{shard}/tttttt_replica\', \'{replica}\'',
	    'ENGINE = MergeTree()',
        'PARTITION BY toDayOfMonth(toDate(BinlogTime)) ORDER BY (BinlogTime) SETTINGS index_granularity = 8192;'
	) AS createSQL INTO @columnList, @posString, @dataString, @createSQL
FROM information_schema.columns 
WHERE table_schema = @db_name AND table_name = @tb_name;
SELECT CONCAT('DTKey=',@db_name,'.',@tb_name,'///INSERT INTO ',@dstname,'(',@columnList,') VALUES (', @posString,')')
UNION ALL
SELECT CONCAT('DTKey=',@db_name,'.',@tb_name,'-ColType///',@dataString)
UNION ALL
SELECT @createSQL;

#SQL2: 查询最新数据
SELECT 
    argMax(id, ParseTime) as ids,
    argMax(k, ParseTime) as ks,
    argMax(c, ParseTime) as cs,
    argMax(pad, ParseTime) as pads,
    argMax(BinlogTime, ParseTime) as BinlogTimes,
    argMax(SQLType, ParseTime) as SQLTypes,
    argMax(ServerId, ParseTime) as SQLTypes
FROM sbtest
WHERE SQLType IN ('insert','update')
GROUP BY id
ORDER BY id DESC LIMIT 100;

#SQL3: Clickhouse字段改名
alter table dbname.tbname add column ServerId String after ParseTime;
alter table dbname.tbname update ServerId = serverId where 1=1;
alter table dbname.tbname drop column serverId;
