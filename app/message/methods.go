package message

import (
	"log"
	"regexp"
	"strings"

	"github.com/869413421/wechatbot/app/config"
	"github.com/869413421/wechatbot/app/session"
	"github.com/eatmoreapple/openwechat"
)

// removeMarkdown 去除markdown语法，转换为纯文本
func removeMarkdown(text string) string {
	// 去除代码块 ```code```
	text = regexp.MustCompile("(?s)```[\\w]*\\n(.*?)```").ReplaceAllString(text, "$1")
	
	// 去除行内代码 `code`
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllString(text, "$1")
	
	// 去除粗体 **text** 或 __text__
	text = regexp.MustCompile("\\*\\*([^*]+)\\*\\*").ReplaceAllString(text, "$1")
	text = regexp.MustCompile("__([^_]+)__").ReplaceAllString(text, "$1")
	
	// 去除斜体 *text* 或 _text_
	text = regexp.MustCompile("\\*([^*]+)\\*").ReplaceAllString(text, "$1")
	text = regexp.MustCompile("_([^_]+)_").ReplaceAllString(text, "$1")
	
	// 去除删除线 ~~text~~
	text = regexp.MustCompile("~~([^~]+)~~").ReplaceAllString(text, "$1")
	
	// 去除标题标记 # ## ### 等
	text = regexp.MustCompile("^#{1,6}\\s+(.+)$").ReplaceAllString(text, "$1")
	
	// 去除链接 [text](url) -> text
	text = regexp.MustCompile("\\[([^\\]]+)\\]\\([^\\)]+\\)").ReplaceAllString(text, "$1")
	
	// 去除图片 ![alt](url) -> alt
	text = regexp.MustCompile("!\\[([^\\]]*)\\]\\([^\\)]+\\)").ReplaceAllString(text, "$1")
	
	// 去除列表标记 - * + 和数字列表
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		// 去除无序列表标记
		line = regexp.MustCompile("^\\s*[-*+]\\s+").ReplaceAllString(line, "")
		// 去除有序列表标记
		line = regexp.MustCompile("^\\s*\\d+\\.\\s+").ReplaceAllString(line, "")
		result = append(result, line)
	}
	text = strings.Join(result, "\n")
	
	// 去除多余的空行（保留最多一个空行）
	text = regexp.MustCompile("\n{3,}").ReplaceAllString(text, "\n\n")
	
	return strings.TrimSpace(text)
}

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

	if requestText == "help" {
		reply = config.HelpText
	} else {
		// 移除角色修改功能，直接处理消息
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

	// 去除markdown语法并回复用户
	reply = removeMarkdown(reply)
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

	// 检查是否@了机器人
	isAt := msg.IsAt()
	self := sender.Self()
	selfNickName := self.NickName
	log.Printf("Group message IsAt() result: %v, Content contains @: %v, Bot nickname: %v", isAt, strings.Contains(msg.Content, "@"), selfNickName)
	
	// 不是@的不处理，但如果消息内容包含@和机器人昵称，也认为是@了机器人
	if !isAt {
		// 如果消息内容包含@符号和机器人昵称，可能是IsAt()方法失效，尝试通过内容判断
		if strings.Contains(msg.Content, "@") && strings.Contains(msg.Content, selfNickName) {
			log.Printf("Warning: Message contains @ and bot name but IsAt() returned false, processing anyway")
			// 继续处理
		} else {
			log.Printf("Group message not @ bot, skipping")
			return nil
		}
	}

	// 替换掉@文本，然后向GPT发起请求
	// self 和 selfNickName 已在上面定义
	replaceText := "@" + selfNickName
	requestText := strings.TrimSpace(strings.ReplaceAll(msg.Content, replaceText, ""))
	var reply string

	if requestText == "help" {
		reply = config.HelpText
	} else {
		// 移除角色修改功能，直接处理消息
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

	// 去除markdown语法并回复@我的用户
	reply = removeMarkdown(reply)
	reply = strings.TrimSpace(reply)
	reply = strings.Trim(reply, "\n")
	atText := "@" + groupSender.NickName
	replyText := atText + "\n--输入help查看帮助--\n" + reply
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
