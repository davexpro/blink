package client

import (
	"compress/gzip"
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
	"strings"
	"sync/atomic"
	"time"

	"github.com/bytedance/gopkg/lang/fastrand"
	dx25519 "github.com/davexpro/crypto/ed25519"
	"github.com/denisbrodbeck/machineid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/davexpro/blink"
	"github.com/davexpro/blink/internal/consts"
	"github.com/davexpro/blink/internal/pkg/blog"
	"github.com/davexpro/blink/internal/util"
)

type BlinkClient struct {
	endpoint string
	deviceId string

	pubKey        ed25519.PublicKey  // client ed25519 pub key
	priKey        ed25519.PrivateKey // client ed25519 pri key
	srvKey        ed25519.PublicKey  // server ed25519 pub key
	sharedKey     []byte             // chacha20poly1305 shared key
	enableEncrypt bool               // enable chacha20poly1305

	wsConn      *websocket.Conn
	isConnected atomic.Bool

	exitCh   chan struct{}
	exitMark atomic.Bool
}

var (
	errInvalidConn = errors.New("invalid conn")
)

func NewBlinkClient(endpoint string, srvKey ed25519.PublicKey) *BlinkClient {
	// generate ed25519 key pairs
	pub, pri, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

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

	// generate shared key
	sharedKey, err := dx25519.SharedKeyByEd25519(pri, srvKey)

	return &BlinkClient{
		endpoint:  endpoint,
		deviceId:  did,
		pubKey:    pub,
		priKey:    pri,
		srvKey:    srvKey,
		sharedKey: sharedKey,
		exitCh:    make(chan struct{}),
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

// connectServer connect server's endpoint
func (h *BlinkClient) connectServer(ctx context.Context) error {
	wsCli := &websocket.Dialer{
		HandshakeTimeout:  time.Second * 10,
		EnableCompression: true,
	}
	headers := http.Header{
		"User-Agent":       {fmt.Sprintf("Blink Client/%s", blink.Version)},
		"Content-Encoding": {"gzip"},
	}
	pubKeyStr, _ := x509.MarshalPKIXPublicKey(h.pubKey)
	headers.Add(consts.HeaderClientKey, base64.StdEncoding.EncodeToString(pubKeyStr))
	headers.Add(consts.HeaderClientVersion, blink.Version)

	conn, httpResp, err := wsCli.DialContext(ctx, h.endpoint, headers)
	if err != nil {
		return err
	}

	fmt.Println(httpResp)

	h.wsConn = conn
	h.isConnected.Store(true)

	// send auth frame
	authBytes, err := util.Chacha20Encrypt(h.sharedKey, []byte(consts.BlinkIdentifier))
	if err != nil {
		blog.Errorf("crypto `Chacha20Encrypt` failed, detail: %s", err)
		return err
	}

	err = conn.WriteMessage(websocket.BinaryMessage, authBytes)
	if err != nil {
		blog.Errorf("conn `WriteMessage` failed, detail: %s", err)
		return err
	}

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

// heartbeat keep ws conn alive
func (h *BlinkClient) heartbeat() {
	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case <-h.exitCh:
			return
		case <-timer.C:
			if !h.isConnected.Load() || h.wsConn == nil {
				return
			}
			body := fmt.Sprintf("%s|%s|%s|%s|%d",
				blink.Name, blink.Version, runtime.GOARCH, runtime.GOOS, time.Now().UnixMilli())
			err := h.wsConn.WriteMessage(websocket.PingMessage, []byte(body))
			if err != nil {
				blog.Errorf("ws `WriteMessage` error: %s", err)
				if strings.Contains(err.Error(), "broken pipe") {
					h.disconnected()
				}
			}
		}
		timer.Reset(consts.HeartbeatDuration)
	}
}

// writeMsg json -> gzip -> chacha20-poly
func (h *BlinkClient) writeMsg(msgType int, body []byte) error {
	// 1. gzip the body
	buf := newBuffer()
	defer freeBuffer(buf)
	gw, _ := gzip.NewWriterLevel(buf, 5)
	if _, err := gw.Write(body); err != nil {
		blog.Errorf("gzip `Write` failed, detail: %s", err)
		return err
	}
	_ = gw.Close()

	/* copy to the result buffer */
	gzBody := make([]byte, buf.Len())
	copy(gzBody, buf.Bytes())

	// 2. encrypt th body
	encBody, err := util.Chacha20Encrypt(h.sharedKey, gzBody)
	if err != nil {
		blog.Errorf("util `Chacha20Encrypt` error: %w", err)
		return err
	}

	// 3. send msg
	if !h.isConnected.Load() || h.wsConn == nil {
		return errInvalidConn
	}

	err = h.wsConn.WriteMessage(msgType, encBody)
	if err != nil {
		blog.Errorf("ws `WriteMessage` error: %s", err)
		return err
	}

	blog.Infof("send msg, raw: %d gzip: %d encrypted: %d", len(body), len(gzBody), len(encBody))

	return nil
}

func (h *BlinkClient) disconnected() {
	h.isConnected.Store(false)
	h.wsConn = nil
}
