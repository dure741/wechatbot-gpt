package handlers

import (
	"log"
	"regexp"
	"strings"

	"github.com/869413421/wechatbot/config"
	"github.com/869413421/wechatbot/gtp"
	"github.com/eatmoreapple/openwechat"
)

var _ MessageHandlerInterface = (*UserMessageHandler)(nil)

// UserMessageHandler 私聊消息处理
type UserMessageHandler struct {
}

// handle 处理消息
func (g *UserMessageHandler) handle(msg *openwechat.Message) error {
	if msg.IsText() {
		return g.ReplyText(msg)
	}
	return nil
}

// NewUserMessageHandler 创建私聊处理器
func NewUserMessageHandler() MessageHandlerInterface {
	return &UserMessageHandler{}
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
		reply, err = gtp.Completions(g.getSessionId(sender), role, role)
	} else {
		reply, err = gtp.Completions(g.getSessionId(sender), requestText, "")
	}
	if err != nil {
		log.Printf("gtp request error: %v \n", err)
		msg.ReplyText("机器人神了，我一会发现了就去修。")
		return err
	}
	if reply == "" {
		return nil
	}

	// 回复用户
	reply = strings.TrimSpace(reply)
	reply = strings.Trim(reply, "\n")
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
