package app

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	randMath "math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/DanielRenne/GoCore/core/dbServices"
	"github.com/DanielRenne/GoCore/core/fileCache"
	"github.com/DanielRenne/GoCore/core/ginServer"
	"github.com/DanielRenne/GoCore/core/logger"
	"github.com/DanielRenne/GoCore/core/serverSettings"
	"github.com/DanielRenne/GoCore/core/store"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketRemoval func(info WebSocketConnectionMeta)
type customLog func(desc string, message string)

var CustomLog customLog

type WebSocketConnection struct {
	sync.RWMutex
	Id                   string
	Connection           *websocket.Conn
	Req                  *http.Request
	Context              interface{}
	ContextString        string
	ContextType          string
	ContextLock          sync.RWMutex
	WriteLock            sync.RWMutex
	LastResponseTime     time.Time
	LastResponseTimeLock sync.RWMutex
}

type WebSocketConnectionMeta struct {
	Conn             *WebSocketConnection
	Context          interface{}
	ContextString    string
	ContextType      string
	LastResponseTime time.Time
}

type WebSocketConnectionCollection struct {
	sync.RWMutex
	Connections []*WebSocketConnection
}

type ConcurrentWebSocketConnectionItem struct {
	Index int
	Conn  *WebSocketConnection
}

func (wscc *WebSocketConnectionCollection) Append(item *WebSocketConnection) {
	wscc.Lock()
	defer wscc.Unlock()
	wscc.Connections = append(wscc.Connections, item)

	if store.OnChange != nil {
		go func() {
			defer func() {
				if recover := recover(); recover != nil {
					log.Println("Panic Recovered at store.OnChange():  ", recover)
					return
				}
			}()

			store.OnChange(store.WebSocketStoreKey, "", store.PathAdd, nil, nil)
		}()
	}
}

func (wscc *WebSocketConnectionCollection) Iter() <-chan ConcurrentWebSocketConnectionItem {
	c := make(chan ConcurrentWebSocketConnectionItem)

	f := func() {
		wscc.RLock()
		defer wscc.RUnlock()
		for index := range wscc.Connections {
			value := wscc.Connections[index]
			c <- ConcurrentWebSocketConnectionItem{index, value}
		}
		close(c)
	}
	go f()

	return c
}

type WebSocketCallbackSync struct {
	sync.RWMutex
	callbacks []WebSocketCallback
}

type ConcurrentWebSocketCallbackItem struct {
	Index    int
	Callback WebSocketCallback
}

func (self *WebSocketCallbackSync) Append(item WebSocketCallback) {
	self.RLock()
	defer self.RUnlock()
	self.callbacks = append(self.callbacks, item)
}

func (self *WebSocketCallbackSync) Iter() <-chan ConcurrentWebSocketCallbackItem {
	c := make(chan ConcurrentWebSocketCallbackItem)

	f := func() {
		self.Lock()
		defer self.Unlock()
		for index := range self.callbacks {
			value := self.callbacks[index]
			c <- ConcurrentWebSocketCallbackItem{index, value}
		}
		close(c)
	}
	go f()

	return c
}

type WebSocketPubSubPayload struct {
	Key     string      `json:"Key"`
	Content interface{} `json:"Content"`
}

type WebSocketCallback func(conn *WebSocketConnection, c *gin.Context, messageType int, id string, data []byte)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var WebSocketConnections WebSocketConnectionCollection
var webSocketConnectionsMeta sync.Map
var WebSocketCallbacks WebSocketCallbackSync
var WebSocketRemovalCallback WebSocketRemoval

func Initialize(path string, config string) (err error) {
	err = serverSettings.Initialize(path, config)
	if err != nil {
		return
	}

	serverSettings.WebConfigMutex.RLock()
	inRelease := serverSettings.WebConfig.Application.ReleaseMode == "release"
	serverSettings.WebConfigMutex.RUnlock()

	if inRelease {
		ginServer.Initialize(gin.ReleaseMode, serverSettings.WebConfig.Application.CookieDomain)
	} else {
		ginServer.Initialize(gin.DebugMode, serverSettings.WebConfig.Application.CookieDomain)
	}
	fileCache.Initialize()

	err = dbServices.Initialize()
	if err != nil {
		return
	}
	return
}

func InitializeLite() (err error) {
	ginServer.InitializeLite(gin.ReleaseMode)
	fileCache.Initialize()
	return
}

