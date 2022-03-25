package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/imroc/req"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/urfave/cli/v2"
)

var Banner = `
__      _____                   _____ _______ 
\ \    / /__ \                 / ____|__   __|
 \ \  / /   ) |_ __ __ _ _   _| |       | |   
  \ \/ /   / /| '__/ _' | | | | |       | |   
   \  /   / /_| | | (_| | |_| | |____   | |   
    \/   |____|_|  \__,_|\__, |\_____|  |_|   
                          __/ |               
                         |___/  
`

const (
	dbDriverName = "sqlite3"
	dbName       = "./config.db"
)

var pool sql.DB

/* 保存服务器信息 */
type Server struct {
	id     int
	config string
	cid    int
}

/* 保存订阅信息 */
type Config struct {
	id      int
	subAddr string
}

/* 结构体判断是否为空 */
func (x Server) IsEmpty() bool {
	return reflect.DeepEqual(x, Server{})
}

var Sub = &cli.Command{
	Name:        "config",
	Usage:       "订阅管理",
	Description: "查看、添加、删除、更新",
	Subcommands: []*cli.Command{
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "查看所有的订阅地址",
			Action:  config_list,
		},
		{
			Name:    "add",
			Aliases: []string{"a"},
			Usage:   "添加vmess订阅地址",
			Action:  config_add,
		},
		{
			Name:    "delete",
			Aliases: []string{"d"},
			Usage:   "删除订阅地址",
			Action:  config_delete,
		},
		{
			Name:    "update",
			Aliases: []string{"u"},
			Usage:   "更新所有的订阅",
			Action:  config_update,
		},
	},
	Action: config_list,
}

var Conn = &cli.Command{
	Name:        "conn",
	Usage:       "服务器管理",
	Description: "查看、变更",
	Subcommands: []*cli.Command{
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "查看所有的可用服务器",
			Action:  connect_list,
		},
		{
			Name:   "set",
			Usage:  "设置当前的服务器",
			Action: connect_set,
		},
	},
}
var Service = &cli.Command{
	Name:        "service",
	Usage:       "系统服务管理",
	Description: "开启、关闭",
	Subcommands: []*cli.Command{
		{
			Name:   "on",
			Usage:  "开启系统服务(root启动)",
			Action: service_on,
		},
		{
			Name:   "off",
			Usage:  "关闭系统服务(root启动)",
			Action: service_off,
		},
	},
}

func main() {
	// 输出banner
	fmt.Println(Banner)

	// 打开配置数据库
	db, err := sql.Open(dbDriverName, dbName)
	checkErr(err)
	pool = *db
	defer pool.Close()

	// 作者信息
	author := cli.Author{
		Name:  "fromhex",
		Email: "fromhex@163.com",
	}
	// 初始化命令行工具信息
	app := &cli.App{
		Name:      "V2rayCT",
		Usage:     "A cmd tool by golang of v2ray",
		UsageText: "v2rayct [global options] command subcommand",
		Version:   "v0.0.1",
		Authors:   []*cli.Author{&author},
		Before:    init_sqlite,
		Action:    config_update,
	}

	// 添加子命令
	app.Commands = []*cli.Command{Sub, Conn, Service}
	err = app.Run(os.Args)
	checkErr(err)
}

/* 检测错误 */
func checkErr(err error) {
	if err != nil {
		log.Fatal("\033[1;31m[ERRO] \033[0m", err)
	}
}

func checkErrMsg(err error, msg string) {
	if err != nil {
		log.Fatal("\033[1;31m[ERRO] \033[0m", msg)
	}
}

func checkNoErrMsg(err error, msg string) {
	if err == nil {
		log.Fatal("\033[1;31m[ERRO] \033[0m", msg)
	}
}

