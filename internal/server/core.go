package server

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	dx25519 "github.com/davexpro/crypto/ed25519"
	hertzWs "github.com/hertz-contrib/websocket"

	"github.com/davexpro/blink/internal/consts"
	"github.com/davexpro/blink/internal/pkg/blog"
)

type BlinkServer struct {
	srvKey ed25519.PrivateKey
}

func NewBlinkServer(srvKey ed25519.PrivateKey) *BlinkServer {
	return &BlinkServer{
		srvKey: srvKey,
	}
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

	// for server side endpoints
	srv.GET("/ws", h.serveClientWS)

	// TODO for admin side endpoints
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
}

// serveClientWS serve clients' conn
func (h *BlinkServer) serveClientWS(ctx context.Context, c *app.RequestContext) {
	// 1. request params check
	cliVer, cliKeyRaw := c.GetHeader(consts.HeaderClientVersion), c.GetHeader(consts.HeaderClientKey)
	if len(cliVer) <= 0 || len(cliKeyRaw) <= 0 {
		c.AbortWithMsg(consts.HTTPAbortMessage, http.StatusNotFound)
		return
	}

	// 2. try to unmarshal ed25519 keys
	keyBytes, err := base64.StdEncoding.DecodeString(string(cliKeyRaw))
	if err != nil {
		c.AbortWithMsg(consts.HTTPAbortMessage, http.StatusNotFound)
		blog.CtxWarnf(ctx, "base64 `Decode` failed, detail: %s", err)
		return
	}

	cliKeyAny, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		c.AbortWithMsg(consts.HTTPAbortMessage, http.StatusNotFound)
		blog.CtxWarnf(ctx, "x509 `ParsePKIXPublicKey` failed, detail: %s", err)
		return
	}

	cliKey, ok := cliKeyAny.(ed25519.PublicKey)
	if !ok || len(cliKey) <= 0 {
		c.AbortWithMsg(consts.HTTPAbortMessage, http.StatusNotFound)
		blog.CtxWarnf(ctx, "given key is not ed25519 pub key")
		return
	}

	sharedKey, err := dx25519.SharedKeyByEd25519(h.srvKey, cliKey)
	if err != nil {
		c.AbortWithMsg(consts.HTTPAbortMessage, http.StatusNotFound)
		blog.CtxWarnf(ctx, "dx25519 `SharedKeyByEd25519` failed, detail: %s", err)
		return
	}

	// 3. upgrade conn and serve
	err = upgrader.Upgrade(c, func(conn *hertzWs.Conn) { NewClientConnHandler(ctx, c, conn, cliKey, sharedKey).Serve() })
	if err != nil {
		log.Print("upgrade:", err)
		c.AbortWithMsg("404 page not found", http.StatusNotFound)
		return
	}
}
