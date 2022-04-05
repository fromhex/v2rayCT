package core

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/urfave/cli/v2"
)

var Conn = &cli.Command{
	Name:        "conn",
	Usage:       "服务器管理",
	Description: "查看、变更、测延迟",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "set",
			Aliases:     []string{"s"},
			Usage:       "设置当前的服务器",
			Value:       -1,
			DefaultText: "-1",
		},
		&cli.BoolFlag{
			Name:        "ping",
			Aliases:     []string{"p"},
			Usage:       "测试当前的服务器延迟",
			Value:       false,
			DefaultText: "false",
		},
	},
	Action: ListConn,
}

func PingConn() error {
	client := req.C()
	client.SetProxyURL("http://127.0.0.1:1081")
	resp, err := client.R().Get("http://www.google.com")
	CheckErr(err)
	if resp.StatusCode == 200 {
		log.Println("当前服务器的延迟为：", resp.TotalTime().Milliseconds(), "ms")
	}
	return err
}

func ListConn(c *cli.Context) error {
	pool, _ := InitDB(c)

	if c.Int("set") != -1 {
		SetConn(pool, c.Int("set"))
		return nil
	}

	if c.Bool("ping") {
		PingConn()
		return nil
	}

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

func SetConn(pool *sql.DB, serverId int) error {

	// defaultConn := &Setting{1, serverId}
	// 先看是否是否有服务器信息，同时设置v2ray配置文件
	v2raySet(pool, serverId)

	//查询连接配置库
	sqls, _ := pool.Prepare("delete from setting")
	_, err := sqls.Exec()
	CheckErr(err)

	defalutSql, _ := pool.Prepare("insert into setting values(?)")
	_, err = defalutSql.Exec(serverId)
	CheckErr(err)
	//重启服务
	StartService(nil)
	log.Println("设置v2ray服务器完成")
	return nil
}

// todo 根据模板修改v2ray配置
func v2raySet(pool *sql.DB, serverId int) {
	// 查询num对应server信息
	var target Server
	err := pool.QueryRow("select * from server where id=?", serverId).Scan(&target.id, &target.config, &target.cid)
	CheckErrMsg(err, "没有该服务器")

	serverJson := gjson.Parse(target.config)
	log.Println("设置服务器为：")
	log.Printf("%-5d%-25s%-10d%-s\n", serverId, serverJson.Get("add"), serverJson.Get("port").Int(), serverJson.Get("remark"))

	/* 根据订阅中服务器信息修改v2ray配置 */
	v2rayConfig, _ := sjson.Set(v2rayConfig, "outbounds.0.settings.vnext.0.address", serverJson.Get("add").Str)
	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.settings.vnext.0.port", serverJson.Get("port").Int())
	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.settings.vnext.0.users.0.id", serverJson.Get("id").Str)
	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.settings.vnext.0.users.0.alterId", serverJson.Get("aid").Int())

	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.streamSettings.network", serverJson.Get("net").Str)
	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.streamSettings.wssettings.headers.Host", serverJson.Get("host").Str)
	v2rayConfig, _ = sjson.Set(v2rayConfig, "outbounds.0.streamSettings.wssettings.path", serverJson.Get("path").Str)

	p, _ := os.Executable()
	pwd := filepath.Dir(p)
	pfile, err := os.OpenFile(pwd+"/v2ray-core/config.json", os.O_WRONLY|os.O_CREATE, 0644)
	CheckErrMsg(err, "创建v2ray配置文件失败")
	_, err = pfile.WriteString(v2rayConfig)
	CheckErrMsg(err, "写入v2ray配置文件失败")
}

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
