# mongo_export
A tool to export mongodb data to a file

Save a json in one line, configure it through the configuration file, without repeating the code.

Simple processing logic is provided for the case of nested subdocuments.

Support multi-threaded export.

### How to use

```
./mongo_export_app -export <secName> -d <YYYY-MM-DD>
```

Example:

```
./mongo_export_app -export post -d 2021-01-29
```

If the export file is successful, the corresponding <file.json>.md5 will be generated

### Sample configuration file

#### server.ini

```ini
[uri_set]
# Name custom, used in table.ini
prod_main = mongodb://192.168.153.128:27017
test_main = mongodb://192.168.153.128:27018
[export]
# Export path, the actual path is ${path}/date (YYYY-MM-DD, you need to create a date directory in advance)
path = /root/gocode/mongo_export
# Export quantity per batch when exporting in full
batch_size = 1000
# The length of each time period when exporting by time, in seconds
batch_time = 3600
# Number of concurrent export threads
thread_num = 2
# Whether to convert all keys to lowercase and output
output_format_lower_key = false
```

#### table.ini

```ini
[numbers]
uri         = test_main
database    = testing
collection  = numbers
fields      = likenum,nickname,userid,articleid,_id,street
# Export mode, FULL, TIME (how many days in the past), TIME_HOUR
export_mode = FULL
# THE FOLLOWING ARE OPTIONAL
# Flatten one level of nested fields and process only one level
flatten_fields = {"foo":{"id":"fooId","name":"fooName"},"bar":{"id":"barId","name":"barName"}}
# You can customize the export path, if it does not exist, use the path in server.ini
path = /data/mongo_export

[post]
uri         = prod_main
database    = uki
collection  = post
fields      = _id,userId,createdAt,isDeleted,type,auditStatus,optimalCreatedAt,recommendTimestamp,street,topic,themes,video,selfOnly,likeCount,popularity,damask
# export mode, by time
export_mode = TIME
# There needs to be a time field in the collection, a 10-digit timestamp
export_time_field = createdAt
# How many days in the past to export
export_time_days = 90
# THE FOLLOWING ARE OPTIONAL
# Export path, if not filled, it will be in server.ini by default
path = /data/mongo_export
# Flatten the field, list a layer of documents, you can customize the separator $sep, if not defined, the default is a comma
flatten_list_fields = {"topic":{"name":"topic_name_list","$sep":","}}
# Flatten the field and only process one layer of document
flatten_fields = {"video":{"videoTime":"videoTime","videoMeasure":"videoMeasure","videoUrl":"videoUrl","videoCover":"videoCover"}}
# flatten array field
parse_array_fields = {"themes":{"$sep":","},"tags":{"$sep":"$"}}
# field name conversion
change_name_fields = {"createdAt":{"$keep":"0","createdAt":"TIMESTAMP2DATE","timestamp":"NOCHANGE"},"id":{"$keep":"0","postid":"NOCHANGE"}}
```

