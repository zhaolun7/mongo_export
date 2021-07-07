package export_task

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"mongo_export/conf"
	"mongo_export/utils"
	"strconv"
	"strings"
)

type Executor func() error

func dealSpecialFields(result *bson.M, flattenFieldsMap map[string]map[string]string) {
	for field, mp := range flattenFieldsMap {
		subDoc, err := json.Marshal((*result)[field])
		delete(*result, field)
		if err != nil {
			utils.Logger.Error(err)
			panic(err)
			return
		}
		if subDoc == nil {
			continue
		}
		var subDocMap map[string]interface{} = nil
		err = json.Unmarshal(subDoc, &subDocMap)
		if err != nil {
			utils.Logger.Error(err)
			panic(err)
			return
		}
		for flatKey, flatReplaceKey := range mp {
			//fmt.Println("field=====>flatKey,flatReplaceKey=>",field,flatKey,flatReplaceKey)
			value := subDocMap[flatKey]
			if value != nil {
				//fmt.Println("put key:",flatReplaceKey,value)
				(*result)[flatReplaceKey] = value
			}
		}
	}
}

func dealListSpecialFields(result *bson.M, flattenListFieldsMap map[string]map[string]string) {
	for field, mp := range flattenListFieldsMap {
		subDoc, err := json.Marshal((*result)[field])
		delete(*result, field)
		if err != nil {
			utils.Logger.Error(err)
			panic(err)
			return
		}
		if subDoc == nil {
			continue
		}
		var subDocMapList []map[string]interface{} = nil
		err = json.Unmarshal(subDoc, &subDocMapList)
		if err != nil {
			utils.Logger.Error(err)
			panic(err)
			return
		}
		//fmt.Println("subDocMapList:",subDocMapList)
		sep := mp["$sep"]
		if sep == "" {
			// 默认逗号分隔
			sep = ","
		}
		for _, subDocMap := range subDocMapList {
			for flatKey, flatReplaceKey := range mp {
				if flatKey == "$sep" {

					continue
				}
				value := subDocMap[flatKey]
				if value != nil {
					// 所有类型转换为字符串
					strValue := fmt.Sprintf("%v", value)
					preValue := (*result)[flatReplaceKey]
					if preValue == nil {
						(*result)[flatReplaceKey] = strValue
					} else {
						// stupid but simple way
						(*result)[flatReplaceKey] = fmt.Sprintf("%v", preValue) + sep + strValue
					}
				}
			}
		}
	}
}

func dealArrayFields(result *bson.M, flattenListFieldsMap map[string]map[string]string) {
	for field, mp := range flattenListFieldsMap {
		subDoc, err := json.Marshal((*result)[field])
		delete(*result, field)
		if err != nil {
			utils.Logger.Error(err)
			panic(err)
		}
		sep := mp["$sep"]
		if sep == "" {
			// 默认逗号分隔
			sep = ","
		}
		if subDoc == nil {
			continue
		}
		var subFieldArray []interface{} = nil
		err = json.Unmarshal(subDoc, &subFieldArray)
		if err != nil {
			utils.Logger.Error(err)
			panic(err)
			return
		}
		//fmt.Println("subFieldArray:",subFieldArray)

		var buffer bytes.Buffer
		for idx, subItem := range subFieldArray {
			if idx > 0 {
				buffer.WriteString(sep)
			}
			buffer.WriteString(fmt.Sprintf("%v", subItem))
		}
		//fmt.Println("result:",buffer.String())
		(*result)[field] = buffer.String()
	}
}