func RunLite(port int) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Panic Recovered at RunLite():  ", r)
			time.Sleep(time.Millisecond * 3000)
			RunLite(port)
			return
		}
	}()

	ginServer.Router.GET("/ws", func(c *gin.Context) {
		webSocketHandler(c.Writer, c.Request, c)
	})

	log.Println("GoCore Application Started")

	ginServer.Router.Run(":" + strconv.Itoa(port))

}

func Run() {

	defer func() {
		if r := recover(); r != nil {
			log.Println("Panic Recovered at Run():  ", r)
			time.Sleep(time.Millisecond * 3000)
			Run()
			return
		}
	}()

	if serverSettings.WebConfig.Application.WebServiceOnly == false {

		loadHTMLTemplates()

		ginServer.Router.Static("/web", serverSettings.APP_LOCATION+"/web")

		ginServer.Router.GET("/ws", func(c *gin.Context) {
			webSocketHandler(c.Writer, c.Request, c)
		})
	}

	initializeStaticRoutes()

	go ginServer.Router.RunTLS(":"+strconv.Itoa(serverSettings.WebConfig.Application.HttpsPort), serverSettings.APP_LOCATION+"/keys/cert.pem", serverSettings.APP_LOCATION+"/keys/key.pem")

	log.Println("GoCore Application Started")

	ginServer.Router.Run(":" + strconv.Itoa(serverSettings.WebConfig.Application.HttpPort))

	// go ginServer.Router.GET("/", func(c *gin.Context) {
	// 	c.Redirect(http.StatusMovedPermanently, "https://"+serverSettings.WebConfig.Application.Domain+":"+strconv.Itoa(serverSettings.WebConfig.Application.HttpsPort))
	// })

}

func webSocketHandler(w http.ResponseWriter, r *http.Request, c *gin.Context) {

	// return
	defer func() {
		if recover := recover(); recover != nil {
			log.Println("Panic Recovered at webSocketHandler():  ", recover)
			time.Sleep(time.Millisecond * 3000)
			webSocketHandler(w, r, c)
			return
		}
	}()
	//log.Println("Web Socket Connection")
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		if CustomLog != nil {
			CustomLog("app->webSocketHandler", "Failed to upgrade http connection to websocket:  "+err.Error())
		}
		log.Println("Failed to upgrade http connection to websocket:  " + err.Error())
		return
	}

	//Start the Reader, listen for Close Message, and Add to the Connection Array.

	wsConn := new(WebSocketConnection)
	wsConn.Connection = conn
	wsConn.Req = r
	uuid, err := newUUID()
	if err == nil {
		wsConn.Id = uuid
	} else {
		uuid = randomString(20)
		wsConn.Id = uuid
	}

	SetWebSocketMeta(uuid, WebSocketConnectionMeta{LastResponseTime: time.Now(), Conn: wsConn})

	if CustomLog != nil {
		CustomLog("app->webSocketHandler", "Added Web Socket Connection from "+wsConn.Connection.RemoteAddr().String())
	}

	//Reader
	go logger.GoRoutineLogger(func() {

		defer func() {
			if recover := recover(); recover != nil {
				log.Println("Panic Recovered at webSocketHandler-> Reader():  ", recover)
			}
		}()

		for {
			messageType, p, err := conn.ReadMessage()
			if err == nil {

				meta, ok := GetWebSocketMeta(uuid)
				if ok {
					meta.LastResponseTime = time.Now()
					SetWebSocketMeta(uuid, meta)
				}

				go logger.GoRoutineLogger(func() {

					defer func() {
						if recover := recover(); recover != nil {
							log.Println("Panic Recovered at webSocketHandler-> Reader-> item.Callback():  ", recover)
						}
					}()

					for item := range WebSocketCallbacks.Iter() {
						if item.Callback != nil {
							item.Callback(wsConn, c, messageType, uuid, p)
						}
					}
				}, "GoCore/app.go->webSocketHandler[Callback calls]")
			} else {
				if CustomLog != nil {
					CustomLog("app->deleteWebSocket", "Deleting Web Socket from read Timeout:  "+err.Error()+":  "+wsConn.Connection.RemoteAddr().String())
				}
				deleteWebSocket(wsConn)
				return
			}
		}
	}, "GoCore/app.go->webSocketHandler[Reader]")

	WebSocketConnections.Append(wsConn)

}

