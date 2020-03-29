package main

import (
    "context"
    "os"
    //"io"
    "strconv"
    "strings"
    "bytes"
    "bufio"
    "fmt"
    "log"
    "time"
    "flag"
    "io/ioutil"
    "database/sql"
    "github.com/siddontang/go-mysql/mysql"
    "github.com/siddontang/go-mysql/replication"
    "github.com/ClickHouse/clickhouse-go"
    "github.com/shopspring/decimal"
    //valid "github.com/asaskevich/govalidator"
)
var (
    linePrint     = "false"
    lineNum       map[string]int
    lineResult    map[string]string
    lastData      map[string]string
    mybatchSize   int    = 50000
    myServerID    int    = 0
    myHost        string = "127.0.0.1"
    myPort        int    = 3306
    myUser        string = "chenxinglong"
    myPassword    string = "Cxl.123456"
    myCKStr       string = "tcp://127.0.0.1:9000?debug=false"
    gtidSet       string
    gtidFile      string = "./gomyck.gtid"
    lastParseMs   int64  = int64(0)
    lastCommitMs  int64  = int64(0)
    curSecondMs   int64  = int64(0)
    DTNameCnf     map[string]string
    connect       *sql.DB
)
func checkError(err error) {
    if err != nil{
        fmt.Println(err)
        //os.Exit(1)
    }
}
func StrToInt64(tmpStr string) int{
    tmpInt,err := strconv.Atoi(tmpStr)
    checkError(err)
    return tmpInt
}