func dealChangeNameFields(result *bson.M, fieldsMap map[string]map[string]string) {
	//fmt.Println(fieldsMap)
	for field, mp := range fieldsMap {
		//fmt.Println("field:",field)
		var subValue = (*result)[field]
		if subValue == nil {
			//fmt.Println("not find,return")
			continue
		}

		keep := mp["$keep"]
		if keep != "1" {
			delete(*result, field)
		}
		for replaceKey, convertMode := range mp {
			if replaceKey == "$keep" {
				continue
			}
			if convertMode == "TIMESTAMP2DATE" {
				switch subValue.(type) {
				case int:
					(*result)[replaceKey] = utils.Utc2dateDay(int64(subValue.(int)))
				case int32:
					(*result)[replaceKey] = utils.Utc2dateDay(int64(subValue.(int32)))
				case int64:
					(*result)[replaceKey] = utils.Utc2dateDay(subValue.(int64))
				case float32:
					(*result)[replaceKey] = utils.Utc2dateDay(int64(subValue.(float32)))
				case float64:
					(*result)[replaceKey] = utils.Utc2dateDay(int64(subValue.(float64)))
				default:
					panic("convert failed")
				}
			} else if convertMode == "TIMESTAMP_FORMAT" {
				var v string
				switch subValue.(type) {
				case int:
					v = fmt.Sprintf("%d", subValue)
				case int32:
					v = fmt.Sprintf("%d", subValue)
				case int64:
					v = fmt.Sprintf("%d", subValue)
				case float32:
					v = fmt.Sprintf("%f", subValue)
				case float64:
					v = fmt.Sprintf("%f", subValue)
				default:
					panic("convert failed")
				}
				intV,err := strconv.Atoi(v[0:10])
				if err != nil {
					fmt.Println(v)
					panic(err)
					return
				}
				(*result)[replaceKey] = intV
			}  else if convertMode == "TRIM_LEFT_ZERO" {
				var v string
				//fmt.Println(reflect.TypeOf(subValue))
				//panic("hehe")
				switch subValue.(type) {
				case string:
					v = fmt.Sprintf("%s", subValue)
				case primitive.ObjectID:
					obj := subValue.(primitive.ObjectID)
					v = obj.Hex()
				}
				(*result)[replaceKey] = strings.TrimLeft(v,"0")
			} else {
				(*result)[replaceKey] = subValue
			}
		}
	}
}

func lowerKey(result *bson.M) *bson.M {
	if conf.OutputLowerFormatKey == false {
		return result
	}
	var lowerResult = make(bson.M, 0)
	for k, v := range *result {
		lowerK := strings.ToLower(k)
		lowerResult[lowerK] = v
	}
	return &lowerResult
}

func innerCreateMap(tableInfo map[string]string, sepName string, resultMap map[string]map[string]map[string]string) {
	flattenFieldsStr, exits1 := tableInfo[sepName]
	var flattenFieldsMap map[string]map[string]string = nil
	if exits1 {
		err := json.Unmarshal([]byte(flattenFieldsStr), &flattenFieldsMap)
		if err != nil {
			utils.Logger.Error(err)
			panic(err)
		}
		resultMap[sepName] = flattenFieldsMap
	}
}

func createSpecialHandleMaps(tableInfo map[string]string) map[string]map[string]map[string]string {
	resultMap := make(map[string]map[string]map[string]string, 0)
	//* 特殊处理1，展开1层子文档,之后在程序中解析
	innerCreateMap(tableInfo, "flatten_fields", resultMap)
	//* 特殊处理2，展开1层list子文档,之后在程序中解析
	innerCreateMap(tableInfo, "flatten_list_fields", resultMap)
	//* 特殊处理3，展开1层list子文档,之后在程序中解析
	innerCreateMap(tableInfo, "parse_array_fields", resultMap)
	//* 特殊处理4，json字段名转换
	innerCreateMap(tableInfo, "change_name_fields", resultMap)

	return resultMap
}

func processSpecialHandleMaps(result *bson.M, specialMap map[string]map[string]map[string]string) {
	if len(specialMap) == 0 {
		return
	}
	{
		// 特殊处理1，展开1层子文档
		flattenFieldsMap := specialMap["flatten_fields"]
		if flattenFieldsMap != nil {
			dealSpecialFields(result, flattenFieldsMap)
		}
	}
	{
		// 特殊处理2，展开1层list子文档
		flattenListFieldsMap := specialMap["flatten_list_fields"]
		if flattenListFieldsMap != nil {
			dealListSpecialFields(result, flattenListFieldsMap)
		}
	}
	{
		// 特殊处理3，Array数组值转换为string
		parseArrayFieldsMap := specialMap["parse_array_fields"]
		if parseArrayFieldsMap != nil {
			dealArrayFields(result, parseArrayFieldsMap)
		}
	}
	{
		// 特殊处理4，json字段名转换
		changeNameFieldsMap := specialMap["change_name_fields"]
		if changeNameFieldsMap != nil {
			dealChangeNameFields(result, changeNameFieldsMap)
		}
	}

}