func CloseAllSockets() {

	items := []*WebSocketConnection{}

	for item := range WebSocketConnections.Iter() {
		wsConn := item.Conn
		items = append(items, wsConn)
	}

	for i := range items {
		connection := items[i]
		connection.Connection.UnderlyingConn().Close()
		// deleteWebSocket(connection)
	}

}

func loadHTMLTemplates() {

	if serverSettings.WebConfig.Application.HtmlTemplates.Enabled {

		levels := "/*"
		dirLevel := ""

		switch serverSettings.WebConfig.Application.HtmlTemplates.DirectoryLevels {
		case 0:
			levels = "/*"
			dirLevel = ""
		case 1:
			levels = "/**/*"
			dirLevel = "root/"
		case 2:
			levels = "/**/**/*"
			dirLevel = "root/root/"
		}

		ginServer.Router.LoadHTMLGlob(serverSettings.APP_LOCATION + "/web/" + serverSettings.WebConfig.Application.HtmlTemplates.Directory + levels)

		ginServer.Router.GET("", func(c *gin.Context) {
			c.HTML(http.StatusOK, dirLevel+"index.tmpl", gin.H{})
		})
	} else {

		if serverSettings.WebConfig.Application.DisableRootIndex {
			return
		}

		ginServer.Router.GET("", func(c *gin.Context) {
			if serverSettings.WebConfig.Application.RootIndexPath == "" {
				ginServer.ReadHTMLFile(serverSettings.APP_LOCATION+"/web/index.htm", c)
			} else {
				ginServer.ReadHTMLFile(serverSettings.APP_LOCATION+"/web/"+serverSettings.WebConfig.Application.RootIndexPath, c)
			}
		})
	}
}

func initializeStaticRoutes() {

	ginServer.Router.GET("/swagger", func(c *gin.Context) {
		// c.Redirect(http.StatusMovedPermanently, "https://"+serverSettings.WebConfig.Application.Domain+":"+strconv.Itoa(serverSettings.WebConfig.Application.HttpsPort)+"/web/swagger/dist/index.html")

		ginServer.ReadHTMLFile(serverSettings.APP_LOCATION+"/web/swagger/dist/index.html", c)
	})
}

func RegisterWebSocketDataCallback(callback WebSocketCallback) {
	WebSocketCallbacks.Append(callback)
}

func ReplyToWebSocket(conn *WebSocketConnection, data []byte) {
	defer func() {
		if recover := recover(); recover != nil {
			log.Println("Panic Recovered at ReplyToWebSocket():  ", recover)
			time.Sleep(time.Millisecond * 3000)
			ReplyToWebSocket(conn, data)
			return
		}

	}()

	go logger.GoRoutineLogger(func() {
		defer func() {
			if recover := recover(); recover != nil {
				CustomLog("app->ReplyToWebSocket", "Panic Recovered at ReplyToWebSocket():  "+fmt.Sprintf("%+v", recover))
			}
		}()
		conn.WriteLock.Lock()
		defer conn.WriteLock.Unlock()
		conn.Connection.WriteMessage(websocket.BinaryMessage, data)

	}, "GoCore/app.go->ReplyToWebSocket[WriteMessage]")

}

func ReplyToWebSocketJSON(conn *WebSocketConnection, v interface{}) {

	defer func() {
		if recover := recover(); recover != nil {
			log.Println("Panic Recovered at ReplyToWebSocketJSON():  ", recover)
			time.Sleep(time.Millisecond * 3000)
			ReplyToWebSocketJSON(conn, v)
			return
		}
	}()

	go logger.GoRoutineLogger(func() {
		defer func() {
			if recover := recover(); recover != nil {
				CustomLog("app->ReplyToWebSocketJSON", "Panic Recovered at ReplyToWebSocketJSON():  "+fmt.Sprintf("%+v", recover))
			}
		}()
		conn.WriteLock.Lock()
		defer conn.WriteLock.Unlock()
		conn.Connection.SetWriteDeadline(time.Now().Add(time.Duration(10000) * time.Millisecond))
		conn.Connection.WriteJSON(v)

	}, "GoCore/app.go->ReplyToWebSocketJSON[WriteJSON]")

}

