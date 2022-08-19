package edm

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"strconv"
	"time"

	"cparrow.com/common/logc"
	"github.com/valyala/fasthttp"
)

const (
	EVENT_DELIVERED    = 1
	EVENT_OPENED       = 2
	EVENT_CLICKED      = 3
	EVENT_UNSUBSCRIBED = 4
	EVENT_COMPLAINED   = 5
	EVENT_DROPPED      = 6
	EVENT_BOUNCED      = 7

	MG_TIME_LAYOUT = "Mon, _2 Jan 2006 15:04:05 -0700 (MST)"

	SUPPLIER_MAILGUN  = 1
	SUPPLIER_SENDGRID = 2
)

var CampEventType = map[string]int8{
	"delivered":    1, // 送达
	"opened":       2, // 打开
	"clicked":      3, // 点击
	"unsubscribed": 4, // 退订
	"complained":   5, // 投诉
	"dropped":      6, // 退信
	"bounced":      7, // 硬反弹
}

const (
	TAGS_USER      = "user"
	TAGS_CAMP      = "camp"
	TAGS_SEND_TIME = "send"
)

// Mailgun发送任务
type MailRequest struct {
	FormName    string        // 发件人名称
	FormAddress string        // 发件人地址
	Subject     string        // 邮件主题
	TextHtml    string        // 邮件内容
	TextPlain   string        // 邮件文本
	IsTracking  bool          // 是否启用跟踪
	Tags        []string      // 任务标签
	Variables   MailVariables // 任务变量
	Headers     MailHeaders   // 邮件头参数
	ToList      []*MailgunTo  // 收个人列表
}

type MailVariables map[string]string
type MailHeaders map[string]string

// Mailgun收件人
type MailgunTo struct {
	Name      string        // 收件人名称
	Address   string        // 收件人地址
	Variables MailVariables // 变量
}

// Mailgun接口授权
type Auth struct {
	Url string
	Key string
}

type MailResponse struct {
	Id      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
	IsOk    bool   `json:"-"`
}

type MailgunQuery struct {
	BeginTime   *time.Time // 开始时间
	EndTime     *time.Time // 结束时间
	Ascending   bool       // 正向排序
	MessageId   string     // MailgunID
	Limit       int        // 限制条数
	IsTemporary bool       // 严重程度是否临时性错误
	PageUrl     string     // 数据地址
	EventType   string     // 事件类型
	Tags        string     // 自定义标签
	To          string     // 邮件地址
}

type Event struct {
	SendTime    time.Time   // 发送时间
	SyncId      string      // 同步ID
	EventType   int8        // 事件类型
	DataTime    time.Time   // 数据时间
	Email       string      // Email地址
	ToName      string      // 收件人名字
	Desc        string      // 错误的描述
	Url         string      // 点击的链接
	ClientInfo  *ClientInfo // 客户端信息
	MxHost      string      // 邮件服务器mx主机
	UserId      uint        // 用户ID
	CampId      uint        // 活动ID
	BouncedType int8        // 0 非反弹 1 硬反弹 2 延时硬反弹
}

type ClientInfo struct {
	ClientOs   string // 操作系统
	DeviceType string // 设备类型
	ClientName string // 客户端名字
	ClientType string // 客户端类型
	UserAgent  string // 用户代理
	Ip         string // IP地址
}

type MailgunLogPageData struct {
	List     []*Event
	Next     string
	Previous string
	Last     string
	First    string
}

