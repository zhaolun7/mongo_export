package conf

import (
	"fmt"
	"gopkg.in/ini.v1"
	"mongo_export/utils"
	"strconv"
	"strings"
)

var UriMap = make(map[string]string, 10)
var ExportPath = "./"
var BatchSize int64 = 1000
var BatchTime int64 = 3600
var ThreadNum = 2
var OutputLowerFormatKey = false


var TableInfoMap map[string]map[string]string

func init() {
	initServerConf()
	initTableConf()
}

func initServerConf() {
	cfg, err := ini.Load("conf/server.ini")
	if err != nil {
		panic(err)
	}
	uriSetSection, err := cfg.GetSection("uri_set")
	if err != nil {
		panic(err)
	}
	for _, uriName := range uriSetSection.Keys() {
		key := uriName.Name()
		value := uriName.Value()
		utils.Logger.Infof("uri key: %s,value: %s", key, value)
		UriMap[key] = value
	}

	ExportPath = cfg.Section("export").Key("path").Value()
	if strings.HasSuffix(ExportPath, "/") {
		ExportPath = strings.TrimRight(ExportPath, "/")
	}
	utils.Logger.Infof("ExportPath:%s", ExportPath)
	BatchSize, err = cfg.Section("export").Key("batch_size").Int64()
	utils.Logger.Infof("batch_size:%d", BatchSize)
	BatchTimeStr := cfg.Section("export").Key("batch_time").Value()
	BatchTime, err = strconv.ParseInt(BatchTimeStr, 10, 64)
	utils.Logger.Infof("batch_time:%d", BatchTime)
	ThreadNum, err = cfg.Section("export").Key("thread_num").Int()
	utils.Logger.Infof("thread_num:%d", ThreadNum)
	OutputLowerFormatKey, err = cfg.Section("export").Key("output_format_lower_key").Bool()
	utils.Logger.Infof("output_format_lower_key:%t", OutputLowerFormatKey)
}

func initTableConf() {
	TableInfoMap = make(map[string]map[string]string)
	cfg, err := ini.Load("conf/table.ini")
	if err != nil {
		panic(err)
	}
	for _, section := range cfg.Sections() {
		secName := section.Name()
		if secName != "DEFAULT" {
			secMap := make(map[string]string)
			for _, item := range section.Keys() {
				secMap[item.Name()] = item.Value()
				//fmt.Println(item.Name(),"=>",item.Value())
			}
			uri, ok := secMap["uri"]
			if ok == false {
				utils.Logger.Error("uri is empty")
				panic("uri is empty")
			}
			if _, ok := UriMap[uri]; !ok {
				errMsg := fmt.Sprintf("uri %s is not in conf/server.ini", uri)
				utils.Logger.Error(errMsg)
				panic(errMsg)
			}
			TableInfoMap[secName] = secMap
		}
	}

}
