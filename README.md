# snowflake
1. distributed 64bit-UUID generator based on twitter snowflake
2. sequence generator, for generating sequential numbers, like auto-increment id.

# install 
> snowflake-uuid must be created first            
eg:      
curl http://172.17.42.1:2379/v2/keys/seqs/snowflake-uuid -XPUT -d value="0"  
> key of Next() must be created first in /seqs/<key_name>

install gpm, gvp first        
$go get -u https://github.com/GameGophers/snowflake/        
$cd snowflake     
$source gvp        
$gpm       
$go install snowflake         
$./startup.sh

#install with docker
docker build -t snowflake .     