/* 初始化配置数据库（创建表） */
func init_sqlite(c *cli.Context) error {
	// 判断是否存在表 SELECT count(*) FROM sqlite_master where type='table' and name=?
	log.Println("初始化配置数据库")
	// 订阅表、订阅所有链接表、v2ray配置表（一般只有一项）
	/* Setting保存当前代理信息 */
	sql_table := `
	PRAGMA foreign_keys = ON;
	create table IF NOT EXISTS config(id integer PRIMARY KEY AUTOINCREMENT, subAddr text);

	create table IF NOT EXISTS server(
		id integer PRIMARY KEY AUTOINCREMENT, 
		config text,
		cid interger,
		FOREIGN KEY (cid) REFERENCES config(id) ON DELETE CASCADE
		);

	create table IF NOT EXISTS setting(serverId integer primary key);`

	_, err := pool.Exec(sql_table)
	return err
}

func sub_req(sub *Config) error {
	// 请求url获取服务器
	header := req.Header{
		"User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
	}
	req.SetTimeout(15 * time.Second)
	res, err := req.Get(sub.subAddr, header)
	checkErr(err)
	// 解码信息
	res_decoded, _ := base64.StdEncoding.DecodeString(res.String())
	tmp := strings.Replace(string(res_decoded), "vmess://", "", -1)
	all_encode := strings.Fields(tmp)

	// all_encode := strings.Split(tmp, "\n")
	for _, value := range all_encode {
		// base64解码
		server_json, _ := base64.StdEncoding.DecodeString(value)
		// 保存到数据库
		sqls, _ := pool.Prepare("insert into server values(null,?,?)")
		sqls.Exec(string(server_json), sub.id)
	}

	return nil
}

func config_list(c *cli.Context) error {
	sub_list, _ := config_query()
	log.Println("当前订阅链接数为: ", len(sub_list))
	for _, sub := range sub_list {
		log.Println(sub.id, sub.subAddr)
	}
	return nil
}

func config_add(c *cli.Context) error {

	log.Println("添加订阅")

	sub := c.Args().First()
	var tmp int
	err := pool.QueryRow(`select id from config where subAddr=?`, sub).Scan(&tmp)
	checkNoErrMsg(err, "已经存在订阅")
	// 插入订阅,使用自增id
	sqls, _ := pool.Prepare("insert into config values(null,?)")
	sqls.Exec(sub)
	log.Println("添加订阅成功")
	config_update(nil)
	return nil
}

func config_delete(c *cli.Context) error {
	log.Println("删除订阅")
	id, err := strconv.Atoi(c.Args().First())
	checkErr(err)

	sqls, _ := pool.Prepare("delete from config where id=?")
	_, err = sqls.Exec(id)
	log.Println("删除订阅完成")
	return err
}

func config_update(c *cli.Context) error {
	// 获取订阅链接，开始更新
	sub_list, _ := config_query()
	num := len(sub_list)
	log.Println("当前订阅链接数为: ", num)

	if num == 0 {
		log.Println("请添加订阅链接")
		return nil
	}
	log.Println("开始更新订阅")
	// 删除残留，重置自增ID
	sqls := `
	DELETE FROM server;
	DELETE FROM sqlite_sequence WHERE name='server';`
	pool.Exec(sqls)

	for _, sub := range sub_list {
		log.Println(sub.id, sub.subAddr)
		sub_req(sub)
	}
	// 输出所有的服务器
	log.Println("订阅更新完成")
	connect_list(c)
	server_set(4)
	return nil
}

func connect_list(c *cli.Context) error {

	log.Println("当前所有的可用服务器")
	servers, _ := pool.Query("select * from server order by id")

	log.Printf("%-5s%-25s%-10s%-s\n", "num", "address", "port", "remark")

	for servers.Next() {
		var server Server
		servers.Scan(&server.id, &server.config, &server.cid)
		// log.Println(server.config)
		// json解析字符串,输出到stdout
		serverJson := gjson.Parse(server.config)
		log.Printf("%-5d%-25s%-10d%-s\n", server.id, serverJson.Get("add"), serverJson.Get("port").Int(), serverJson.Get("remark"))
	}

	return nil
}

func connect_set(c *cli.Context) error {
	log.Println("设置v2ray服务器")
	num, err := strconv.Atoi(c.Args().First())
	checkErr(err)
	server_set(num)
	log.Println("设置v2ray服务器完成")
	return nil
}

