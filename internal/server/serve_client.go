package server

import (
	"context"
	"crypto/ed25519"
	"log"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/app"
	hertzWs "github.com/hertz-contrib/websocket"

	"github.com/davexpro/blink/internal/consts"
	"github.com/davexpro/blink/internal/pkg/blog"
	"github.com/davexpro/blink/internal/util"
)

type ClientConnHandler struct {
	ctx    context.Context
	conn   *hertzWs.Conn
	reqCtx *app.RequestContext

	sharedKey  []byte
	isVerified atomic.Bool
}

func NewClientConnHandler(ctx context.Context, c *app.RequestContext, conn *hertzWs.Conn, key ed25519.PublicKey, sharedKey []byte) *ClientConnHandler {
	return &ClientConnHandler{
		ctx:       ctx,
		reqCtx:    c,
		conn:      conn,
		sharedKey: sharedKey,
	}
}

func (h *ClientConnHandler) Serve() {
	c := h.reqCtx
	ctx := h.ctx
	conn := h.conn
	defer func() {
		h.conn.Close()
	}()

	// TODO register conn

	log.Println("x-real-ip", string(c.GetHeader("X-Real-IP")))
	log.Println(consts.HeaderClientKey, string(c.GetHeader(consts.HeaderClientKey)))
	log.Println("xff", string(c.GetHeader("X-Forwarded-For")))
	log.Println(conn.RemoteAddr())
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			blog.CtxErrorf(ctx, "ws `ReadMessage` error: %s", err)
			return
		}

		if mt != hertzWs.BinaryMessage && mt != hertzWs.TextMessage {
			continue
		}

		// should verify the conn first
		if !h.isVerified.Load() {
			identifier, err := util.Chacha20Decrypt(h.sharedKey, message)
			if err != nil {
				blog.CtxErrorf(ctx, "util `Chacha20Decrypt` error: %s", err)
				return
			}
			if string(identifier) != consts.BlinkIdentifier {
				blog.CtxWarnf(ctx, "invalid identifier frame(%d): %s", len(identifier), identifier)
				return
			}
			h.isVerified.Store(true)
		}

		log.Println(mt, len(message), string(message))
		continue

		switch mt {
		case hertzWs.PingMessage:
			blog.CtxInfof(ctx, "msg type %d, body: %s", mt, string(message))
		case hertzWs.PongMessage:
		case hertzWs.BinaryMessage:
		default:
			blog.CtxWarnf(ctx, "unknown msg type %d, body: %s", mt, string(message))
		}

		log.Println(mt, len(message), string(message))
		// gzip 解包
		//gReader, err := gzip.NewReader(bytes.NewReader(message))
		//if err != nil {
		//	log.Println("gzip err", err)
		//	continue
		//}
		//
		//msg, _ := ioutil.ReadAll(gReader)
		//log.Printf("gzip len: %d", len(message))

		// protobuf 解包
		//frame := &pb_gen.Frame{}
		//_, err = fastpb.ReadMessage(msg, int8(fastpb.SkipTypeCheck), frame)
		//if err != nil {
		//	log.Println("err", err)
		//}
		//fmt.Println(frame)

		err = conn.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}
