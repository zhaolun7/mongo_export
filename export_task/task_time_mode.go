package export_task

import (
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"mongo_export/utils"
	"strings"
	"sync"
)

func CreateTimeModeFunc(client *mongo.Client, ctx context.Context, isLeftClosed bool, secName string, tableInfo map[string]string, wg *sync.WaitGroup,
	minId int64, maxId int64, channel chan []string, threadChan <-chan int, exportTimeField string, cnt int) Executor {

	database, _ := tableInfo["database"]
	collection, _ := tableInfo["collection"]
	// 真实uri
	//fmt.Println(exportTimeField)

	flagGreaterThan := "$gt"
	if isLeftClosed {
		flagGreaterThan = "$gte"
	}
	// 0 direct int64, good!
	//filterM := bson.M{exportTimeField:bson.M{flagGreaterThan: minId, "$lte": maxId}}
	/*
		// 1 int64 to string , not work
		filterM := bson.M{exportTimeField:bson.M{flagGreaterThan: strconv.FormatInt(minId,10), "$lte": strconv.FormatInt(maxId,10)}}

		// 2 int64 to int, good!
		filterM := bson.M{exportTimeField:bson.M{flagGreaterThan: int(minId), "$lte": int(maxId)}}
	*/
	// 3 int64 to double, good
	filterM := bson.M{exportTimeField: bson.M{flagGreaterThan: float64(minId), "$lt": float64(maxId)}}
	//utils.Logger.Info("filterM:", filterM)

	// 要读取的字段列表
	projection := bson.D{}
	//projection := bson.D{bson.E{"_id",0}}
	fields := tableInfo["fields"]
	fieldsSlice := strings.Split(fields, ",")
	needId := false
	for _, field := range fieldsSlice {
		projection = append(projection, bson.E{field, 1})
		if field == "_id" {
			needId = true
		}
	}
	if needId == false {
		projection = append(projection, bson.E{"_id", 0})
	}
	specialHandleMaps := createSpecialHandleMaps(tableInfo)
	f := func() error {
		//fmt.Println("start to run")
		//time.Sleep(5*time.Second)
		utils.Logger.Infof("[%s]task[%v] start to run, %s range: %v->%v,len(msgChan):%d", secName, cnt, exportTimeField, minId, maxId,len(channel))
		defer func() {
			<-threadChan
		}()

		if wg != nil {
			defer wg.Done()
		}

		g :=func() ([]string, error) {
			collection := client.Database(database).Collection(collection)
			cur, err := collection.Find(ctx, filterM, options.Find().SetProjection(projection))
			if err != nil {
				panic(err)
			}
			defer cur.Close(ctx)
			resultSlice := make([]string,0,1000)
			for cur.Next(ctx) {
				var result bson.M
				err := cur.Decode(&result)
				if err != nil {
					utils.Logger.Error(err)
					panic(err)
				}
				// 特殊处理
				processSpecialHandleMaps(&result, specialHandleMaps)
				lowerResult := lowerKey(&result)
				str2, err := json.Marshal(*lowerResult)
				if err != nil {
					return nil, err
				}
				resultSlice = append(resultSlice,string(str2))
				//channel <- string(str2)
			}
			if err := cur.Err(); err != nil {
				utils.Logger.Error(err)
				return nil,err
			}
			return resultSlice,nil
		}
		// 重试3次，如果仍然报错，则抛出异常
		for i:=1;i<=3;i++ {
			resultSlice,err := g()
			if err == nil {
				// 将结果传入文件中
				channel <- resultSlice
				break
			} else {
				if i == 3 {
					panic(err)
				}
				utils.Logger.Infof("[%s]task[%v] retry %d times, %s range: %v->%v,len(msgChan):%d", secName, cnt, i+1, exportTimeField, minId, maxId,len(channel))
			}
		}
		return nil
	}
	return f
}