/* 查询订阅配置库，返回一个指针数组 */
func config_query() (sub_list []*Config, err error) {
	subs, err := pool.Query("select * from config")
	for subs.Next() {
		var config Config

		subs.Scan(&config.id, &config.subAddr)
		sub_list = append(sub_list, &config)
	}
	return
}

/* 设置当前连接的服务器 */
func server_set(serverId int) error {

	// defaultConn := &Setting{1, serverId}
	// 先看是否是否有服务器信息，同时设置v2ray配置文件
	v2raySet(serverId)

	//查询连接配置库
	sqls, _ := pool.Prepare("delete from setting")
	_, err := sqls.Exec()
	checkErr(err)

	defalutSql, _ := pool.Prepare("insert into setting values(?)")
	_, err = defalutSql.Exec(serverId)
	checkErr(err)
	//重启服务
	service_on(nil)
	return nil
}

func service_on(c *cli.Context) error {
	log.Println("开启V2rayCT服务")
	// 修改服务文件内容
	pwd, _ := os.Getwd()
	execStart := fmt.Sprintf("ExecStart=%s/v2ray-core/v2ray -config %s/v2ray-core/config.json", pwd, pwd)
	v2rayServiceNew := strings.Replace(v2rayService, "ExecStart=", execStart, -1)
	// 创建v2ray.service服务文件
	sfile, err := os.OpenFile("/etc/systemd/system/v2ray.service", os.O_WRONLY|os.O_CREATE, 0644)
	checkErrMsg(err, "创建v2ray服务文件失败")
	_, err = sfile.WriteString(v2rayServiceNew)
	checkErrMsg(err, "写入v2ray服务文件失败")

	err = exec.Command("systemctl", "restart", "v2ray.service").Run()
	checkErrMsg(err, "systemctl服务启动失败")

	log.Println("开启V2rayCT服务成功")
	return nil
}

func service_off(c *cli.Context) error {
	log.Println("关闭V2rayCT服务")
	err := exec.Command("systemctl", "stop", "v2ray.service").Run()
	checkErrMsg(err, "systemctl服务启动失败")
	log.Println("关闭V2rayCT服务成功")
	return nil
}

// todo 根据模板修改v2ray配置
func v2raySet(serverId int) {
	// 查询num对应server信息
	var target Server
	err := pool.QueryRow("select * from server where id=?", serverId).Scan(&target.id, &target.config, &target.cid)
	checkErrMsg(err, "没有该服务器")

	serverJson := gjson.Parse(target.config)
	log.Println("当前设置的服务器为：")
	log.Printf("%-5d%-25s%-10d%-s\n", serverId, serverJson.Get("add"), serverJson.Get("port").Int(), serverJson.Get("remark"))

	/* 根据订阅中服务器信息修改v2ray配置 */
	v2rayConfig, _ := sjson.Set(v2rayConfig, "outbounds.0.settings.vnext.0.address", serverJson.Get("add").Str)
	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.settings.vnext.0.port", serverJson.Get("port").Int())
	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.settings.vnext.0.users.0.id", serverJson.Get("id").Str)
	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.settings.vnext.0.users.0.alterId", serverJson.Get("aid").Int())

	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.streamSettings.network", serverJson.Get("net").Str)
	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.streamSettings.wssettings.headers.Host", serverJson.Get("host").Str)
	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.streamSettings.wssettings.path", serverJson.Get("path").Str)

	pfile, err := os.OpenFile("v2ray-core/config.json", os.O_WRONLY|os.O_CREATE, 0644)
	checkErrMsg(err, "创建v2ray配置文件失败")
	_, err = pfile.WriteString(v2rayConfig)
	checkErrMsg(err, "写入v2ray配置文件失败")
}

const v2rayService = `[Unit]
Description=V2Ray Service
Documentation=https://www.v2fly.org/
After=network.target nss-lookup.target

[Service]
User=nobody
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
NoNewPrivileges=true
ExecStart=
Restart=on-failure
RestartPreventExitStatus=23

[Install]
WantedBy=multi-user.target`

