package server

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	hertzWs "github.com/hertz-contrib/websocket"
)

type BlinkServer struct {
}

func NewBlinkServer() *BlinkServer {
	return &BlinkServer{}
}

func (h *BlinkServer) Run() error {
	opts := []config.Option{
		server.WithHostPorts(":18888"),
		server.WithExitWaitTime(3 * time.Second),
		{F: func(o *config.Options) { o.NoDefaultServerHeader = true }},
	}
	srv := server.Default(opts...)
	// https://github.com/cloudwego/hertz/issues/121
	srv.NoHijackConnPool = true // for websocket

	srv.GET("/feedback", feedback)
	srv.Spin()
	return nil
}

var upgrader = hertzWs.HertzUpgrader{
	CheckOrigin: func(ctx *app.RequestContext) bool {
		return true
	},
	Error: func(ctx *app.RequestContext, status int, reason error) {
		log.Printf("%d: %s", status, reason)
		//ctx.Response.Header.Set("Sec-Websocket-Version", "13")
		//ctx.AbortWithMsg(reason.Error(), status)
	},
} // use default options

func feedback(ctx context.Context, c *app.RequestContext) {
	err := upgrader.Upgrade(c, func(conn *hertzWs.Conn) {
		log.Println("x-real-ip", string(c.GetHeader("X-Real-IP")))
		log.Println("xff", string(c.GetHeader("X-Forwarded-For")))
		log.Println(conn.RemoteAddr())
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
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
	})
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
}
