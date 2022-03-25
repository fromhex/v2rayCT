# v2rayCT
一个v2ray的命令行工具
## feature
- 支持订阅链接（vmess）的查看、添加、删除、更新
- 支持查看和变更v2ray的服务器
- 支持系统服务的开启和关闭
- 使用sqlite3存储（订阅链接和更新的服务器）


## 使用
```bash
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
go build v2rayCT.go
sudo ./v2rayCT
```
<https://www.goproxy.io/zh/>
<https://goproxy.cn/>
