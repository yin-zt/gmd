# gmd
I create these  programs as experiments to play with golang, or to solve problems for myself. I would gladly accept pointers from others to improve, simplify, or make the code more efficient. If you would like to make any comments then please feel free to email me at 15626499421@163.com

## file
1. ftpserver
   `go run .\main.go ftpserver -u mailong -p 123 -P 9090 -h 127.0.0.1`


## web request
1. request [get]
   `go run .\main.go request -u http://127.0.0.1:8888/api/base/ping`
2. request [post]
   `go run .\main.go request -u http://127.0.0.1:8888/api/base/login -
   d '{\"username\": \"admin\", \"password\": \"VjjxGSQvPTt+9tnQHo7Vo+cVpW\"}'`
