package export_task

import (
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"mongo_export/conf"
	"mongo_export/utils"
	"strings"
	"sync"
)

func CreateFullModeFunc(client *mongo.Client, ctx context.Context, isLeftClosed bool, secName string, tableInfo map[string]string, wg *sync.WaitGroup,
	minId string, maxId string, channel chan []string, threadChan <-chan int, cnt int) Executor {
	database, _ := tableInfo["database"]
	collection, _ := tableInfo["collection"]
	// 真实uri
	//fmt.Println(uri)
	minIdObj, err := primitive.ObjectIDFromHex(minId)
	if err != nil {
		panic(err)
	}
	maxIdObj, err := primitive.ObjectIDFromHex(maxId)
	if err != nil {
		panic(err)
	}
	flagLessThan := "$gt"
	if isLeftClosed {
		flagLessThan = "$gte"
	}
	filterM := bson.M{"_id": bson.M{flagLessThan: minIdObj, "$lte": maxIdObj}}
	//filterJson := fmt.Sprintf("{\"_id\":{\"$gte\":\"%s\",\"$lt\":\"%s\"}}",min_id, max_id)
	//fmt.Println("filter_mp:", filterM)
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
		utils.Logger.Infof("[%s]task %d start to run, _id range: %v->%v,len(msgChan):%d", secName, cnt, minId, maxId, len(channel))
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
			resultSlice := make([]string,0,conf.BatchSize+10)
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
				//fmt.Printf("after result:%T->%v\n\t\t%s\n", result, result, str2)
				//fmt.Println(string(str2))
				//channel <- string(str2)
				resultSlice = append(resultSlice,string(str2))
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
				utils.Logger.Infof("[%s]task %d retry %d times, _id range: %v->%v,len(msgChan):%d", secName, cnt, i+1, minId, maxId, len(channel))
			}
		}
		return nil
	}
	return f
}

func GetMinOrMaxIdForFullMode(client *mongo.Client, ctx context.Context, tableInfo map[string]string, minId string, batchSize int64, order int) string {
	database, _ := tableInfo["database"]
	collectionName, _ := tableInfo["collection"]
	var minIdObj primitive.ObjectID
	var err error = nil
	flagFirstFindMinId := false
	if minId == "NO_MIN_ID" {
		flagFirstFindMinId = true
	} else {
		minIdObj, err = primitive.ObjectIDFromHex(minId)
		if err != nil {
			panic(err)
		}
	}
	collection := client.Database(database).Collection(collectionName)
	findOneOption := options.FindOne()
	findOneOption.SetProjection(bson.D{{"_id", 1}})
	findOneOption.SetSort(bson.D{{"_id", order}})
	findOneOption.SetSkip(batchSize)

	filterM := bson.M{"_id": bson.M{"$gte": minIdObj}}
	if flagFirstFindMinId {
		filterM = bson.M{}
	}
	singleResult := collection.FindOne(ctx, filterM, findOneOption)
	var result bson.M
	err = singleResult.Decode(&result)
	if err != nil {
		return "NO_RESULT"
	}
	return result["_id"].(primitive.ObjectID).Hex()
}
