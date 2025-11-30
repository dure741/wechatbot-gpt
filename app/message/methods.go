package message

import (
	"log"
	"regexp"
	"strings"

	"github.com/869413421/wechatbot/app/config"
	"github.com/869413421/wechatbot/app/session"
	"github.com/eatmoreapple/openwechat"
)

// handlers 所有消息类型类型的处理器
var handlers map[HandlerType]MessageHandlerInterface

func init() {
	handlers = make(map[HandlerType]MessageHandlerInterface)
	handlers[GroupHandler] = NewGroupMessageHandler()
	handlers[UserHandler] = NewUserMessageHandler()
}

// Handler 全局处理入口
func Handler(msg *openwechat.Message) {
	log.Printf("hadler Received msg : %v", msg.Content)
	// 处理群消息
	if msg.IsSendByGroup() {
		handlers[GroupHandler].handle(msg)
		return
	}

	// 好友申请
	if msg.IsFriendAdd() {
		if config.LoadConfig().AutoPass {
			_, err := msg.Agree("你好我是基于chatGPT引擎开发的微信机器人，你可以向我提问任何问题。")
			if err != nil {
				log.Fatalf("add friend agree error : %v", err)
				return
			}
		}
	}

	// 私聊
	handlers[UserHandler].handle(msg)
}

// NewUserMessageHandler 创建私聊处理器
func NewUserMessageHandler() MessageHandlerInterface {
	return &UserMessageHandler{}
}

// handle 处理消息
func (g *UserMessageHandler) handle(msg *openwechat.Message) error {
	if msg.IsText() {
		return g.ReplyText(msg)
	}
	return nil
}

// ReplyText 发送文本消息到群
func (g *UserMessageHandler) ReplyText(msg *openwechat.Message) error {
	// 接收私聊消息
	sender, err := msg.Sender()
	log.Printf("Received User %v Text Msg : %v", sender.NickName, msg.Content)

	// 向GPT发起请求
	requestText := strings.TrimSpace(msg.Content)
	requestText = strings.Trim(msg.Content, "\n")
	var reply string

	reg := regexp.MustCompile(`role:(.*)`)
	if requestText == "help" {
		reply = config.HelpText
		// 判断requestText是否包含"role:"
	} else if reg.MatchString(requestText) {
		// 匹配role:后面的内容
		role := reg.FindStringSubmatch(requestText)[1]
		reply, err = session.Completions(g.getSessionId(sender), role, role)
	} else {
		reply, err = session.Completions(g.getSessionId(sender), requestText, "")
	}
	if err != nil {
		log.Printf("gtp request error: %v \n", err)
		msg.ReplyText("机器人神了，我一会发现了就去修。")
		return err
	}
	if reply == "" {
		log.Printf("WARNING: Empty reply received, sending default message\n")
		reply = "抱歉，我暂时无法处理这个请求，请稍后再试。"
	}

	// 回复用户
	reply = strings.TrimSpace(reply)
	reply = strings.Trim(reply, "\n")
	log.Printf("Sending reply to user: %s\n", reply)
	_, err = msg.ReplyText(reply)
	if err != nil {
		log.Printf("response user error: %v \n", err)
	}
	return err
}

// getSessionId 获取用户会话ID
func (g *UserMessageHandler) getSessionId(user *openwechat.User) string {
	return user.NickName + "-" + user.UserName
}

// NewGroupMessageHandler 创建群消息处理器
func NewGroupMessageHandler() MessageHandlerInterface {
	return &GroupMessageHandler{}
}

// handle 处理消息
func (g *GroupMessageHandler) handle(msg *openwechat.Message) error {
	if msg.IsText() {
		return g.ReplyText(msg)
	}
	return nil
}

// ReplyText 发送文本消息到群
func (g *GroupMessageHandler) ReplyText(msg *openwechat.Message) error {
	// 接收群消息
	sender, err := msg.Sender()
	group := openwechat.Group{sender}
	log.Printf("Received Group %v Text Msg : %v", group.NickName, msg.Content)

	// 不是@的不处理
	if !msg.IsAt() {
		return nil
	}

	// 替换掉@文本，然后向GPT发起请求
	self := sender.Self()
	replaceText := "@" + self.NickName
	requestText := strings.TrimSpace(strings.ReplaceAll(msg.Content, replaceText, ""))
	var reply string

	reg := regexp.MustCompile(`role:(.*)`)
	if requestText == "help" {
		reply = config.HelpText
		// 判断requestText是否包含"role:"
	} else if reg.MatchString(requestText) {
		// 匹配role:后面的内容
		role := reg.FindStringSubmatch(requestText)[1]
		reply, err = session.Completions(g.getSessionId(group), role, role)
	} else {
		reply, err = session.Completions(g.getSessionId(group), requestText, "")
	}
	if err != nil {
		log.Printf("gtp request error: %v \n", err)
		msg.ReplyText("机器人神了，我一会发现了就去修。")
		return err
	}
	if reply == "" {
		log.Printf("WARNING: Empty reply received, sending default message\n")
		reply = "抱歉，我暂时无法处理这个请求，请稍后再试。"
	}

	// 获取@我的用户
	groupSender, err := msg.SenderInGroup()
	if err != nil {
		log.Printf("get sender in group error :%v \n", err)
		return err
	}

	// 回复@我的用户
	reply = strings.TrimSpace(reply)
	reply = strings.Trim(reply, "\n")
	atText := "@" + groupSender.NickName
	replyText := atText + `
	--输入"help"查看帮助--
	` + reply
	log.Printf("Sending reply to group: %s\n", replyText)
	_, err = msg.ReplyText(replyText)
	if err != nil {
		log.Printf("response group error: %v \n", err)
	}
	return err
}

// 通过group生成sessionId
func (g *GroupMessageHandler) getSessionId(group openwechat.Group) string {
	return group.NickName + "-" + group.UserName
}
