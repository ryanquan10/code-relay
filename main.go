package main

import "codex-relay/client"

func main() {
	go client.Claude.Relay()
	go client.Codex.Relay()
}
