#!/bin/sh
export LANG=en_US.UTF-8
pushd `dirname $0` >> /dev/null
path=`pwd`
popd >> /dev/null
cd  $path/
if [ "x$1" != "x" ];then 
    yesterday=$1
else
    yesterday=$(date -d "yesterday" +%F)
fi

week_ago=$(date -d "$yesterday -7 day" +%F)
echo "["$(date +'%F %T')'] start task, date' $yesterday
logfile=$path/logs/export_${yesterday}.log
#exportpath=/data1/mongo_export/data/${yesterday}
exportpath=$(grep path conf/server.ini | grep -v '#' | cut -d '=' -f 2 | awk '{print $1}')/${yesterday}
#secname=order
echo "[$(date +'%F %T')] clean date:$week_ago"
rm -rf $(grep path conf/server.ini | grep -v '#' | cut -d '=' -f 2 | awk '{print $1}')/${week_ago}
rm -rf $exportpath
mkdir -p $exportpath

for secname in $(cat $path/export.conf | grep -v '#' | grep -v '^$' | xargs)
do
    echo "["$(date +'%F %T')"] ["$secname"] start export" >> $logfile
    cd $path
    for ((i=1;i<=3;i++))
    do
        ./mongo_to_hdfs_app -export ${secname} -d ${yesterday}
        if [ $? -ne 0 ];then
            echo "./mongo_to_hdfs_app -export ${secname} -d ${yesterday} failed" >> $logfile
            if [  $i == 3 ];then
               continue 2
            fi
            continue
        else
            echo "["$(date +'%F %T')"] ["$secname"] finish export" >> $logfile
            break
        fi
    done
    
    cd $exportpath
    tar --remove-files -zcf ${secname}.tgz.tmp ${secname}.json ${secname}.json.md5
    if [ $? -ne 0 ];then
        echo "failed : tar --remove-files -zcf ${secname}.tgz ${secname}.json"
    else
        mv ${secname}.tgz.tmp ${secname}.tgz
        echo "["$(date +'%F %T')"] ["$secname"] finish tar" >> $logfile
    fi
done

echo "["$(date +'%F %T')'] end task, date' $yesterday
