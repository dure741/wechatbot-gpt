package bootstrap

import (
	"github.com/869413421/wechatbot/app/message"
	"github.com/869413421/wechatbot/app/task"
	"github.com/eatmoreapple/openwechat"
	"log"
)

var globalBot *openwechat.Bot

func Run() {
	//bot := openwechat.DefaultBot()
	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式，上面登录不上的可以尝试切换这种模式
	globalBot = bot

	// 注册消息处理函数
	bot.MessageHandler = message.Handler
	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 启动任务提醒服务
	startTaskReminderService()

	// 创建热存储容器对象
	reloadStorage := openwechat.NewJsonFileHotReloadStorage("storage.json")
	// 执行热登录
	err := bot.HotLogin(reloadStorage)
	if err != nil {
		if err = bot.Login(); err != nil {
			log.Printf("login error: %v \n", err)
			return
		}
	}
	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}

// startTaskReminderService 启动任务提醒服务
func startTaskReminderService() {
	task.StartReminderService(func(tasks []*task.Task) {
		// 当有任务需要提醒时，记录日志
		// 注意：这里暂时只记录日志，实际发送消息需要在消息处理模块中实现
		for _, t := range tasks {
			log.Printf("Task reminder: %s (Due: %s)\n", t.Title, t.DueTime.Format("2006-01-02 15:04:05"))
		}
	})
}

