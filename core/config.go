package core

import (
	"database/sql"
	"encoding/base64"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/imroc/req/v3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
)

/* 保存订阅信息 */
type Config struct {
	id      int
	subAddr string
}

/* 保存服务器信息 */
type Server struct {
	id     int
	config string
	cid    int
}

var Sub = &cli.Command{
	Name:        "config",
	Usage:       "订阅管理",
	Description: "查看、添加、删除、更新",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "add",
			Aliases: []string{"a"},
			Usage:   "添加vmess订阅地址",
		},
		&cli.IntFlag{
			Name:    "delete",
			Aliases: []string{"d"},
			Usage:   "删除vmess订阅地址",
		},
	},
	Action: ListConfig,
}

func ListConfig(c *cli.Context) error {
	pool, _ := InitDB(c)
	if c.String("add") != "" {
		AddConfig(pool, c.String("add"))
		return nil
	}
	if c.Int("delete") != 0 {
		DeleteConfig(pool, c.Int("delete"))
		return nil
	}

	UpdateConfig(pool)
	return nil
}

func AddConfig(pool *sql.DB, sub string) error {
	var tmp int
	err := pool.QueryRow(`select id from config where subAddr=?`, sub).Scan(&tmp)
	CheckNoErrMsg(err, "已经存在订阅")
	sqls, _ := pool.Prepare("insert into config values(null,?)")
	sqls.Exec(sub)
	log.Println("添加订阅成功")
	return nil
}

func DeleteConfig(pool *sql.DB, id int) error {
	sqls, _ := pool.Prepare("delete from config where id=?")
	_, err := sqls.Exec(id)
	log.Println("删除订阅完成")
	return err
}

/* 更新config中的订阅 */
func UpdateConfig(pool *sql.DB) error {
	var tmp int
	pool.QueryRow("select count(1) from config").Scan(&tmp)
	if tmp == 0 {
		CheckMsg("请添加订阅")
	}
	log.Println("当前订阅链接数为: ", tmp)

	// 删除残留，重置自增ID
	sqls := `
	DELETE FROM server;
	DELETE FROM sqlite_sequence WHERE name='server';`
	pool.Exec(sqls)

	// 获取数据库中的订阅链接
	subs, _ := pool.Query("select * from config")
	// 添加goruntime同步
	var wg sync.WaitGroup
	for subs.Next() {
		var config Config
		subs.Scan(&config.id, &config.subAddr)
		// 列出现有的订阅
		log.Println(config.id, config.subAddr)
		// 异步请求订阅的服务器
		wg.Add(1)
		go ReqSub(pool, &config, &wg)
	}
	wg.Wait()
	log.Println("更新订阅完成")
	return nil
}

func ReqSub(pool *sql.DB, sub *Config, wg *sync.WaitGroup) error {
	defer wg.Done()
	// 请求url获取服务器
	req.SetTimeout(15 * time.Second)
	res, err := req.SetHeader("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.84 Safari/537.36").Get(sub.subAddr)
	CheckErrMsg(err, "请求订阅失败")
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

/* 初始化配置数据库（创建表） */
func InitDB(c *cli.Context) (pool *sql.DB, err error) {
	// fmt.Print(Banner)
	dbDriverName := c.String("dbDriverName")
	dbName := c.String("dbName")
	log.Println("初始化配置数据库")
	pool, _ = sql.Open(dbDriverName, dbName)

	// 输出banner
	// 判断是否存在表 SELECT count(*) FROM sqlite_master where type='table' and name=?

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

	_, err = pool.Exec(sql_table)
	return pool, err
}
