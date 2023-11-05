package server

import (
	"context"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
	hertzWs "github.com/hertz-contrib/websocket"

	"github.com/davexpro/blink/internal/pkg/blog"
)

type ClientConnHandler struct {
	ctx    context.Context
	conn   *hertzWs.Conn
	reqCtx *app.RequestContext
}

func NewClientConnHandler(ctx context.Context, c *app.RequestContext, conn *hertzWs.Conn) *ClientConnHandler {
	return &ClientConnHandler{
		ctx:    ctx,
		reqCtx: c,
		conn:   conn,
	}
}

func (h *ClientConnHandler) Serve() {
	c := h.reqCtx
	ctx := h.ctx
	conn := h.conn

	log.Println("x-real-ip", string(c.GetHeader("X-Real-IP")))
	log.Println("xff", string(c.GetHeader("X-Forwarded-For")))
	log.Println(conn.RemoteAddr())
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		switch mt {
		case hertzWs.PingMessage:
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
