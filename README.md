# mongo_export
导出mongodb的数据到文件中

一行一个json，通过配置文件进行配置，减少写代码的工作量

针对嵌套子文档的情况提供了简单的处理逻辑

### 使用方法

```
./mongo_export_app -export <secName> -d <YYYY-MM-DD>
```

示例：

```
./mongo_export_app -export post -d 2021-01-29
```

如果导出文件成功，会生成相应的md5

### 示例配置文件

#### server.ini

```ini
[uri_set]
# 名称自定义，在table.ini中使用
prod_main = mongodb://192.168.153.128:27017
test_main = mongodb://192.168.153.128:27018
[export]
# 导出路径，实际路径为${path}/日期(需要提前自行建立日期目录)
path = /root/gocode/mongo_export
# 全量导出时每批次导出数量
batch_size = 1000
# 按时间导出时每个时间段长度，单位秒
batch_time = 3600
# 并发导出线程数
thread_num = 2
# 是否将key全部转换为小写并输出
output_format_lower_key = false
```

#### table.ini

```ini
[numbers]
uri         = test_main
database    = testing
collection  = numbers
fields      = likenum,nickname,userid,articleid,_id,street
# 导出模式，FULL 全量 ,TIME 导出过去多少天
export_mode = FULL
# 展平一层嵌套字段，只处理一层
flatten_fields = {"foo":{"id":"fooId","name":"fooName"},"bar":{"id":"barId","name":"barName"}}
# 可以自定义导出路径，如果不存在则使用server.ini中的path
path = /data/mongo_export

[post]
uri         = prod_main
database    = uki
collection  = post
fields      = _id,userId,createdAt,isDeleted,type,auditStatus,optimalCreatedAt,recommendTimestamp,street,topic,themes,video,selfOnly,likeCount,popularity,damask
# 导出模式，按时间
export_mode = TIME
# 时间字段，数字时间戳
export_time_field = createdAt
# 导出过去多少天的内容
export_time_days = 90
# 导出路径
path = /data/mongo_export
# 展平字段，list套一层document，可以自定义分隔符 $sep, 不定义默认为逗号
flatten_list_fields = {"topic":{"name":"topic_name_list","$sep":","}}
# 展平字段，只处理一层document
flatten_fields = {"video":{"videoTime":"videoTime","videoMeasure":"videoMeasure","videoUrl":"videoUrl","videoCover":"videoCover"}}
# 展平array字段
parse_array_fields = {"themes":{"$sep":","},"tags":{"$sep":"$"}}
# json字段名转换
change_name_fields = {"createdAt":{"$keep":"0","createdAt":"TIMESTAMP2DATE","timestamp":"NOCHANGE"},"id":{"$keep":"0","postid":"NOCHANGE"}}
```

