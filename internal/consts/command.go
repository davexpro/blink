package consts

const (
	cmdBase = 31415926 // pi=3.1415926

	// 基本操作
	cmdTerminal       = cmdBase + 1 // 远程终端
	cmdScreenShot     = cmdBase + 2 // 屏幕快照
	cmdScreenRealtime = cmdBase + 3 // 实时屏幕
	cmdScreenControl  = cmdBase + 4 // 屏幕操作
	cmdDevice         = cmdBase + 5 // 设备信息
	cmdNetwork        = cmdBase + 6 // 网络信息

	// 进程操作相关
	cmdProcess     = cmdBase + 11 // 进程列表
	cmdProcessRun  = cmdBase + 12 // 启动进程
	cmdProcessKill = cmdBase + 13 // 结束进程

	// 文件操作相关
	cmdFileCreate   = cmdBase + 21
	cmdFileModify   = cmdBase + 22
	cmdFileDelete   = cmdBase + 23
	cmdFileUpload   = cmdBase + 24
	cmdFileDownload = cmdBase + 25

	// 操作系统相关
	cmdSystemShutdown  = cmdBase + 31 // 关机
	cmdSystemReboot    = cmdBase + 32 // 重启
	cmdSystemLogout    = cmdBase + 33 // 注销 or 登出
	cmdSystemLock      = cmdBase + 34 // 锁屏
	cmdSystemSleep     = cmdBase + 35 // 睡眠
	cmdSystemHibernate = cmdBase + 36

	// 客户端相关
	cmdUpgrade   = cmdBase + 91 // 升级客户端
	cmdUninstall = cmdBase + 98 // 卸载客户端
	cmdTerminate = cmdBase + 99 // 退出客户端
	x
)
