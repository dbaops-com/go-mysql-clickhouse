#!/bin/bash
conf='./gomyckInit.cnf'
workDir='/data/gomyck'
ckAddr='tcp://127.0.0.1:9000?debug=false'
mkdir -p ./conf
cat $conf | grep -v '#' | while read line
do
    fileName=`echo $line | awk '{print $1}'`
    fileType=`echo $line | awk '{print $2}'`
    ServerId=`echo $line | awk '{print $3}'`
    HOST=`echo $line | awk '{print $4}'`
    PORT=`echo $line | awk '{print $5}'`
    USER=`echo $line | awk '{print $6}'`
    PASS=`echo $line | awk '{print $7}'`
    dbname=`echo $line | awk '{print $8}'`
    tbname=`echo $line | awk '{print $9}'`
    dstname=`echo $line | awk '{print $10}'`
    tmpCnf="[mysql]\nuser=${USER}\nhost=${HOST}\nport=${PORT}\npassword=${PASS}"
    printf "$tmpCnf" > ./.recreate.mycnf
    tableInfoSQL="
SET @db_name = '${dbname}';
SET @tb_name = '${tbname}';
SET @dstname = '${dstname}';
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
        'ENGINE = MergeTree()',
        'PARTITION BY toStartOfMonth(toDate(BinlogTime)) ORDER BY (BinlogTime) SETTINGS index_granularity = 8192;'
    ) AS createSQL INTO @columnList, @posString, @dataString, @createSQL
FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = @tb_name;
SELECT CONCAT('DTKey=',@db_name,'.',@tb_name,'///INSERT INTO ',@dstname,'(',@columnList,') VALUES (', @posString,')') AS contents
UNION ALL
SELECT CONCAT('DTKey=',@db_name,'.',@tb_name,'-ColType///',@dataString)
UNION ALL
SELECT @createSQL;"
    tableInfo=`mysql --defaults-file=./.recreate.mycnf -N -e "${tableInfoSQL}" | head -n 2`
    createSQL=`mysql --defaults-file=./.recreate.mycnf -N -e "${tableInfoSQL}" | tail -n 1`
    echo "${createSQL}" > ./conf/createTable.sql

    confString="batchSize=50000
gtidFile=./${fileName}.gtid
ServerID=${ServerId}
Host=${HOST}
Port=${PORT}
User=${USER}
Password=${PASS}
CKStr=${ckAddr}"
    superConf="[program:${fileName}]
directory=${workDir}
command=${workDir}/go-mysql-clickhouse --conf ${workDir}/${fileName}.cnf
autostart=true
autorestart=true
startsecs=10
stdout_logfile=${workDir}/${fileName}.log
stdout_logfile_maxbytes=100MB
stdout_logfile_backups=2
stdout_capture_maxbytes=1MB
stderr_logfile=${workDir}/${fileName}.err
stderr_logfile_maxbytes=100MB
stderr_logfile_backups=2
stderr_capture_maxbytes=1MB
user = root"
    # Create Configration File
    if [ "${fileType}" == "NewFile" ];then
        #echo "$superConf" > "/etc/supervisor/conf.d/gomyck-${fileName}.conf"
        echo "$superConf" > ./conf/gomyck-${fileName}.conf
        echo "${fileName}"
        echo "${confString}" > ./conf/${fileName}.cnf
        echo "${tableInfo}" >> ./conf/${fileName}.cnf
    elif [ "${fileType}" == "Append" ];then
        echo "${tableInfo}" >> ./conf/${fileName}.cnf
    fi
    # GTID
    gtidInfoSQL='show master status;'
    #gtidInfo=`mysql -N -h ${HOST} -P${PORT} -u${USER} -p${PASS} -e "${gtidInfoSQL}" | tail -n 1`
    gtidInfo=`mysql --defaults-file=./.recreate.mycnf -e "${gtidInfoSQL}" | tail -n 1 | awk '{print $3}'`
    printf "${gtidInfo}" > ./conf/${fileName}.gtid
done
