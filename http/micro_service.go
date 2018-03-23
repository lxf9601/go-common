package http

import (
	"github.com/valyala/fasthttp"
	"time"
	"strconv"
	"errors"
	"cparrow.com/common/logc"
	"encoding/json"
)

// 微服务API
type MicroSrvApi struct {
	client fasthttp.Client // http客户端
	Url string             // 服务接口地址
}

type MicroSrvApiOptions struct {
	Url string // 服务接口地址
}

func NewMicroSrvApi(options *MicroSrvApiOptions) *MicroSrvApi {
	api := &MicroSrvApi{client: fasthttp.Client{}, Url: options.Url}
	return api
}

// Get 获取资源
func (this *MicroSrvApi) Get(uri string) (*ApiResponse, error) {
	req := fasthttp.AcquireRequest()
	rep := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(rep)
	req.Header.SetMethodBytes([]byte("GET"))
	reqUri := this.Url + "/" + uri
	req.Header.SetRequestURI(reqUri)
	err := this.client.DoTimeout(req, rep, 60*time.Second)
	if err != nil {
		logc.Error(err)
		return nil, err
	}
	if rep.StatusCode() != 200 {
		err := "http " + reqUri + "error:" + strconv.Itoa(rep.StatusCode())
		logc.Error(err)
		return nil, errors.New(err)
	}
	response := new(ApiResponse)
	err = json.Unmarshal(rep.Body(), response)
	if err != nil {
		err := "http " + reqUri + "error:" + string(rep.Body())
		logc.Error(err)
		return nil, errors.New(err)
	}
	return response, nil
}