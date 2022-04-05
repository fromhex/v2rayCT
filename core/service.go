package core

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

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

var Service = &cli.Command{
	Name:        "service",
	Usage:       "系统服务管理",
	Description: "开启、关闭",
	Subcommands: []*cli.Command{
		{
			Name:   "on",
			Usage:  "开启系统服务(root启动)",
			Action: StartService,
		},
		{
			Name:   "off",
			Usage:  "关闭系统服务(root启动)",
			Action: StopService,
		},
	},
}

func StartService(c *cli.Context) error {
	log.Println("开启V2rayCT服务")
	// 修改服务文件内容
	p, _ := os.Executable()
	pwd := filepath.Dir(p)
	execStart := fmt.Sprintf("ExecStart=%s/v2ray-core/v2ray -config %s/v2ray-core/config.json", pwd, pwd)
	v2rayServiceNew := strings.Replace(v2rayService, "ExecStart=", execStart, -1)
	// 创建v2ray.service服务文件
	sfile, err := os.OpenFile("/etc/systemd/system/v2ray.service", os.O_WRONLY|os.O_CREATE, 0644)
	CheckErrMsg(err, "创建v2ray服务文件失败")
	_, err = sfile.WriteString(v2rayServiceNew)
	CheckErrMsg(err, "写入v2ray服务文件失败")

	err = exec.Command("systemctl", "enable", "v2ray.service").Run()
	CheckErrMsg(err, "systemctl服务设置开机自启动失败")

	err = exec.Command("systemctl", "restart", "v2ray.service").Run()
	CheckErrMsg(err, "systemctl服务启动失败")

	log.Println("开启V2rayCT服务成功")
	return nil
}

func StopService(c *cli.Context) error {
	log.Println("关闭V2rayCT服务")
	err := exec.Command("systemctl", "stop", "v2ray.service").Run()
	CheckErrMsg(err, "systemctl服务启动失败")
	log.Println("关闭V2rayCT服务成功")
	return nil
}
