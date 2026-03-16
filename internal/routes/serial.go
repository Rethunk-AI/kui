package routes

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/kui/kui/internal/libvirtconn"
	mw "github.com/kui/kui/internal/middleware"
)

const (
	websocketCloseSerialUnavailable = 1011
)

func (r *routerState) serialProxy() http.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}

		hostID := chi.URLParam(req, "host_id")
		libvirtUUID := chi.URLParam(req, "libvirt_uuid")
		if hostID == "" || libvirtUUID == "" {
			writeJSONError(w, http.StatusBadRequest, "host_id and libvirt_uuid required")
			return
		}

		meta, err := r.db.GetVMMetadata(req.Context(), hostID, libvirtUUID)
		if err != nil {
			r.logger.Error("get vm_metadata failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to get VM")
			return
		}
		if meta == nil || !meta.Claimed {
			writeJSONError(w, http.StatusNotFound, "VM not found")
			return
		}

		conn, err := r.getConnectorForHost(req.Context(), hostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer conn.Close()

		state, err := conn.GetState(req.Context(), libvirtUUID)
		if err != nil {
			r.logger.Error("get domain state failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			writeJSONError(w, http.StatusNotFound, "VM not found")
			return
		}
		if state != libvirtconn.DomainStateRunning && state != libvirtconn.DomainStatePaused {
			writeJSONError(w, http.StatusBadGateway, "VM not running")
			return
		}

		// Serial uses libvirt virDomainOpenConsole; for qemu+ssh, libvirt tunnels all API calls over SSH.
		serialStream, err := conn.OpenSerialConsole(req.Context(), libvirtUUID)
		if err != nil {
			r.logger.Error("open serial console failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			writeJSONError(w, http.StatusBadGateway, "serial console unavailable")
			return
		}
		defer serialStream.Close()

		wsConn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			r.logger.Error("WebSocket upgrade failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			return
		}
		defer wsConn.Close()

		r.proxySerial(req.Context(), wsConn, serialStream, hostID, libvirtUUID)
	}
}

func (r *routerState) proxySerial(ctx context.Context, ws *websocket.Conn, serial io.ReadWriteCloser, hostID, libvirtUUID string) {
	var wg sync.WaitGroup
	wg.Add(2)

	// WebSocket -> serial
	go func() {
		defer wg.Done()
		proxyWSToSerial(r.logger, ws, serial, hostID, libvirtUUID)
	}()

	// serial -> WebSocket
	go func() {
		defer wg.Done()
		proxySerialToWS(r.logger, ws, serial, hostID, libvirtUUID)
	}()

	wg.Wait()
}

func proxyWSToSerial(logger *slog.Logger, ws *websocket.Conn, serial io.WriteCloser, hostID, libvirtUUID string) {
	defer serial.Close()
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return
			}
			logger.Debug("WebSocket read error in serial proxy", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocketCloseSerialUnavailable, "proxy error"), time.Now().Add(time.Second))
			return
		}
		if _, err := serial.Write(data); err != nil {
			logger.Debug("serial write error", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocketCloseSerialUnavailable, "serial backend closed"), time.Now().Add(time.Second))
			return
		}
	}
}

func proxySerialToWS(logger *slog.Logger, ws *websocket.Conn, serial io.ReadCloser, hostID, libvirtUUID string) {
	defer serial.Close()
	buf := make([]byte, 32*1024)
	for {
		n, err := serial.Read(buf)
		if err != nil {
			if err != io.EOF {
				logger.Debug("serial read error", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			}
			_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocketCloseSerialUnavailable, "serial backend closed"), time.Now().Add(time.Second))
			return
		}
		if n == 0 {
			continue
		}
		if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
			logger.Debug("WebSocket write error in serial proxy", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			return
		}
	}
}
