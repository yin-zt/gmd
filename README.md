# gmd
I create these  programs as experiments to play with golang, or to solve problems for myself. I would gladly accept pointers from others to improve, simplify, or make the code more efficient. If you would like to make any comments then please feel free to email me at `mailong9527@163.com`

## file
1. ftpserver: supports you to deploy an ftp server for sharing filedata
   `gmd ftpserver -u mailong -p 123 -P 9090 -h 127.0.0.1`


## web request
1. request [get]
   `gmd request -u http://127.0.0.1:8888/api/base/ping`
2. request [post]
   `gmd request -u http://127.0.0.1:8888/api/base/login -
   d '{\"username\": \"admin\", \"password\": \"VjjxGSQvPTt+9tnQHo7Vo+cVpW\"}'`

## shell
1. exec: supports you to exec command on windows or linux platform，such as：
   `gmd exec -c hostname`
2. ip: supports you to search ip of the local machine
   `gmd ip`
3. color: suports you to select one color to decaloted your message:
   `gmd color -m "message" -c color`
