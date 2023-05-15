package http

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	comm "github.com/lxf9601/go-common"
	"github.com/lxf9601/go-common/logc"

	"github.com/valyala/fasthttp"
)

const (
	CONTENT_TYPE_HTML = "text/html; charset=utf-8"
	CONTENT_TYPE_JSON = "application/json;  charset=utf-8"
)

type ApiResponse struct {
	Ret  int         `json:"ret"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type Response struct {
	StatusCode  int
	Data        interface{}
	ContentType string
}

type TplModel map[string]interface{}

type Controller struct {
}

func (this *Controller) Success(data interface{}) *ApiResponse {
	res := ApiResponse{}
	res.Msg = ""
	res.Data = data
	return &res
}

func (this *Controller) Fail(ret int, msg string) *ApiResponse {
	res := ApiResponse{}
	res.Ret = ret
	res.Msg = msg
	return &res
}

type HttpContext struct {
	RawCtx      *fasthttp.RequestCtx
	contentType string
}

func (this *HttpContext) BindForm(objValue reflect.Value, tag string) {
	objType := objValue.Type()
	for i := 0; i < objType.Elem().NumField(); i++ {
		field := objType.Elem().Field(i)
		name := field.Type.Name()
		f := objValue.Elem().Field(i)
		if name == "string" {
			f.SetString(this.FormString(field.Tag.Get(tag)))
		} else if strings.Index(name, "uint") == 0 {
			f.SetUint(this.FormUint64(field.Tag.Get(tag)))
		} else if strings.Index(name, "int") == 0 {
			f.SetInt(this.FormInt64(field.Tag.Get(tag)))
		} else if strings.Index(name, "float") == 0 {
			f.SetFloat(this.FormFloat64(field.Tag.Get(tag)))
		} else if name == "Position" {
			f.SetString(this.FormString(field.Tag.Get(tag)))
		}
	}
}

func (this *HttpContext) GetContentType() string {
	return this.contentType
}

func (this *HttpContext) SetContentType(contentType string) {
	this.contentType = contentType
	this.RawCtx.Response.Header.Set("Content-Type", contentType)
}

func (this *HttpContext) FormJSON(key string) map[string]interface{} {
	var obj interface{}
	json.Unmarshal(this.RawCtx.FormValue(key), &obj)
	return obj.(map[string]interface{})
}

func (this *HttpContext) FormStringSlice(key string) []string {
	str := this.FormString(key)
	if str != "" {
		return strings.Split(this.FormString(key), ",")
	} else {
		return make([]string, 0, 0)
	}
}

func (this *HttpContext) FormKeyExists(key string) bool {
	if len(this.RawCtx.FormValue(key)) == 0 {
		return false
	} else {
		return true
	}
}

func (this *HttpContext) FormString(key string) string {
	return string(this.RawCtx.FormValue(key))
}

func (this *HttpContext) FormBool(key string) bool {
	i, err := strconv.ParseBool(string(this.RawCtx.FormValue(key)))
	if err != nil {
		return false
	} else {
		return i
	}
}

func (this *HttpContext) FormUint32(key string) uint32 {
	i, err := strconv.ParseUint(string(this.RawCtx.FormValue(key)), 10, 32)
	if err != nil {
		return uint32(0)
	} else {
		return uint32(i)
	}
}

func (this *HttpContext) FormUint64(key string) uint64 {
	i, err := strconv.ParseUint(string(this.RawCtx.FormValue(key)), 10, 64)
	if err != nil {
		return uint64(0)
	} else {
		return i
	}
}

func (this *HttpContext) FormUint(key string) uint {
	i, err := strconv.ParseUint(string(this.RawCtx.FormValue(key)), 10, 32)
	if err != nil {
		return uint(0)
	} else {
		return uint(i)
	}
}

func (this *HttpContext) FormInt64(key string) int64 {
	i, err := strconv.ParseInt(string(this.RawCtx.FormValue(key)), 10, 64)
	if err != nil {
		return 0
	} else {
		return i
	}
}

func (this *HttpContext) FormInt(key string) int {
	i, err := strconv.Atoi(string(this.RawCtx.FormValue(key)))
	if err != nil {
		return 0
	} else {
		return i
	}
}

func (this *HttpContext) FormFloat64(key string) float64 {
	i, err := strconv.ParseFloat(string(this.RawCtx.FormValue(key)), 64)
	if err != nil {
		return 0
	} else {
		return i
	}
}

type Router struct {
	routerMap map[string]*RouterLocation
}

type RouterGroup struct {
	Interceptors []Interceptor // 拦截器
	Url          string        // 路径
	Router       *Router
}

type RouterLocation struct {
	Controller *interface{}
	Handler    string
	Method     string
	IsAuth     bool
	group      *RouterGroup
}

// interface definition
type Interceptor interface {
	BeforeHandle(controller *interface{}, ctx *HttpContext) (*Response, error)
	AfterHandle(controller *interface{}, ctx *HttpContext, resp interface{})
}

func (this *Router) Init() {
	this.routerMap = make(map[string]*RouterLocation, 100)
}

func (this *Router) Group(url string, interceptors []Interceptor, handler func(group *RouterGroup)) {
	group := &RouterGroup{Url: url, Interceptors: interceptors, Router: this}
	handler(group)
}

func (this *RouterGroup) Get(url string, controller *interface{}, handler string) {
	this.Router.routerMap[this.Url+url] = &RouterLocation{Controller: controller, Handler: handler,
		Method: http.MethodGet, group: this}
}

func (this *RouterGroup) Post(url string, controller *interface{}, handler string) {
	this.Router.routerMap[this.Url+url] = &RouterLocation{Controller: controller, Handler: handler,
		Method: http.MethodPost, group: this}
}

func (this *Router) Match(url string) *RouterLocation {
	return this.routerMap[url]
}

func (this *Router) Get(url string, controller *interface{}, handler string) {
	this.routerMap[url] = &RouterLocation{Controller: controller, Handler: handler, Method: http.MethodGet}
}

func (this *Router) Post(url string, controller *interface{}, handler string) {
	this.routerMap[url] = &RouterLocation{Controller: controller, Handler: handler, Method: http.MethodPost}
}

func (this *Router) Any(url string, controller *interface{}, handler string) {
	this.routerMap[url] = &RouterLocation{Controller: controller, Handler: handler, Method: http.MethodPost}
}

func (this *Router) AnyAuth(url string, controller *interface{}, handler string) {
	this.routerMap[url] = &RouterLocation{Controller: controller, Handler: handler, Method: http.MethodPost, IsAuth: true}
}

func ShowTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func HttpHandler(appPath string, router *Router) func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			if err := recover(); err != nil {
				logc.Error(err)
				logc.Error(string(comm.PanicTrace(5)))
			}
		}()
		if bytes.HasPrefix(ctx.Path(), []byte("/static")) {
			fs := &fasthttp.FS{
				Root:               appPath + "views",
				IndexNames:         []string{"index.html"},
				GenerateIndexPages: true,
				Compress:           true,
				AcceptByteRange:    true,
			}
			fs.PathRewrite = fasthttp.NewPathSlashesStripper(1)
			fs.NewRequestHandler()(ctx)
			return
		} else if string(ctx.Path()) == "/" || bytes.HasSuffix(ctx.Path(), []byte(".js")) ||
			bytes.HasSuffix(ctx.Path(), []byte(".png")) ||
			bytes.HasSuffix(ctx.Path(), []byte(".html")) ||
			bytes.HasSuffix(ctx.Path(), []byte(".css")) ||
			bytes.HasPrefix(ctx.Path(), []byte("/fonts")) {
			fs := &fasthttp.FS{
				Root:               appPath + "views",
				IndexNames:         []string{"index.html"},
				GenerateIndexPages: true,
				Compress:           true,
				AcceptByteRange:    true,
			}
			fs.PathRewrite = fasthttp.NewPathSlashesStripper(0)
			fs.NewRequestHandler()(ctx)
			return
		}
		n := router.Match(string(ctx.Path()))
		if n == nil {
			view := string(ctx.Path())[1:]
			tplPath := path.Join(appPath+"views", view+".tpl")
			f, err := os.Open(tplPath)
			if err == nil {
				defer f.Close()
				ctx.SetContentType(CONTENT_TYPE_HTML)
				t := template.New("").Funcs(template.
					FuncMap{"ShowTime": ShowTime})
				t, err = t.ParseGlob(path.Join(appPath+"views/common", "*.tpl"))
				t, err = t.ParseFiles(tplPath)
				err = t.ExecuteTemplate(ctx.Response.BodyWriter(), view+".tpl", nil)
				if err != nil {
					ctx.Write(ctx.Path())
				}
			} else {
				ctx.Write(ctx.Path())
			}
			return
		}
		c := new(HttpContext)
		c.RawCtx = ctx
		if n.IsAuth {
			if len(ctx.Request.Header.Cookie("user_name")) == 0 {
				ctx.Redirect("/m_login", 200)
				return
			}
		}
		if n.group != nil && n.group.Interceptors != nil {
			for _, interceptor := range n.group.Interceptors {
				resp, err := interceptor.BeforeHandle(n.Controller, c)
				if err != nil {
					if resp == nil {
						ctx.SetStatusCode(500)
						ctx.Response.Reset()
						ctx.SetBodyString(err.Error())
					} else if resp.ContentType == CONTENT_TYPE_JSON {
						j, _ := json.Marshal(resp.Data)
						ctx.Write(j)
					}
					return
				}
			}
		}
		if strings.ToLower(string(ctx.Method())) == "options" {
			return
		}
		v := reflect.ValueOf(*n.Controller)
		m := v.MethodByName(n.Handler)
		if m.IsZero() {
			logc.Errorf(string(ctx.Path()))
			return
		}
		if m.IsNil() {
			ctx.NotFound()
			return
		}
		params := make([]reflect.Value, 1)
		params[0] = reflect.ValueOf(c)
		vl := m.Call(params)

		if len(vl) > 0 {
			if vl[0].Type().String() != "string" {
				if c.GetContentType() != CONTENT_TYPE_HTML {
					c.SetContentType(CONTENT_TYPE_JSON)
					j, _ := json.Marshal(vl[0].Interface())
					if n.group != nil && n.group.Interceptors != nil {
						for _, interceptor := range n.group.Interceptors {
							interceptor.AfterHandle(n.Controller, c, j)
						}
					}
					encoding := string(ctx.Request.Header.Peek("Accept-Encoding"))
					if len(j) > 1024 && strings.Index(encoding, "gzip") != -1 {
						ctx.Response.Header.Add("Content-Encoding", "gzip")
						w := gzip.NewWriter(ctx.Response.BodyWriter())
						defer w.Close()
						w.Write(j)
					} else {
						ctx.Write(j)
						if logc.IsDebug() {
							logc.Debug(ctx.Request.URI().String() + ">>" + string(j))
						}
					}
				}
			} else {
				c.SetContentType(CONTENT_TYPE_HTML)
				view, _ := vl[0].Interface().(string)
				if view != "" {
					tplPath := appPath + "views/" + view + ".tpl"
					f, err := os.Open(tplPath)
					if err == nil {
						defer f.Close()
						t := template.New("").Funcs(template.
							FuncMap{"ShowTime": ShowTime})
						t.ParseFiles()
						t, err = t.ParseGlob(path.Join(appPath+"views/common", "*.tpl"))
						t, err = t.ParseFiles(path.Join(appPath+"views", view+".tpl"))
						err = t.ExecuteTemplate(ctx.Response.BodyWriter(), view+".tpl", vl[1].Interface())
					}
				}
			}

		}
	}

}
