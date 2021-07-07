package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io"
	"mongo_export/conf"
	"mongo_export/export_task"
	"mongo_export/utils"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	var secName string
	var dateStr string
	flag.StringVar(&secName,"export", "", "secName")
	flag.StringVar(&dateStr,"d", "", "date")
	flag.Parse()
	if secName == "" {
		errMsg := "need secName: -export secName"
		utils.Logger.Error(errMsg)
		panic(errMsg)
	}
	if dateStr == "" {
		errMsg := "need date: -d date"
		utils.Logger.Error(errMsg)
		panic(errMsg)
	}
	sectionMap := conf.TableInfoMap[secName]
	if sectionMap == nil {
		errMsg := "not exit secName:" + secName
		utils.Logger.Error(errMsg)
		panic(errMsg)
	}
	exportMode := conf.TableInfoMap[secName]["export_mode"]
	utils.Logger.Info("export_mode:", exportMode)
	exportMode = strings.ToUpper(exportMode)
	if "FULL" != exportMode && "TIME" != exportMode && "TIME_HOUR" != exportMode{
		errMsg := "valid export_mode:" + exportMode
		utils.Logger.Error(errMsg)
		panic(errMsg)

	}
	exportData(secName, exportMode, dateStr)

}

func writeToFile(ch chan []string, done chan bool, path string) {
	f, err := os.Create(path)
	if err != nil {
		utils.Logger.Error(err)
		panic(err)
	}
	for slice := range ch {
		for _,d := range slice {
			_, err = fmt.Fprintln(f, d)
			if err != nil {
				utils.Logger.Error(err)
				f.Close()
				done <- false
				return
			}
		}
	}
	err = f.Close()
	if err != nil {
		utils.Logger.Error(err)
		done <- false
		return
	}
	done <- true
}

func getFileMd5(path string) string {
	// 文件全路径名
	pFile, err := os.Open(path)
	if err != nil {
		fmt.Errorf("打开文件失败，filename=%v, err=%v", path, err)
		return ""
	}
	defer pFile.Close()
	md5h := md5.New()
	io.Copy(md5h, pFile)

	return hex.EncodeToString(md5h.Sum(nil))
}

