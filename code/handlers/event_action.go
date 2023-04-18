package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"start-feishubot/initialization"
	"start-feishubot/services"
	"start-feishubot/services/openai"
	"start-feishubot/utils"
	"start-feishubot/utils/audio"
	"strings"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type MsgInfo struct {
	handlerType HandlerType
	msgType     string
	msgId       *string
	chatId      *string
	qParsed     string
	fileKey     string
	imageKey    string
	sessionId   *string
	mention     []*larkim.MentionEvent
}
type ActionInfo struct {
	handler *MessageHandler
	ctx     *context.Context
	info    *MsgInfo
}

type Action interface {
	Execute(a *ActionInfo) bool
}

type ProcessedUniqueAction struct { //消息唯一性
}

func (*ProcessedUniqueAction) Execute(a *ActionInfo) bool {
	if a.handler.msgCache.IfProcessed(*a.info.msgId) {
		return false
	}
	a.handler.msgCache.TagProcessed(*a.info.msgId)
	return true
}

type ProcessMentionAction struct { //是否机器人应该处理
}

func (*ProcessMentionAction) Execute(a *ActionInfo) bool {
	// 私聊直接过
	if a.info.handlerType == UserHandler {
		return true
	}
	// 群聊判断是否提到机器人
	if a.info.handlerType == GroupHandler {
		if a.handler.judgeIfMentionMe(a.info.mention) {
			return true
		}
		return false
	}
	return false
}

type EmptyAction struct { /*空消息*/
}

func (*EmptyAction) Execute(a *ActionInfo) bool {
	if len(a.info.qParsed) == 0 {
		sendMsg(*a.ctx, "🤖️：你想知道什么呢~", a.info.chatId)
		fmt.Println("msgId", *a.info.msgId,
			"message.text is empty")
		return false
	}
	return true
}

type ClearAction struct { /*清除消息*/
}

func (*ClearAction) Execute(a *ActionInfo) bool {
	if _, foundClear := utils.EitherTrimEqual(a.info.qParsed,
		"/clear", "清除"); foundClear {
		sendClearCacheCheckCard(*a.ctx, a.info.sessionId,
			a.info.msgId)
		return false
	}
	return true
}

type RolePlayAction struct { /*角色扮演*/
}

func (*RolePlayAction) Execute(a *ActionInfo) bool {
	if system, foundSystem := utils.EitherCutPrefix(a.info.qParsed,
		"/system ", "角色扮演 "); foundSystem {
		a.handler.sessionCache.Clear(*a.info.sessionId)
		systemMsg := append([]openai.Messages{}, openai.Messages{
			Role: "system", Content: system,
		})
		a.handler.sessionCache.SetMsg(*a.info.sessionId, systemMsg)
		sendSystemInstructionCard(*a.ctx, a.info.sessionId,
			a.info.msgId, system)
		return false
	}
	return true
}

type HelpAction struct { /*帮助*/
}

func (*HelpAction) Execute(a *ActionInfo) bool {
	if _, foundHelp := utils.EitherTrimEqual(a.info.qParsed, "/help",
		"帮助"); foundHelp {
		sendHelpCard(*a.ctx, a.info.sessionId, a.info.msgId)
		return false
	}
	return true
}

type PicAction struct { /*图片*/
}

