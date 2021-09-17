#!/bin/bash
pid=`ps -ef | grep DoTaskC |  grep -v "grep" | awk '{print $2}'`
if [ "$pid" = "" ]
then
	echo "no DoTask Server alive !"
else
	echo $pid
	kill -9 $pid
	echo "stop success !"
fi
../bin/DoTaskC > /dev/null 2>&1  &
echo "restart sucess!"