func ReplyToWebSocketPubSub(conn *WebSocketConnection, key string, v interface{}) {
	defer func() {
		if recover := recover(); recover != nil {
			conn.WriteLock.Unlock()
		}
	}()

	var payload WebSocketPubSubPayload
	payload.Key = key
	payload.Content = v

	go func() {
		defer func() {
			if recover := recover(); recover != nil {
				CustomLog("app->ReplyToWebSocketPubSub", "Panic Recovered at ReplyToWebSocketPubSub():  "+fmt.Sprintf("%+v", recover))
			}
		}()
		conn.WriteLock.Lock()
		defer conn.WriteLock.Unlock()
		conn.Connection.SetWriteDeadline(time.Now().Add(time.Duration(10000) * time.Millisecond))
		conn.Connection.WriteJSON(payload)
	}()

}

func BroadcastWebSocketData(data []byte) {

	defer func() {
		if recover := recover(); recover != nil {
			log.Println("Panic Recovered at WebSocketConnections():  ", recover)
			time.Sleep(time.Millisecond * 3000)
			BroadcastWebSocketData(data)
			return
		}
	}()

	for item := range WebSocketConnections.Iter() {
		conn := item.Conn
		go logger.GoRoutineLogger(func() {
			defer func() {
				if recover := recover(); recover != nil {
					CustomLog("app->BroadcastWebSocketData", "Panic Recovered at BroadcastWebSocketData():  "+fmt.Sprintf("%+v", recover))
				}
			}()
			conn.WriteLock.Lock()
			defer conn.WriteLock.Unlock()
			conn.Connection.WriteMessage(websocket.BinaryMessage, data)

		}, "GoCore/app.go->BroadcastWebSocketData[WriteMessage]")
	}
}

func BroadcastWebSocketJSON(v interface{}) {
	defer func() {
		if recover := recover(); recover != nil {
			log.Println("Panic Recovered at BroadcastWebSocketJSON():  ", recover)
			time.Sleep(time.Millisecond * 3000)
			BroadcastWebSocketJSON(v)
			return
		}
	}()

	for item := range WebSocketConnections.Iter() {
		conn := item.Conn
		go logger.GoRoutineLogger(func() {
			defer func() {
				if recover := recover(); recover != nil {
					CustomLog("app->BroadcastWebSocketJSON", "Panic Recovered at BroadcastWebSocketJSON():  "+fmt.Sprintf("%+v", recover))
				}
			}()
			conn.WriteLock.Lock()
			defer conn.WriteLock.Unlock()
			conn.Connection.SetWriteDeadline(time.Now().Add(time.Duration(10000) * time.Millisecond))
			conn.Connection.WriteJSON(v)

		}, "GoCore/app.go->BroadcastWebSocketData[WriteJSON]")
	}
}

func PublishWebSocketJSON(key string, v interface{}) {
	defer func() {
		if recover := recover(); recover != nil {
			log.Println("Panic Recovered at PublishWebSocketJSON():  ", recover)
			time.Sleep(time.Millisecond * 3000)
			PublishWebSocketJSON(key, v)
			return
		}
	}()
	var payload WebSocketPubSubPayload
	payload.Key = key
	payload.Content = v

	//Serialize and Deserialize to prevent Race Conditions from caller.
	data, _ := json.Marshal(payload)
	json.Unmarshal(data, &payload)

	for item := range WebSocketConnections.Iter() {
		conn := item.Conn
		go logger.GoRoutineLogger(func() {
			defer func() {
				if recover := recover(); recover != nil {
					CustomLog("app->PublishWebSocketJSON", "Panic Recovered at PublishWebSocketJSON():  "+fmt.Sprintf("%+v", recover))
				}
			}()
			conn.WriteLock.Lock()
			defer conn.WriteLock.Unlock()
			conn.Connection.SetWriteDeadline(time.Now().Add(time.Duration(10000) * time.Millisecond))
			conn.Connection.WriteJSON(payload)

		}, "GoCore/app.go->WriteJSON")
	}
}