func exportData(sectionName string, exportMode string, dateStr string) {
	ch := make(chan []string, 100)
	chCon := make(chan int, conf.ThreadNum)
	done := make(chan bool)
	var wg = sync.WaitGroup{}
	outputPath := conf.TableInfoMap[sectionName]["path"]
	if outputPath == "" {
		outputPath = conf.ExportPath
	}
	if strings.HasSuffix(outputPath,"/") {
		outputPath = strings.TrimRight(outputPath, "/")
	}

	// 文件导出路径
	fileAbsPath := fmt.Sprintf("%s/%s/%s.json",outputPath, dateStr ,sectionName)
	utils.Logger.Info("fileAbsPath:",fileAbsPath)
	go writeToFile(ch,done,fileAbsPath)
	aliasUri, _ := conf.TableInfoMap[sectionName]["uri"]
	// 真实uri
	uri, _ := conf.UriMap[aliasUri]
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Minute)
	defer cancel()
	opts := options.Client().ApplyURI(uri)
	//// TODO : bad
	//opts.ReadPreference = readpref.SecondaryPreferred()
	//client, err := mongo.Connect(ctx, opts)
	utils.Logger.Info("opts.ReadPreference:",opts.ReadPreference)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		panic(err)
	}
	// defer to close client
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	if exportMode == "FULL" {
		// 全量导出
		firstMinId := export_task.GetMinOrMaxIdForFullMode(client,ctx,conf.TableInfoMap[sectionName],"NO_MIN_ID",0,1) // 最小的id
		minId := firstMinId
		lastMaxId := export_task.GetMinOrMaxIdForFullMode(client,ctx,conf.TableInfoMap[sectionName],"NO_MIN_ID",0,-1) //最大的id
		//fmt.Println("before for",minId,lastMaxId)
		cnt := 0
		for {
			isLeftClosed := false
			// 第一次循环左闭合
			if minId == firstMinId {
				isLeftClosed = true
			}
			maxId := export_task.GetMinOrMaxIdForFullMode(client,ctx,conf.TableInfoMap[sectionName],minId,conf.BatchSize,1)
			if maxId == "NO_RESULT" {
				maxId = lastMaxId
			} else if strings.Compare(maxId,lastMaxId) > 0 {
				// 在循环过程中，用户新插入了数据，此时这段是不要同步的,要等到第二天执行同步,否则循环不好结束
				maxId = lastMaxId
			}
			// 除了第一个左闭右闭，剩余的区间为左开右闭
			f :=export_task.CreateFullModeFunc(client,ctx, isLeftClosed,sectionName,conf.TableInfoMap[sectionName], &wg, minId, maxId, ch, chCon, cnt)
			cnt ++
			wg.Add(1)
			chCon <- 1

			go f()

			minId = maxId
			if maxId == lastMaxId {
				break
			}
		}
	} else if exportMode == "TIME" || exportMode == "TIME_HOUR" {
		// 按时间范围导出
		exportTimeField := conf.TableInfoMap[sectionName]["export_time_field"]
		utils.Logger.Info("export_time_field:",exportTimeField)
		if exportTimeField == "" {
			msg:= "miss config -> export_time_field"
			utils.Logger.Error(msg)
			panic(msg)
		}
		exportTimeDays := conf.TableInfoMap[sectionName]["export_time_days"]
		if exportTimeDays == "" {
			msg:= "miss config -> export_time_days"
			utils.Logger.Error(msg)
			panic(msg)
		}
		exportTimeDaysInt,err := strconv.ParseInt(exportTimeDays, 10, 64)
		if err != nil {
			utils.Logger.Error(err)
			panic(err)
		}

		lastMaxTimestamp := utils.GetTimestamp(dateStr) + 24 * 3600
		firstMinTimestamp := utils.GetTimestamp(dateStr) - (exportTimeDaysInt - 1) * 24 * 3600
		if exportMode == "TIME_HOUR" {
			// 小时级任务，时间间隔写死一小时
			lastMaxTimestamp = utils.GetTimestampFromHour(dateStr) + 3600
			firstMinTimestamp = utils.GetTimestampFromHour(dateStr) - (exportTimeDaysInt - 1) * 3600
			conf.BatchTime = 3600
		}


		minTime := firstMinTimestamp
		cnt := 0
		for {
			maxTime := minTime + conf.BatchTime
			if maxTime > lastMaxTimestamp {
				maxTime = lastMaxTimestamp
			}
			//fmt.Println("time period =>:[",cnt,"] => ",minTime,maxTime)

			//isLeftClosed := false
			//// 第一次循环左闭合
			//if minTime == firstMinTimestamp {
			//	isLeftClosed = true
			//}
			// 按时间导出，所有时间区间应为左闭右开
			f :=export_task.CreateTimeModeFunc(client,ctx,true,sectionName,conf.TableInfoMap[sectionName], &wg, minTime, maxTime, ch, chCon, exportTimeField, cnt)
			cnt ++
			wg.Add(1)
			chCon <- 1
			go f()

			minTime = maxTime
			if maxTime == lastMaxTimestamp {
				break
			}
		}
	}

	go func() {
		wg.Wait()
		close(ch)
	}()
	d := <-done
	if d == true {
		utils.Logger.Info("File written successfully")
		md5 := getFileMd5(fileAbsPath)
		md5FilePath := fileAbsPath + ".md5"
		f, err := os.Create(md5FilePath)
		if err != nil {
			utils.Logger.Error("failed to write md5 file:" + md5FilePath,err)
		}
		defer f.Close()
		f.WriteString(md5)
	} else {
		utils.Logger.Info("File writing failed")
	}
}