func (*PicAction) Execute(a *ActionInfo) bool {
	// 开启图片创作模式
	if _, foundPic := utils.EitherTrimEqual(a.info.qParsed,
		"/picture", "图片创作"); foundPic {
		a.handler.sessionCache.Clear(*a.info.sessionId)
		a.handler.sessionCache.SetMode(*a.info.sessionId,
			services.ModePicCreate)
		a.handler.sessionCache.SetPicResolution(*a.info.sessionId,
			services.Resolution256)
		sendPicCreateInstructionCard(*a.ctx, a.info.sessionId,
			a.info.msgId)
		return false
	}

	mode := a.handler.sessionCache.GetMode(*a.info.sessionId)
	//fmt.Println("mode: ", mode)

	// 收到一张图片,且不在图片创作模式下, 提醒是否切换到图片创作模式
	if a.info.msgType == "image" && mode != services.ModePicCreate {
		sendPicModeCheckCard(*a.ctx, a.info.sessionId, a.info.msgId)
		return false
	}

	if a.info.msgType == "image" && mode == services.ModePicCreate {
		//保存图片
		imageKey := a.info.imageKey
		//fmt.Printf("fileKey: %s \n", imageKey)
		msgId := a.info.msgId
		//fmt.Println("msgId: ", *msgId)
		req := larkim.NewGetMessageResourceReqBuilder().MessageId(
			*msgId).FileKey(imageKey).Type("image").Build()
		resp, err := initialization.GetLarkClient().Im.MessageResource.Get(context.Background(), req)
		//fmt.Println(resp, err)
		if err != nil {
			//fmt.Println(err)
			fmt.Sprintf("🤖️：图片下载失败，请稍后再试～\n 错误信息: %v", err)
			return false
		}

		f := fmt.Sprintf("%s.png", imageKey)
		resp.WriteFile(f)
		defer os.Remove(f)
		resolution := a.handler.sessionCache.GetPicResolution(*a.
			info.sessionId)

		openai.ConvertJpegToPNG(f)
		openai.ConvertToRGBA(f, f)

		//图片校验
		err = openai.VerifyPngs([]string{f})
		if err != nil {
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：无法解析图片，请发送原图并尝试重新操作～"),
				a.info.msgId)
			return false
		}
		bs64, err := a.handler.gpt.GenerateOneImageVariation(f, resolution)
		if err != nil {
			replyMsg(*a.ctx, fmt.Sprintf(
				"🤖️：图片生成失败，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}
		replayImagePlainByBase64(*a.ctx, bs64, a.info.msgId)
		return false

	}

	// 生成图片
	if mode == services.ModePicCreate {
		resolution := a.handler.sessionCache.GetPicResolution(*a.
			info.sessionId)
		bs64, err := a.handler.gpt.GenerateOneImage(a.info.qParsed,
			resolution)
		if err != nil {
			replyMsg(*a.ctx, fmt.Sprintf(
				"🤖️：图片生成失败，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}
		replayImageCardByBase64(*a.ctx, bs64, a.info.msgId, a.info.sessionId,
			a.info.qParsed)
		return false
	}

	return true
}

type MessageAction struct { /*消息*/
}

func (*MessageAction) Execute(a *ActionInfo) bool {
	msg := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	msg = append(msg, openai.Messages{
		Role: "user", Content: a.info.qParsed,
	})
	completions, err := a.handler.gpt.Completions(msg)
	if err != nil {
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	msg = append(msg, completions)
	a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)
	//if new topic
	if len(msg) == 2 {
		//fmt.Println("new topic", msg[1].Content)
		sendNewTopicCard(*a.ctx, a.info.sessionId, a.info.msgId,
			completions.Content)
		return false
	}
	err = replyMsg(*a.ctx, completions.Content, a.info.msgId)
	if err != nil {
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	return true
}

type AudioAction struct { /*语音*/
}

func (*AudioAction) Execute(a *ActionInfo) bool {
	// 只有私聊才解析语音,其他不解析
	if a.info.handlerType != UserHandler {
		return true
	}

	//判断是否是语音
	if a.info.msgType == "audio" {
		fileKey := a.info.fileKey
		//fmt.Printf("fileKey: %s \n", fileKey)
		msgId := a.info.msgId
		//fmt.Println("msgId: ", *msgId)
		req := larkim.NewGetMessageResourceReqBuilder().MessageId(
			*msgId).FileKey(fileKey).Type("file").Build()
		resp, err := initialization.GetLarkClient().Im.MessageResource.Get(context.Background(), req)
		//fmt.Println(resp, err)
		if err != nil {
			fmt.Println(err)
			return true
		}
		f := fmt.Sprintf("%s.ogg", fileKey)
		resp.WriteFile(f)
		defer os.Remove(f)

		//fmt.Println("f: ", f)
		output := fmt.Sprintf("%s.mp3", fileKey)
		// 等待转换完成
		audio.OggToWavByPath(f, output)
		defer os.Remove(output)
		//fmt.Println("output: ", output)

		text, err := a.handler.gpt.AudioToText(output)
		if err != nil {
			fmt.Println(err)

			sendMsg(*a.ctx, fmt.Sprintf("🤖️：语音转换失败，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}

		replyMsg(*a.ctx, fmt.Sprintf("🤖️：%s", text), a.info.msgId)
		//fmt.Println("text: ", text)
		a.info.qParsed = text
		return true
	}

	return true

}

type SQLAction struct { //生成sql并查询结果

}

func (*SQLAction) Execute(a *ActionInfo) bool {
	//判断是否是SQL
	if _, foundSQL := utils.EitherCutPrefix(a.info.qParsed,
		"/data ", "数据查询 "); foundSQL {
		msg := []openai.Messages{
			{Role: "system", Content: "你是一个SQL语句生成器，负责帮我生成SQL语句，语句基于Postgres语法。表结构信息如下："},
			{Role: "assistant", Content: "eth_dim.dim_addr_contracts每个合约一条记录，包含如下列：contract_address(string)合约地址，deployer（string）部署合约的地址，block_timestamp（bigint）合约的部署时间；"},
			{Role: "assistant", Content: "eth_dim.dim_addr_deposit_addresses每个充币地址一条记录，包含如下列：address（string）充币地址，exchange_name（string）充币地址所属交易所的名称"},
		}
		msg = append(msg, openai.Messages{
			Role: "user", Content: " 生成这个查询SQL: " + a.info.qParsed,
		})
		completions, err := a.handler.gpt.Completions(msg)
		if err != nil {
			replyMsg(*a.ctx, fmt.Sprintf(
				"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}
		start := strings.Index(completions.Content, "```")
		end := strings.Index(completions.Content[start+3:], "```")
		var sql string
		if start == -1 || end == -1 {
			replyMsg(*a.ctx, fmt.Sprintf(
				"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		} else {
			sql = completions.Content[start+3 : start+3+end]
		}
		holo := initialization.GetPGClient()
		rows, err := holo.Query(context.Background(), sql)
		if err != nil {
			replyMsg(*a.ctx, fmt.Sprintf(
				"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}
		msgReply := "|"
		// fmt.Printf("|")
		// 获取列描述
		colDescriptions := rows.FieldDescriptions()
		for _, v := range colDescriptions {
			// fmt.Printf("%v|", v.Name)
			msgReply += fmt.Sprintf("%v|", v.Name)
		}
		// fmt.Printf("\n")
		msgReply += "\n"
		// 遍历结果
		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				log.Fatalf("Failed to read row values: %v", err)
			}
			// fmt.Printf("|")
			msgReply += "|"
			for _, value := range values {
				// fmt.Printf("%v|", value)
				msgReply += fmt.Sprintf("%v|", value)
			}
			// fmt.Printf("\n")
			msgReply += "\n"
		}
		err = replyMsg(*a.ctx, msgReply, a.info.msgId)
		if err != nil {
			replyMsg(*a.ctx, fmt.Sprintf(
				"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}
		return false
	}
	return true
}
