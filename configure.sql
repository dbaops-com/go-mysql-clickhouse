#配置文件关键信息生成SQL
SELECT 
concat('SQLType,BinlogTime,ParseTime, ',group_concat(column_name ORDER BY ORDINAL_POSITION ASC)),
concat('?,?,?,',group_concat(if(length(column_name)>0, '?',''))),
CONCAT('0,0,1,',
GROUP_CONCAT(IF(column_type LIKE '%int%', 1, IF(column_type LIKE '%decimal%', 2, 0)) 
             ORDER BY ORDINAL_POSITION ASC SEPARATOR ',' )) AS dataSQL, 
CONCAT('CREATE TABLE IF NOT EXISTS ', table_name, '_shard ( ',
GROUP_CONCAT(CONCAT(column_name, ' ', 
IF(column_type LIKE '%int%','UInt64',
IF(column_type LIKE '%decimal%','String',
IF(column_type LIKE '%char%','String',''))))
ORDER BY ORDINAL_POSITION ASC SEPARATOR ','),
', SQLType String, BinlogTime String, ParseTime UInt64) 
ENGINE = ReplicatedMergeTree(\'/clickhouse/tables/{layer}-{shard}/tttttt_replica\', \'{replica}\') 
PARTITION BY toDayOfMonth(toDate(uuuuuu / 1000)) ORDER BY (oooooo) SETTINGS index_granularity = 8192
;') AS createSQL
FROM information_schema.columns WHERE table_schema = 'ssssss' AND table_name = 'tttttt';
需修改如下参数：
ssssss：schema_name
tttttt：table_name
oooooo：orderby_column_list
uuuuuu：Unixtime_column