func Int64ToStr(tmpInt int64) string{
    return strconv.FormatInt(tmpInt, 10)
}
func ReadFile(dsnFile string) string {
    tmpLines, err := ioutil.ReadFile(dsnFile)
    checkError(err)
    return string(tmpLines)
}
func writeToFile(msg string)  {
    if Exist(gtidFile) == false {
        f,err := os.Create(gtidFile)
        defer f.Close()
        checkError(err)
    }
    err := ioutil.WriteFile(gtidFile, []byte(msg), 777)
    checkError(err)
}
func SaveData(tmpDTName string){
    tmpDate := ""
    tmpSQLType := ""
    tmpGTIDStr := ""
    tmpMaxNumKey := tmpDTName + "-MaxNum"
    tmpMaxNum := lineNum[tmpMaxNumKey]
    tmpColNum := lineNum[tmpDTName + "-ColNum"]
    tmpSQLValueStr:= ""
    if tmpMaxNum < 1 {
        return
    }
    if err := connect.Ping(); err != nil {
        if exception, ok := err.(*clickhouse.Exception); ok {
            fmt.Printf("[%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
        } else {
            fmt.Println(err)
        }
        return
    }
    var (
        tx, _   = connect.Begin()
        stmt, _ = tx.Prepare(DTNameCnf[tmpDTName])
    )
    defer stmt.Close()
    // Create Conn EOF
    //var tmpSQLValue [tmpColNum]string
    extend_ColNum := 3
    tmpSQLValue := make([]string, tmpColNum + extend_ColNum)
    //SQLTail:=");"
    for i:=10; i<=tmpMaxNum; i=i+10 {
        tmpSQLValueStr = lineResult[tmpDTName + "-Value-" + strconv.Itoa(i)]
        //fmt.Println(tmpSQLValueStr)
        tmpLineSlice := strings.SplitN(tmpSQLValueStr,"--", 2)
        for _, tmpLine := range strings.Split(tmpLineSlice[0], "\n"){
            //fmt.Println(tmpLine)
            lineSlice := strings.SplitN(tmpLine, ":", 2)
            if lineSlice[0] == "Date" {
                tmpDate = lineSlice[1]
            } else if lineSlice[0] == "GTID" {
                tmpGTIDStr = lineSlice[1]
            } else if lineSlice[0] == "ColNum" {
                tmpColNum = StrToInt64(lineSlice[1])
            } else if lineSlice[0] == "Type" {
                tmpSQLType = lineSlice[1]
            }
        }
        //SQLType
        tmpSQLValue[0] = tmpSQLType
        //BinglogTime
        tmpSQLValue[1] = tmpDate
        //ParseTime
        //fmt.Println(tmpSQLValue)
        tmpSQLValue[2] = Int64ToStr(time.Now().UnixNano())
        for _, tmpLine := range strings.Split(tmpLineSlice[1], "\n"){
            tmpLineSlice := strings.SplitN(tmpLine,":",2)
            if len(tmpLineSlice) > 1 {
                tmpSQLValue[StrToInt64(tmpLineSlice[0])+extend_ColNum]=tmpLineSlice[1]
            }
        }
        getColTypeKey := tmpDTName + "-ColType"
        tmpColType := strings.Split(DTNameCnf[getColTypeKey],",")
        Vinterface := make([]interface{}, len(tmpSQLValue))
        for i, str := range tmpSQLValue {
            if tmpColType[i] == "1" {
                Vinterface[i] = StrToInt64(str)
            } else if tmpColType[i] == "2"{
                tmpDecimal, err := decimal.NewFromString(str)
                checkError(err)
		Vinterface[i] = tmpDecimal.String()
            } else {
                Vinterface[i] = strings.ReplaceAll(str,"\"","")
            }
        }
        _, err := stmt.Exec(Vinterface...)
        checkError(err)
    }
    // Commit Data
    if err := tx.Commit(); err != nil {
         log.Fatal(err)
     }
    // fmt.Println(tmpSQLValue)
    curTime := time.Now()
    binlogInt, err := time.Parse("2006-01-02 15:04:05", strings.ReplaceAll(tmpDate, "\"",""))
    checkError(err)
    duration := strconv.FormatInt(int64(curTime.Unix())+ int64(28800) - int64(binlogInt.Unix()), 10)
    fmt.Println(curTime.Format("2006-01-02T15:04:05.000") + " =>Last Commit[" + tmpDate +"](Lag: "+ duration +"): " + strconv.Itoa(lineNum[tmpMaxNumKey]/10))
    lineNum[tmpMaxNumKey] = 0
    lastCommitMs = time.Now().UnixNano()/int64(1000000)

    gtidSet := tmpGTIDStr
    if len(gtidSet) > 1{
        writeToFile(gtidSet)
    }
}
func ParseData(){
    cfg := replication.BinlogSyncerConfig{
        ServerID: uint32(myServerID),
        Flavor:   "mysql",
        Host:     myHost,
        Port:     uint16(myPort),
        User:     myUser,
        Password: myPassword,
//        UseDecimal: true,
    }
    syncer := replication.NewBinlogSyncer(cfg)

    // Start sync with specified binlog file and position
    //streamer, _ := syncer.StartSync(mysql.Position{"mysql-bin.000659", 8948850})
    tmpGTID, _ := mysql.ParseGTIDSet("mysql", gtidSet)
    streamer, err := syncer.StartSyncGTID(tmpGTID)
    checkError(err)
    // or you can start a gtid replication like
    // streamer, _ := syncer.StartSyncGTID(gtidSet)
    // the mysql GTID set likes this "de278ad0-2106-11e4-9f8e-6edd0ca20947:1-2"
    // the mariadb GTID set likes this "0-1-100"
    lineNum    = make(map[string]int)
    lineResult = make(map[string]string)
    lastData   = make(map[string]string)
    TableIndex   := [4]int{1, 7, 8, 9}
    tmpDTName    := ""
    eventRowNum  := 0
    //tmpRowNum    := 0
    tmpDate      := ""
    tmpColNum    := ""
    tmpSQLValue  := ""
    tmpSQLType   := ""
    tmpGTIDStr   := ""
    preLag1ns    := int64(0)
    preLag2ns    := int64(0)
    preLag3ns    := int64(0)
    preLagTime   := int64(0)
    pre1ns       := int64(0)
    //tmpSecond    := int64(0)
    //DTNameCnf["sysbench.sbtest1"]="INSERT INTO default.sbtest1(sql_type,time_ns,id,k,c,pad) VALUES (?,?,?,?,?,?)"
    //DTNameCnf["sysbench.sbtest1-ColType"]="0,0,1,1,0,0"
    for {
        lag1ns := time.Now().UnixNano()
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        lag2ns := time.Now().UnixNano()
        ev, err := streamer.GetEvent(ctx)
        lag3ns := time.Now().UnixNano()
        cancel()
        curSecondMs = time.Now().UnixNano()/int64(1000000)
        if curSecondMs - lastCommitMs > int64(5000) {
            for tmpDTName := range DTNameCnf{
                SaveData(tmpDTName)
            }
        }
        if err == context.DeadlineExceeded {
            // meet timeout
            continue
        }
        //ev.Dump(os.Stdout)
        buf := new(bytes.Buffer)
        ev.Dump(buf)
        lines := buf.String()
        textSlice := strings.SplitN(lines,"===",3)
        lineTitle := lines[4:13]
        if linePrint == "true" && lineTitle != "Heartbeat" {
            fmt.Println(">>>>>>>>>>>>")
            fmt.Println(lines)
            fmt.Println("<<<<<<<<<<<<")
        }
        //TableMapE
        //QueryEven
        //UpdateRow
        //WriteRows
        //DeleteRow
        if (lineTitle == "TableMapE") {
            tmpLineSlice := strings.Split(textSlice[2],"\n")
            for _, j := range TableIndex {
                lineSlice := strings.SplitN(tmpLineSlice[j], ":", 2)
                if len(lineSlice) > 1 {
                    key  := lineSlice[0]
                    value:= strings.Replace(strings.Trim(lineSlice[1]," "),"\n","",-1)
                    lastData[key] = value
                }
            }
            //metric
            //fmt.Println(lastData)
            tmpDTName = lastData["Schema"] + "." + lastData["Table"]
            tmpDate   = lastData["Date"]
            tmpColNum = lastData["Column count"]
            lineNum[tmpDTName + "-ColNum"] = StrToInt64(lastData["Column count"])
        //} else if strings.Contains(lineTitle,"RowQueryEvent") {
        //  continue
        } else if (lineTitle == "QueryEven") && len(DTNameCnf[lastData["Schema"]+"."+lastData["Table"]]) > 1 {
            tmpLineSlice := strings.Split(textSlice[2],"\n")
            if len(tmpLineSlice) >=9 && strings.Contains(tmpLineSlice[9],"GTIDSet") {
                lineSlice := strings.SplitN(tmpLineSlice[9], ":", 2)
                if len(lineSlice) > 1 {
                    key  := lineSlice[0]
                    value:= strings.Replace(strings.Trim(lineSlice[1]," "),"\n","",-1)
                    lastData[key] = value
                }
            }
            //metric
            tmpDate   = lastData["Date"]
            tmpGTIDStr= lastData["GTIDSet"]
        } else if (lineTitle == "UpdateRow" || lineTitle == "WriteRows" || lineTitle == "DeleteRow") && len(DTNameCnf[lastData["Schema"]+"."+lastData["Table"]]) > 1 {
            if lineTitle == "UpdateRow" {
                tmpSQLType = "update"
            } else if lineTitle == "WriteRows"{
                tmpSQLType = "insert"
            } else if lineTitle == "DeleteRow"{
                tmpSQLType = "delete"
            }
            tmpSQLValue = "Date:"+tmpDate+"\nColNum:"+tmpColNum+"\nGTID:"+tmpGTIDStr+"\nType:"+tmpSQLType+"\n"
            tmpLineSlice := strings.Split(lines,"--")
            eventRowNum = 0
            for _, tmpLine := range tmpLineSlice {
                if len(tmpLine) > 2 && tmpLine[0:3] == "===" {
                    continue
                }
                eventRowNum += 1
                if (tmpSQLType == "update" && eventRowNum%2 == 0) || tmpSQLType == "insert" || tmpSQLType == "delete" {
                    tmpSQLValue = tmpSQLValue + "--" + tmpLine
                    lineNum[tmpDTName + "-MaxNum"] += 10
                    lineResult[tmpDTName + "-Value-" + strconv.Itoa(lineNum[tmpDTName + "-MaxNum"])] = tmpSQLValue
                    //fmt.Println(tmpSQLValue)
                }
                // Save Data
                if lineNum[tmpDTName + "-MaxNum"] >= mybatchSize*10 {
                    SaveData(tmpDTName)
                }
            }
        }
        lag4ns := time.Now().UnixNano()
        preLag1ns += (lag2ns - lag1ns)
        preLag2ns += (lag3ns - lag2ns)
        preLag3ns += (lag4ns - lag3ns)
        preLagTime += 1
        if lag1ns - pre1ns >= 1000000000 {
            fmt.Printf("Line Number:%6s    Lag:%6s %6s %6s\n", strconv.FormatInt(preLagTime, 10), strconv.FormatInt(preLag1ns/preLagTime, 10), strconv.FormatInt(preLag2ns/preLagTime, 10), strconv.FormatInt(preLag3ns/preLagTime, 10))
            preLag1ns = int64(0)
            preLag2ns = int64(0)
            preLag3ns = int64(0)
            preLagTime = int64(0)
            pre1ns = lag4ns
        }
    }
}
// Create a binlog syncer with a unique server id, the server id must be different from other MySQL's.
// flavor is mysql or mariadb
func ItemInit(){
    flag_gtid := flag.String("gtid", "chenxinglong",       "MySQL Master's GTID")
    flag_show := flag.String("show", "false",              "Print Binglog events")
    flag_conf := flag.String("conf", "./binlogStream.cnf", "binlogStream's Conf file")
    flag.Parse()
    db_gtid  := *flag_gtid
    linePrint = *flag_show
    myConf   := *flag_conf
    // Reading CONF file
    fmt.Println(myConf)
    file, err := os.Open(myConf)
    checkError(err)
    defer file.Close()

    scanner := bufio.NewScanner(file)
    DTNameCnf = make(map[string]string)
    for scanner.Scan() {
        tmpItem := strings.SplitN(scanner.Text(), "=", 2)
        switch tmpItem[0] {
        case "batchSize":
            mybatchSize  = StrToInt64(tmpItem[1])
        case "gtidFile":
            gtidFile = tmpItem[1]
        case "ServerID":
            myServerID   = StrToInt64(tmpItem[1])
        case "Host":
            myHost       = tmpItem[1]
        case "Port":
            myPort       = StrToInt64(tmpItem[1])
        case "User":
            myUser       = tmpItem[1]
        case "Password":
            myPassword   = tmpItem[1]
        case "CKStr":
            myCKStr      = tmpItem[1]
        case "DTKey":
            tmpKVStr := tmpItem[1]
            tmpKV := strings.SplitN(tmpKVStr, "///", 2)
            DTNameCnf[tmpKV[0]] = tmpKV[1]
        }
    }
    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }
    //fmt.Println(mybatchSize,myServerID,myHost,myPort,myUser,myPassword,myCKStr,DTNameCnf)
    // Must Have GTID
    if db_gtid != "chenxinglong" {
        gtidSet = db_gtid
    } else if Exist(gtidFile) {
        gtidSet = ReadFile(gtidFile)
    } else {
        fmt.Println("binlogStream Program must have GTID")
        os.Exit(0)
    }
    // Create Conn
    connect, err = sql.Open("clickhouse", myCKStr)
    checkError(err)
}
func Exist(filename string) bool {
    _, err := os.Stat(filename)
    return err == nil || os.IsExist(err)
}
func main(){
    ItemInit()
    ParseData()
}