const v2rayConfig = `
{
    "inbounds": [
        {
            "port": 1080,
            "protocol": "socks",
            "settings": {
                "auth": "noauth",
                "udp": true,
                "userLevel": 8
            },
            "sniffing": {
                "destOverride": [
                    "http",
                    "tls"
                ],
                "enabled": true
            },
            "tag": "socks"
        },
        {
            "port": 1081,
            "protocol": "http",
            "settings": {
                "userLevel": 8
            },
            "tag": "http"
        },
        {
            "tag": "transparent",
            "port": 12345,
            "protocol": "dokodemo-door",
            "settings": {
                "network": "tcp,udp",
                "followRedirect": true,
                "timeout": 30
            },
            "sniffing": {
                "enabled": true,
                "destOverride": [
                    "http",
                    "tls"
                ]
            },
            "streamSettings": {
                "sockopt": {
                    "tproxy": "tproxy"
                }
            }
        }
    ],
    "log": {
        "loglevel": "warning"
    },
    "outbounds": [
        {
            "mux": {
                "enabled": true
            },
            "protocol": "vmess",
            "settings": {
                "vnext": [
                    {
                        "address": "123",
                        "port": 19083,
                        "users": [
                            {
                                "id": "123",
                                "alterId": 0,
                                "security": "auto",
                                "level": 8
                            }
                        ]
                    }
                ]
            },
            "streamSettings": {
                "network": "ws",
                "security": "",
                "tlssettings": {
                    "allowInsecure": true,
                    "serverName": ""
                },
                "wssettings": {
                    "connectionReuse": true,
                    "headers": {
                        "Host": ""
                    },
                    "path": "123"
                },
                "sockopt": {
                    "mark": 255
                }
            },
            "tag": "proxy"
        },
        {
            "protocol": "freedom",
            "settings": {
                "domainStrategy": "UseIP"
            },
            "tag": "direct",
            "streamSettings": {
                "sockopt": {
                    "mark": 255
                }
            }
        },
        {
            "protocol": "blackhole",
            "settings": {
                "response": {
                    "type": "http"
                }
            },
            "tag": "block"
        },
        {
            "tag": "dns-out",
            "protocol": "dns",
            "streamSettings": {
                "sockopt": {
                    "mark": 255
                }
            }
        }
    ],
    "policy": {
        "levels": {
            "8": {
                "connIdle": 300,
                "downlinkOnly": 1,
                "handshake": 4,
                "uplinkOnly": 1
            }
        },
        "system": {
            "statsInboundUplink": true,
            "statsInboundDownlink": true
        }
    },
    "dns": {
        "servers": [
            "8.8.8.8",
            "1.1.1.1",
            "114.114.114.114",
            {
                "address": "223.5.5.5",
                "port": 53,
                "domains": [
                    "geosite:cn",
                    "ntp.org",
                    ""
                ]
            }
        ]
    },
    "routing": {
        "domainStrategy": "IPOnDemand",
        "rules": [
            {
                "type": "field",
                "inboundTag": [
                    "transparent"
                ],
                "port": 53,
                "network": "udp",
                "outboundTag": "dns-out"
            },
            {
                "type": "field",
                "inboundTag": [
                    "transparent"
                ],
                "port": 123,
                "network": "udp",
                "outboundTag": "direct"
            },
            {
                "type": "field",
                "ip": [
                    "223.5.5.5",
                    "114.114.114.114"
                ],
                "outboundTag": "direct"
            },
            {
                "type": "field",
                "ip": [
                    "8.8.8.8",
                    "1.1.1.1"
                ],
                "outboundTag": "proxy"
            },
            {
                "type": "field",
                "protocol": [
                    "bittorrent"
                ],
                "outboundTag": "direct"
            },
            {
                "type": "field",
                "ip": [
                    "geoip:private",
                    "geoip:cn"
                ],
                "outboundTag": "direct"
            },
            {
                "type": "field",
                "domain": [
                    "geosite:cn"
                ],
                "outboundTag": "direct"
            }
        ]
    },
    "stats": {}
}
`
