package main

import (
	"fmt"
	"os"
	"path/filepath"
	"v2rayCT/core"

	"github.com/urfave/cli/v2"
)

const (
	Banner = `
__      _____                   _____ _______
\ \    / /__ \                 / ____|__   __|
 \ \  / /   ) |_ __ __ _ _   _| |       | |
  \ \/ /   / /| '__/ _' | | | | |       | |
   \  /   / /_| | | (_| | |_| | |____   | |
    \/   |____|_|  \__,_|\__, |\_____|  |_|
                          __/ |
                         |___/
`
)

func main() {

	// 打开配置数据库
	// 作者信息

	p, _ := os.Executable()
	path := filepath.Dir(p)

	author := cli.Author{
		Name:  "fromhex",
		Email: "fromhex@163.com",
	}
	// 初始化命令行工具信息
	app := &cli.App{
		Name:                 "V2rayCT",
		Usage:                "A cmd tool by golang of v2ray",
		Version:              "v0.0.2",
		Authors:              []*cli.Author{&author},
		EnableBashCompletion: true,
		Before: func(ctx *cli.Context) error {
			fmt.Print(Banner)
			return nil
		},
		Action: core.ListConfig,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dbDriverName",
				Value:       "sqlite3",
				DefaultText: "sqlite3",
			},
			&cli.StringFlag{
				Name:        "dbName",
				Value:       path + "/config.db",
				DefaultText: path + "/config.db",
			},
		},
	}
	// 添加子命令
	app.Commands = []*cli.Command{core.Sub, core.Conn, core.Service}
	err := app.Run(os.Args)
	core.CheckErr(err)
}
