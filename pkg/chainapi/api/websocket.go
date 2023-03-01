package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/edwingeng/deque/v2"
	guuid "github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/logging"
)

const (
	maxChanBufferRegister   = 1024
	maxChanBufferUnregister = 1024
	maxOnChainTrxs          = 1024
)

var (
	wsLogger = logging.Logger("websocket")

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

type (
	TrxOnChainEvent struct {
		TrxID string `json:"trx_id"`
	}

	WebsocketManager struct {
		Lock        sync.Mutex
		Clients     map[string]*Client
		Register    chan *Client
		UnRegister  chan *Client
		OnChainTrxs *deque.Deque[*chain.OnChainTrxEvent]
	}

	Client struct {
		Id            string
		Socket        *websocket.Conn
		OnChainTrxs   chan *chain.OnChainTrxEvent
		HeartbeatTime int64
	}
)

func NewWebsocketManager() *WebsocketManager {
	return &WebsocketManager{
		Register:    make(chan *Client, maxChanBufferRegister),
		UnRegister:  make(chan *Client, maxChanBufferUnregister),
		Clients:     make(map[string]*Client),
		OnChainTrxs: deque.NewDeque[*chain.OnChainTrxEvent](),
	}
}

func (manager *WebsocketManager) RegisterClient(c *Client) {
	manager.Lock.Lock()
	defer manager.Lock.Unlock()

	manager.Clients[c.Id] = c
}

func (manager *WebsocketManager) UnRegisterClient(c *Client) {
	manager.Lock.Lock()
	defer manager.Lock.Unlock()

	delete(manager.Clients, c.Id)
}

func (manager *WebsocketManager) register() {
	for {
		select {
		case c := <-manager.Register:
			wsLogger.Debugf("client %s connected", c.Id)
			manager.RegisterClient(c)
		case c := <-manager.UnRegister:
			c.Socket.Close()

			manager.UnRegisterClient(c)
		}
	}
}

func (manager *WebsocketManager) checkAndSend() {
	for {
		for !manager.OnChainTrxs.IsEmpty() {
			event := manager.OnChainTrxs.PopBack()
			for _, c := range manager.Clients {
				wsLogger.Debugf("put event %+v to client: %s", event, c.Id)
				c.OnChainTrxs <- event
			}
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func (manager *WebsocketManager) Start() {
	// register/un-register
	go func() {
		defer func() {
			if r := recover(); r != nil {
				wsLogger.Debugf("r: %+v", r)
			}
		}()

		manager.register()
	}()

	// check and send
	go func() {
		defer func() {
			if r := recover(); r != nil {
				wsLogger.Debugf("r: %+v", r)
			}
		}()

		manager.checkAndSend()
	}()
}

func (c *Client) Read(manager *WebsocketManager) {
	defer func() {
		wsLogger.Debugf("close: %s, date: %s", c.Id, time.Now())
		manager.UnRegister <- c
	}()

	for {
		_type, msg, err := c.Socket.ReadMessage()
		if err != nil || _type == websocket.CloseMessage {
			wsLogger.Debugf("c.Socket.ReadMessage failed: %s, type: %s", err, _type)
			break
		}

		wsLogger.Debugf("got websocket msg: %s from: %s", msg, c.Id)
	}
}

func (c *Client) Write() error {
	defer func() {
		wsLogger.Debugf("client [%s] disconnect", c.Id)
		if err := c.Socket.Close(); err != nil {
			wsLogger.Debugf("client [%s] disconnect failed: %s", c.Id, err)
		}
	}()

	for {
		select {
		case event, ok := <-c.OnChainTrxs:
			if !ok {
				return c.Socket.WriteMessage(websocket.CloseMessage, []byte{})
			}

			if err := c.Socket.WriteJSON(event); err != nil {
				wsLogger.Debugf("client [%s] write event %+v failed: %s", c.Id, event, err)
				return err
			}
		}
	}
}

// websocket handler
func (manager *WebsocketManager) WsConnect(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	client := &Client{
		Id:            guuid.NewString(),
		Socket:        ws,
		OnChainTrxs:   make(chan *chain.OnChainTrxEvent, maxOnChainTrxs),
		HeartbeatTime: time.Now().Unix(),
	}

	manager.RegisterClient(client)
	go client.Read(manager)
	go client.Write()

	wsLogger.Debugf("new client: %+v", client)

	return nil
}
