package handlers

import (
	"start-feishubot/initialization"
	"start-feishubot/services"
	"start-feishubot/services/openai"
	"testing"
)

func TestSQLAction(t *testing.T) {
	config := initialization.LoadConfig("../config.yaml")
	initialization.LoadPGClient(*config)
	gpt := openai.NewChatGPT(*config)
	msgInfo := MsgInfo{
		handlerType: GroupHandler,
		msgType:     "sqldata",
		qParsed:     "生成这个查询SQL: 查询哪些交易所部署的合约最多",
	}
	data := &ActionInfo{
		ctx: nil,
		handler: &MessageHandler{
			sessionCache: services.GetSessionCache(),
			msgCache:     services.GetMsgCache(),
			gpt:          gpt,
			config:       *config,
		},
		info: &msgInfo,
	}
	(&SQLAction{}).Execute(data)
}
