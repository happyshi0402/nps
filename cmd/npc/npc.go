package main

import (
	"../../client"
	"../../lib"
	_ "../../lib"
	"flag"
	"strings"
)

const VERSION = "v0.0.13"

var (
	serverAddr = flag.String("server", "", "服务器地址ip:端口")
	verifyKey  = flag.String("vkey", "", "验证密钥")
	logType    = flag.String("log", "stdout", "日志输出方式（stdout|file）")
)

func main() {
	flag.Parse()
	lib.InitDaemon("npc")
	if *logType == "stdout" {
		lib.InitLogFile("npc", true)
	} else {
		lib.InitLogFile("npc", false)
	}
	stop := make(chan int)
	for _, v := range strings.Split(*verifyKey, ",") {
		lib.Println("客户端启动，连接：", *serverAddr, " 验证令牌：", v)
		go client.NewRPClient(*serverAddr, v).Start()
	}
	<-stop
}
