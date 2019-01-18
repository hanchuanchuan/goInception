// go-mysqlbinlog: a simple binlog tool to sync remote MySQL binlog.
// go-mysqlbinlog supports semi-sync mode like facebook mysqlbinlog.
// see http://yoshinorimatsunobu.blogspot.com/2014/04/semi-synchronous-replication-at-facebook.html
package session

// time go run mainRemote.go -start-time="2018-09-17 00:00:00" -stop-time="2018-09-25 00:00:00" -o=1.sql
import (
	"fmt"
	"github.com/imroc/req"
	log "github.com/sirupsen/logrus"
	"time"
)

// const (
//     // SOCKET_HOST = "127.0.0.1"
//     // SOCKET_PORT = 8090
//     // URL string = "http://127.0.0.1:8090/socket"
// )

var URL string

var header = req.Header{"Accept": "application/json"}

func init() {
	req.SetTimeout(5 * time.Second)
}

// 向客户端推送消息
func sendMsg(user string, event string, title string, text string, kwargs map[string]interface{}) bool {

	if user == "" {
		return true
	}
	if URL == "" {
		return false
	}

	url := fmt.Sprintf("%s/room/%s", URL, user)

	param := req.Param{
		"event":   event,
		"title":   title,
		"content": text,
	}

	if kwargs != nil && len(kwargs) > 0 {
		for k, v := range kwargs {
			param[k] = v
		}
	}

	r, err := req.Post(url, header, param)
	if err != nil {
		log.Error("请求websocket失败!")
		log.Print(err)
		return false
	}
	// r.ToJSON(&foo)       // 响应体转成对象
	// log.Printf("%+v", r) // 打印详细信息

	resp := r.Response()

	if resp.StatusCode == 200 {
		return true
	} else {
		log.Error("请求websocket失败!")
		log.Printf("%+v", r) // 打印详细信息
		return false
	}
}
