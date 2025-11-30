package message

import "github.com/eatmoreapple/openwechat"

// MessageHandlerInterface 消息处理接口
type MessageHandlerInterface interface {
	handle(*openwechat.Message) error
	ReplyText(*openwechat.Message) error
}

// UserMessageHandler 私聊消息处理
type UserMessageHandler struct {
}

// GroupMessageHandler 群消息处理
type GroupMessageHandler struct {
}

var _ MessageHandlerInterface = (*UserMessageHandler)(nil)
var _ MessageHandlerInterface = (*GroupMessageHandler)(nil)