// ListEvent 查询mailgun日志
func ListMgEvent(query *MailgunQuery, auth *Auth) (
	*MailgunLogPageData, error) {
	pageData := new(MailgunLogPageData)
	req := fasthttp.AcquireRequest()
	rep := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rep)
	req.Header.SetMethodBytes([]byte("GET"))
	userPwd := base64.StdEncoding.EncodeToString([]byte("api:" + auth.Key))
	req.Header.AddBytesKV([]byte("Authorization"), []byte("Basic "+userPwd))
	req.Header.AddBytesKV([]byte("Accept-Encoding"), []byte("gzip, deflate"))
	params := map[string]string{}
	if query.PageUrl == "" {
		if query.MessageId != "" {
			params["message-id"] = query.MessageId
		}
		if query.Tags != "" {
			params["tags"] = query.Tags
		}
		if query.Limit > 0 {
			params["limit"] = strconv.Itoa(query.Limit)
		}
		if query.Ascending {
			params["ascending"] = "yes"
		} else {
			params["ascending"] = "no"
		}
		if query.EventType != "" {
			params["event"] = query.EventType
		}
		if query.To != "" {
			params["to"] = query.To
		}
		if params["event"] == "failed" {
			if query.IsTemporary {
				params["severity"] = "temporary"
			} else {
				params["severity"] = "permanent"
			}
		}
		if query.BeginTime != nil {
			params["begin"] = query.BeginTime.Format(MG_TIME_LAYOUT)
		}
		if query.EndTime != nil {
			params["end"] = query.EndTime.Format(MG_TIME_LAYOUT)
		}
		query.PageUrl = auth.Url + "/events?"
		buf := bytes.Buffer{}
		for k, v := range params {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(k)
			buf.WriteByte('=')

			buf.WriteString(url.QueryEscape(v))
		}
		query.PageUrl += buf.String()
		req.Header.SetRequestURI(query.PageUrl)
	} else {
		req.Header.SetRequestURI(query.PageUrl)
	}
	client := fasthttp.Client{}
	err := client.DoTimeout(req, rep, 60*time.Second)
	if err != nil {
		return pageData, err
	}
	if rep.StatusCode() != 200 {
		return pageData, errors.New("http error:" + strconv.Itoa(rep.StatusCode()))
	}

	queryResult := map[string]interface{}{}
	err = json.Unmarshal(rep.Body(), &queryResult)
	if err == nil {
		if queryResult["items"] != nil {
			items := queryResult["items"].([]interface{})
			itemLen := len(items)
			var item map[string]interface{}
			var event *Event
			for i := 0; i < itemLen; i++ {
				item = items[i].(map[string]interface{})
				variables := item["user-variables"].(map[string]interface{})
				event = new(Event)
				if len(variables) > 0 {
					campId, _ := strconv.ParseUint(variables["camp_id"].(string), 10, 32)
					event.CampId = uint(campId)
					userId, _ := strconv.ParseUint(variables["user_id"].(string), 10, 32)
					event.UserId = uint(userId)
					sendTime, _ := strconv.ParseUint(variables["send_time"].(string), 10, 32)
					event.SendTime = time.Unix(int64(sendTime), 0)
				} else {
					tags := item["tags"].([]interface{})
					event = new(Event)
					for _, v := range tags {
						tag := v.(string)
						if tag[0:4] == TAGS_CAMP {
							campId, _ := strconv.ParseUint(tag[5:], 10, 32)
							event.CampId = uint(campId)
						} else if tag[0:4] == TAGS_USER {
							userId, _ := strconv.ParseUint(tag[5:], 10, 32)
							event.UserId = uint(userId)
						} else if tag[0:4] == TAGS_SEND_TIME {
							sendTime, _ := strconv.ParseUint(tag[5:], 10, 32)
							event.SendTime = time.Unix(int64(sendTime), 0)
						}
					}
				}
				timestamp := int64(math.Round(item["timestamp"].(float64)))
				if item["recipient"] != nil {
					event.Email = item["recipient"].(string)
				}
				if item["url"] != nil {
					event.Url = item["url"].(string)
				}
				if item["event"] == "failed" {
					status := item["delivery-status"].(map[string]interface{})
					if status["message"] != nil {
						event.Desc = status["message"].(string)
					} else if status["description"] != nil {
						event.Desc = status["description"].(string)
					}
					if status["mx-host"] != nil {
						event.MxHost = status["mx-host"].(string)
					}
					event.EventType = EVENT_DROPPED
					if item["reason"] == "bounce" || item["reason"] == "old" {
						flags := item["flags"].(map[string]interface{})
						if flags["is-delayed-bounce"].(bool) {
							event.BouncedType = 2
						} else {
							event.BouncedType = 1
						}
					}
				} else {
					event.EventType = CampEventType[item["event"].(string)]
				}
				h := md5.New()
				h.Write([]byte(strconv.FormatUint(uint64(event.CampId), 10) + "-" +
					strconv.FormatInt(timestamp, 10) + "-" +
					event.Email + "-" + strconv.Itoa(int(event.EventType))))

				cipherStr := h.Sum(nil)
				event.SyncId = hex.EncodeToString(cipherStr)
				if event.EventType == EVENT_OPENED ||
					event.EventType == EVENT_CLICKED {
					clientInfo := item["client-info"].(map[string]interface{})
					event.ClientInfo = new(ClientInfo)
					event.ClientInfo.ClientName = clientInfo["client-name"].(string)
					event.ClientInfo.ClientType = clientInfo["client-type"].(string)
					event.ClientInfo.ClientOs = clientInfo["client-os"].(string)
					event.ClientInfo.UserAgent = clientInfo["user-agent"].(string)
					event.ClientInfo.DeviceType = clientInfo["device-type"].(string)

					if item["ip"] != nil {
						event.ClientInfo.Ip = item["ip"].(string)
					}
				}

				event.DataTime = time.Unix(timestamp, 0)
				pageData.List = append(pageData.List, event)
			}
			if queryResult["paging"] != nil {
				pageing := queryResult["paging"].(map[string]interface{})
				if pageing["next"] != nil {
					pageData.Next = pageing["next"].(string)
				}
				if pageing["last"] != nil {
					pageData.Last = pageing["last"].(string)
				}
				if pageing["first"] != nil {
					pageData.First = pageing["first"].(string)
				}
				if pageing["previous"] != nil {
					pageData.Previous = pageing["previous"].(string)
				}
			}
		} else {
			logc.Error("mailgun event api error:%s", err)
			return nil, errors.New("mailgun api error")
		}
	} else {
		return nil, err
	}
	return pageData, nil
}
