package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bytedance/gopkg/lang/fastrand"
	"github.com/bytedance/sonic"
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
	errInvalidSign = errors.New("invalid sign")
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

	blog.Infof(util.MustMarshalString(httpResp.Header))

	h.wsConn = conn
	h.isConnected.Store(true)

	// 1. send identifier
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

	// 2. serve the connection
	go h.serve()
	go h.heartbeat()

	return nil
}

// serve .
func (h *BlinkClient) serve() {
	for {
		select {
		case <-h.exitCh:
			return
		default:
			if !h.isConnected.Load() || h.wsConn == nil {
				return
			}

			timeNow := util.GetCurrentTimeCache()
			ctx := context.Background()
			ctx = context.WithValue(ctx, consts.CtxKeyStartTime, timeNow.UnixMilli())
			msgType, msgData, err := h.wsConn.ReadMessage()
			if err != nil {
				blog.Errorf("conn read fail: %s", err)
				h.disconnected()
				return
			}

			// handle the `ping`
			if msgType == websocket.PingMessage {
				body := fmt.Sprintf("%s|%s|%s|%s|%d",
					blink.Name, blink.Version, runtime.GOARCH, runtime.GOOS, timeNow.UnixMilli())
				err = h.wsConn.WriteMessage(websocket.PingMessage, []byte(body))
				if err != nil {
					blog.Errorf("ws `WriteMessage` error: %s", err)
					if strings.Contains(err.Error(), "broken pipe") {
						h.disconnected()
					}
				}
				continue
			}

			// we only recognize binary message
			if msgType != websocket.BinaryMessage {
				continue
			}

			// unmarshal the cmd
			frame, err := h.unmarshalFrame(msgData)
			if err != nil {
				blog.CtxErrorf(ctx, "cli `unmarshalFrame` error: %s", err)
				continue
			}

			err = h.handle(frame)
			if err != nil {
				blog.CtxErrorf(ctx, "cli `handle` error: %s", err)
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
			// TODO heartbeat node stats
			body := fmt.Sprintf("%s|%s|%s|%s|%d",
				blink.Name, blink.Version, runtime.GOARCH, runtime.GOOS, util.GetCurrentTimeCache().UnixMilli())
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

// unmarshalFrame unmarshal the frame from server
func (h *BlinkClient) unmarshalFrame(body []byte) (*consts.Frame, error) {
	// 1. decrypt th body
	decrypted, err := util.Chacha20Encrypt(h.sharedKey, body)
	if err != nil {
		blog.Errorf("util `Chacha20Encrypt` error: %w", err)
		return nil, err
	}

	// 2. ungzip the body
	gzipReader, err := gzip.NewReader(bytes.NewReader(decrypted))
	if err != nil {
		blog.Errorf("gzip `NewReader` error: %w", err)
		return nil, err
	}

	body, err = io.ReadAll(gzipReader)
	if err != nil {
		blog.Errorf("io `ReadAll` error: %w", err)
		return nil, err
	}

	// 3. unmarshal to json
	frame := &consts.Frame{}
	err = sonic.Unmarshal(body, &frame)
	if err != nil {
		blog.Errorf("sonic `Unmarshal` error: %w", err)
		return nil, err
	}

	// 4. verify the sign
	sign := frame.Signature
	if len(sign) < 6 {
		return nil, errInvalidSign
	}
	frame.Signature = ""
	bs, _ := sonic.Marshal(frame)
	signBs, _ := base64.StdEncoding.DecodeString(sign)
	if !ed25519.Verify(h.srvKey, bs, signBs) {
		return nil, errInvalidSign
	}

	return frame, nil
}

// writeFrame write frame to server
func (h *BlinkClient) writeFrame(frame *consts.Frame) error {
	if frame == nil {
		return nil
	}

	// 1. fill the required props
	frame.DeviceID = h.deviceId
	frame.Timestamp = util.GetCurrentTimeCache().UnixMilli()

	// 2. marshal and send it
	bs, _ := sonic.Marshal(frame)
	return h.writeMsg(websocket.BinaryMessage, bs)
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

// disconnected set the connection is disconnected
func (h *BlinkClient) disconnected() {
	h.isConnected.Store(false)
	h.wsConn = nil
}
