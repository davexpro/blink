package client

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/bytedance/gopkg/lang/fastrand"
	"github.com/denisbrodbeck/machineid"
	"github.com/gorilla/websocket"

	"github.com/davexpro/blink"
	"github.com/davexpro/blink/internal/consts"
	"github.com/davexpro/blink/internal/pkg/blog"
)

type BlinkClient struct {
	endpoint string
	deviceId string

	pubKey        string // client ed25519 pub key
	priKey        string // client ed25519 pri key
	srvKey        string // server ed25519 pri key
	enableEncrypt bool   // enable chacha20poly1305

	wsConn      *websocket.Conn
	isConnected atomic.Bool

	exitCh   chan struct{}
	exitMark atomic.Bool
}

func NewBlinkClient(endpoint string) *BlinkClient {
	// generate ed25519 key pairs
	pub, pri, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	pubKey, _ := x509.MarshalPKIXPublicKey(pub)
	priKey, _ := x509.MarshalPKCS8PrivateKey(pri)

	// generate device id
	did, err := machineid.ProtectedID(blink.Name)
	if err != nil {
		blog.Errorf("did generated failed #1, err: %s", err)
		did, err = machineid.ID()
		if err != nil {
			blog.Errorf("did generated failed #2, err: %s", err)
			secBuffer := make([]byte, 16)
			fastrand.Read(secBuffer)
			did = hex.EncodeToString(secBuffer)
		}
	}

	return &BlinkClient{
		endpoint: endpoint,
		deviceId: did,
		pubKey:   base64.StdEncoding.EncodeToString(pubKey),
		priKey:   base64.StdEncoding.EncodeToString(priKey),
		exitCh:   make(chan struct{}),
	}
}

func (h *BlinkClient) Run() error {
	var retryCnt int32
	for {
		select {
		case <-h.exitCh:
			return nil
		default:

		}

		// 判断是否连接上，连接上则 continue 然后定期检查
		if isConn := h.isConnected.Load(); isConn {
			time.Sleep(time.Second * 15)
			continue
		}

		// 连接服务器
		ctx := context.Background()
		err := h.connectServer(ctx)
		if err == nil {
			time.Sleep(time.Second * 15)
			continue
		}
		blog.Errorf("conn server failed, endpoint: %s err: %s", h.endpoint, err)
		
		// 连接重试的 backoff 策略
		retryCnt += 1
		if retryCnt <= 3 {
			time.Sleep(time.Second)
		} else if retryCnt <= 7 {
			time.Sleep(time.Second * 5)
		} else if retryCnt <= 11 {
			time.Sleep(time.Second * 11)
		} else {
			time.Sleep(time.Second * 23)
		}
	}
}

func (h *BlinkClient) Shutdown() {
	if h.exitMark.CompareAndSwap(false, true) {
		close(h.exitCh)
	}
	time.Sleep(time.Second * 3)
	os.Exit(0)
}

func (h *BlinkClient) connectServer(ctx context.Context) error {
	wsCli := &websocket.Dialer{
		HandshakeTimeout:  time.Second * 10,
		EnableCompression: true,
	}
	headers := http.Header{
		"User-Agent":       {fmt.Sprintf("Blink Client/%s", blink.Version)},
		"Content-Encoding": {"gzip"},
	}
	headers.Add(consts.HeaderClientKey, h.pubKey)
	headers.Add(consts.HeaderClientVersion, blink.Version)

	conn, httpResp, err := wsCli.DialContext(ctx, h.endpoint, headers)
	if err != nil {
		return err
	}

	fmt.Println(httpResp)

	h.wsConn = conn
	h.isConnected.Store(true)
	go h.serve()
	go h.heartbeat()

	return nil
}

func (h *BlinkClient) serve() {
	for {
		select {
		case <-h.exitCh:
			return
		default:
			if !h.isConnected.Load() || h.wsConn == nil {
				return
			}

			ctx := context.Background()
			msgType, msgData, err := h.wsConn.ReadMessage()
			if nil != err {
				blog.Errorf("conn read fail: %s", err)
				h.isConnected.Store(false)
				return
			}

			switch msgType {
			case websocket.BinaryMessage:
				fmt.Println(ctx, msgData)
				//t.handleFrame(ctx, msgData)
			case websocket.PingMessage:
				h.wsConn.WriteMessage(websocket.PongMessage, []byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
			default:
			}
		}
	}
}

func (h *BlinkClient) heartbeat() {
	for {
		select {
		case <-h.exitCh:
			return
		default:
			if h.isConnected.Load() || h.wsConn == nil {
				return
			}

			err := h.wsConn.WriteMessage(websocket.TextMessage, []byte(runtime.GOARCH+"|"+runtime.GOOS))
			if err != nil {
				blog.Errorf("ws write error: %w", err)
			}
			time.Sleep(time.Second * 15)
		}
	}
}
