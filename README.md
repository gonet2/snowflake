# SNOWFLAKE
[![Build Status](https://travis-ci.org/gonet2/snowflake.svg?branch=master)](https://travis-ci.org/gonet2/snowflake)

# 设计理念
1. 分布式uuid发生器，twitter snowflake的go语言版本      
2. 序列发生器        


uuid格式为:

    +-------------------------------------------------------------------------------------------------+
    |              |                                    |                     |                       |
    | UNUSED(1BIT) |         TIMESTAMP(41BIT)           |  MACHINE-ID(10BIT)  |   SERIAL-NO(12BIT)    |
    |              |                                    |                     |                       |
    +-------------------------------------------------------------------------------------------------+



            

其中10bit的machine-id，通过etcd做唯一性，使得每个snowflake实例可以做到0配置        

# 安装 
默认情况下uuid发生器依赖的snowflake-uuid键值对必须预先在etcd中创建，snowflake启动的时候会读取，例如： 

       curl http://172.17.42.1:2379/v2/keys/seqs/snowflake-uuid -XPUT -d value="0"          

如果完全由用户自定义machine_id,可以通过环境变量指定，如:

       export MACHINE_ID=123

如果要使用序列发生器Next()，必须预先创建一个key，例如:       

       curl http://172.17.42.1:2379/v2/keys/seqs/userid -XPUT -d value="0"          
 
其他部分参考Dockerfile         

# 使用
![snowflake](snowflake.gif)
参考测试用例和snowflake.proto          

# 环境变量
> NSQD_HOST: eg : http://172.17.42.1:4151          
> ETCD_HOST: eg: http://172.17.42.1:2379       
> MACHINE_ID: eg: 123
