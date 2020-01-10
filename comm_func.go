package comm

import (
	"bytes"
	"deploy/comm"
	"errors"
	"os"
	"runtime"
	"time"

	"strconv"
	"strings"

	"os/exec"
	"path/filepath"

	"reflect"

	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/configor"
)

func Dump(o interface{}) {
	spew.Dump(o)
}

func NowTime() string {
	return time.Now().Local().Format(TIME_LAYOUT)
}

const TIME_LAYOUT = "2006-01-02 15:04:05"
const DATE_LAYOUT = "2006-01-02"
const YEAR_LAYOUT = "2006"

func PanicTrace(kb int) []byte {
	s := []byte("/src/runtime/panic.go")
	e := []byte("\ngoroutine ")
	line := []byte("\n")
	stack := make([]byte, kb<<10) //4KB
	length := runtime.St协成通讯协议ack(stack, true)
	start := bytes.Index(stack, s)
	stack = stack[start:length]
	start = bytes.Index(stack, line) + 1
	stack = stack[start:]
	end := bytes.LastIndex(stack, line)
	if end != -1 {
		stack = stack[:end]
	}
	end = bytes.Index(stack, e)
	if end != -1 {
		stack = stack[:end]
	}
	stack = bytes.TrimRight(stack, "\n")
	return stack
}

func GetAppPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		i = strings.LastIndex(path, "\\")
	}
	if i < 0 {
		return "", errors.New(`error: Can't find "/" or "\".`)
	}
	return string(path[0 : i+1]), nil
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// IpToAton ip地址转为整数
func IpToAton(ip string) uint {
	bits := strings.Split(ip, ".")

	b0, _ := strconv.Atoi(bits[0])
	b1, _ := strconv.Atoi(bits[1])
	b2, _ := strconv.Atoi(bits[2])
	b3, _ := strconv.Atoi(bits[3])

	var sum uint

	sum += uint(b0) << 24
	sum += uint(b1) << 16
	sum += uint(b2) << 8
	sum += uint(b3)

	return sum
}

// 加载配置
func LoadConfig(config interface{}, configFile string) {
	appPath, _ := comm.GetAppPath()
	appPathField := reflect.ValueOf(config).Elem().FieldByName("AppPath")
	appPathField.SetString(appPath)
	confPath := appPath + configFile
	exists, _ := comm.PathExists(confPath)
	if !exists {
		path, _ := filepath.Abs("./")
		appPath = path + "/"
		appPathField.SetString(appPath)
		confPath = appPath + configFile
		exists, _ := comm.PathExists(confPath)
		if !exists {
			path, _ := filepath.Abs("../")
			appPath = path + "/"
			appPathField.SetString(appPath)
			confPath = appPath + configFile
		}
	}
	configor.Load(&config, confPath)
}

// 解析
func JWTParse(tokenStr string, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if token != nil {
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			return claims, err
		} else {
			return nil, err
		}
	} else {
		return nil, errors.New("无效的token")
	}
}

func FormatFloat32(val float32, digit int) float32 {
	formatVal, _ := strconv.ParseFloat(fmt.Sprintf("%."+strconv.Itoa(digit)+"f", val), 64)
	return float32(formatVal)
}

func FormatFloat64(val float64, digit int) float64 {
	formatVal, _ := strconv.ParseFloat(fmt.Sprintf("%."+strconv.Itoa(digit)+"f", val), 64)
	return formatVal
}