func SetWebSocketTimeout(timeout int) {
	defer func() {
		if recover := recover(); recover != nil {
			log.Println("Panic Recovered at SetWebSocketTimeout():  ", recover)
			time.Sleep(time.Millisecond * 3000)
			SetWebSocketTimeout(timeout)
			return
		}
	}()

	// if CustomLog != nil {
	// 	CustomLog("app->SetWebSocketTimeout", "Checking for Web Socket Timeouts.")
	// }

	webSocketConnectionsMeta.Range(func(key interface{}, value interface{}) bool {
		meta, ok := value.(WebSocketConnectionMeta)
		if ok {
			if meta.LastResponseTime.Add(time.Millisecond * time.Duration(timeout)).Before(time.Now()) {
				if CustomLog != nil {
					CustomLog("app->SetWebSocketTimeout", "Removed Websocket due to timeout from :  "+meta.Conn.Connection.RemoteAddr().String())
				}
				log.Println("Removed Websocket due to timeout from :  " + meta.Conn.Connection.RemoteAddr().String())
				deleteWebSocket(meta.Conn)
			}
		}
		return true
	})

	//
	// var socketsToRemove []*WebSocketConnection
	//
	// for item := range WebSocketConnections.Iter() {
	// 	wsConn := item.Conn
	// 	wsConn.LastResponseTimeLock.RLock()
	// 	lastResponseTime := wsConn.LastResponseTime
	// 	wsConn.LastResponseTimeLock.RUnlock()
	//
	// 	if lastResponseTime.Add(time.Millisecond * time.Duration(timeout)).Before(time.Now()) {
	// 		socketsToRemove = append(socketsToRemove, wsConn)
	// 	}
	// }
	//
	// for i := 0; i < len(socketsToRemove); i++ {
	//
	// 	c := socketsToRemove[i]
	// 	if CustomLog != nil {
	// 		CustomLog("app->SetWebSocketTimeout", "Removed Websocket due to timeout from :  "+c.Connection.RemoteAddr().String())
	// 	}
	// 	log.Println("Removed Websocket due to timeout from :  " + c.Connection.RemoteAddr().String())
	// 	deleteWebSocket(c)
	//
	// }

	time.Sleep(time.Millisecond * time.Duration(timeout))
	SetWebSocketTimeout(timeout)
}

func deleteWebSocket(c *WebSocketConnection) {

	// return
	index := -1
	idToRemove := ""

	for item := range WebSocketConnections.Iter() {
		wsConn := item.Conn
		if wsConn.Id == c.Id {
			index = item.Index
			idToRemove = item.Conn.Id
		}
	}

	if index > -1 {
		go func() {
			defer func() {
				if recover := recover(); recover != nil {
					CustomLog("app->deleteWebSocket", "Panic Recovered at deleteWebSocket():  "+fmt.Sprintf("%+v", recover))
					return
				}
			}()

			if CustomLog != nil {
				CustomLog("app->deleteWebSocket", "Deleting Web Socket from client:  "+c.Connection.RemoteAddr().String())
			}

			WebSocketConnections.Lock()
			defer WebSocketConnections.Unlock()
			WebSocketConnections.Connections = removeWebSocket(WebSocketConnections.Connections, index)

			if store.OnChange != nil {
				go func() {
					defer func() {
						if recover := recover(); recover != nil {
							log.Println("Panic Recovered at store.OnChange():  ", recover)
							return
						}
					}()

					store.OnChange(store.WebSocketStoreKey, "", store.PathRemove, nil, nil)
				}()
			}

			if WebSocketRemovalCallback != nil {
				info, ok := GetWebSocketMeta(c.Id)
				if ok {
					go func(c *WebSocketConnection) {
						defer func() {
							if recover := recover(); recover != nil {
								log.Println("Panic Recovered at deleteWebSocket():  ", recover)
								return
							}
						}()
						WebSocketRemovalCallback(info)
					}(c)
				}

			}

			if idToRemove != "" {
				RemoveWebSocketMeta(idToRemove)
			}
		}()

	}
}

func GetWebSocketMeta(id string) (info WebSocketConnectionMeta, ok bool) {
	result, ok := webSocketConnectionsMeta.Load(id)
	if ok {
		info = result.(WebSocketConnectionMeta)
		return
	}
	return
}

func SetWebSocketMeta(id string, info WebSocketConnectionMeta) {
	webSocketConnectionsMeta.Store(id, info)
}

func RemoveWebSocketMeta(id string) {
	webSocketConnectionsMeta.Delete(id)
}

func GetAllWebSocketMeta() (items *sync.Map) {
	return &webSocketConnectionsMeta
}

func removeWebSocket(s []*WebSocketConnection, i int) []*WebSocketConnection {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func randomString(strlen int) string {
	randMath.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[randMath.Intn(len(chars))]
	}
	return string(result)
}
