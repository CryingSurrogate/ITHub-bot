package main

import (
	"github.com/nats-io/nats.go"
)

func initiateNats() {
	nc, _ := nats.Connect("nats://95.165.107.100:4222")
	nc.Subscribe("ith.bot.ads", func(msg *nats.Msg) {
		// decode msg
		// send msg to all chats in cache
		// reply to nats msg with result
	})

}
