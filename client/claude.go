package client

import "log"

type claudeRelay struct{}

func (c *claudeRelay) Relay() {
	log.Println("Claude Relay started")
	// Claude 中继逻辑
	select {}
}

// 导出全局变量
var Claude = &claudeRelay{}
