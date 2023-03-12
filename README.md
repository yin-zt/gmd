# gmd
I create these  programs as experiments to play with golang, or to solve problems for myself. I would gladly accept pointers from others to improve, simplify, or make the code more efficient. If you would like to make any comments then please feel free to email me at `mailong9527@163.com`

## file
1. ftpserver: supports you to deploy an ftp server for sharing filedata
   `gmd ftpserver -u mailong -p 123 -P 9090 -h 127.0.0.1`
2. Httpserver: deploy a http server to share some files in a given floder.
   `gmd httpserver -h 127.0.0.1 -p 8080`


## web request
1. request [get]
   `gmd request -u http://127.0.0.1:8888/api/base/ping`
2. request [post]
   `gmd request -u http://127.0.0.1:8888/api/base/login -
   d '{\"username\": \"admin\", \"password\": \"VjjxGSQvPTt+9tnQHo7Vo+cVpW\"}'`

## shell
1. shell: support you to use gmd exec scriptfile includind python\shell\powershell etc on the local-machine.
   `go run .\main.go shell -f test.ps1 -d "." -t 10`
2. exec: supports you to exec command on windows or linux platform，such as：
   `gmd exec -c hostname`
3. ip: supports you to search ip of the local machine
   `gmd ip`
4. color: suports you to select one color to decalate your message:
   `gmd color -m "message" -c color`
5. uuid: generate a uuid string randomly.
   `gmd uuid`
6. randint: generate one number between a given interval.
   `gmd randint -r 10:40`
7. md5: support generate md5 value of one string or file content.
   `gmd md5 -s "string" || gmd md5 -f filename`
8. cut: support get sub string from a given string.
   `gmd cut -s "abcdefgf" -p "2:5"`
9. split: support split one string with given "separator".
   `echo "hello world" | gmd split -s " "`
10. replace: support replace old string with given substring.
    `gmd replace -s worldhello -n FUCK -o world`
11. match: support find out substring which satisfy regex.
    `gmd match -s "hell(i)45oworld" -m "[\d+]+" -o "i";`
12. pq: `gmd pq -m html -f xxx.html`

## interactive operation
1. keys: `echo '{"aa": "bb", "test": "hello world"}' | gmd keys`
2. len: `echo "aabbcc" | gmd len  or  echo '{"key1": "val1", "key2": "val2"}' | gmd len`
3. kvs: `echo [k1, k2, k3] | gmd kvs  || echo '{"k1": "v1", "k2": "v2"}' | gmd kvs`
4. join: `echo '["aa", "bb", "cc"]' | gmd join -s "-" -w "GG"`
5. json_val: `echo '{"tt":"helloworld", "bb": "fufu"}' | gmd json_val`
6. jq: `echo '{"tt":"helloworld", "bb": "fufu"}' | gmd jq`
7. 

## sql operation
1. sqlite3: 
   `gmd sqlite -s sql -f filename -t tablename`
   `gmd jf -c * -w "condition" --limit 10`
2. 


## info
1. info: output information about gmd including version, server info and etc. you can use bellow command:
   `gmd info`
