package edm

var SendGridEventType = map[string]int8{
	"delivered":   1, // 送达
	"open":        2, // 打开
	"click":       3, // 点击
	"unsubscribe": 4, // 退订
	"spamreport":  5, // 投诉
	"dropped":     6, // 退信
	"bounce":      7, // 硬反弹
}

type SendGridClientInfo struct {
	DeviceType string // 设备类型
	UserAgent  string // 用户代理
	Ip         string // IP地址
}
