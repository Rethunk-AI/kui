package routes

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"libvirt.org/go/libvirtxml"

	"github.com/kui/kui/internal/config"
	"github.com/kui/kui/internal/libvirtconn"
	mw "github.com/kui/kui/internal/middleware"
	"github.com/kui/kui/internal/sshtunnel"
)

const (
	websocketCloseVNCUnavailable = 1011
)

// vncPortFromDomainXML parses domain XML and returns the VNC port if the domain has
// VNC graphics with a usable port. Returns 0, err if no VNC or port unavailable.
func vncPortFromDomainXML(domainXML string) (int, error) {
	var dom libvirtxml.Domain
	if err := dom.Unmarshal(domainXML); err != nil {
		return 0, err
	}
	if dom.Devices == nil {
		return 0, nil
	}
	for _, g := range dom.Devices.Graphics {
		if g.VNC == nil {
			continue
		}
		if g.VNC.Port > 0 {
			return g.VNC.Port, nil
		}
		if g.VNC.Socket != "" {
			// Unix socket: not supported for TCP proxy in MVP
			return 0, nil
		}
		if strings.EqualFold(g.VNC.AutoPort, "yes") {
			// Autoport without resolved port: live XML should have port when running
			return 0, nil
		}
	}
	return 0, nil
}

// isLocalLibvirtURI returns true if the URI indicates a local libvirt connection.
func isLocalLibvirtURI(uri string) bool {
	u := strings.TrimSpace(uri)
	return strings.HasPrefix(u, "qemu:///") || strings.HasPrefix(u, "qemu+unix:")
}

func (r *routerState) vncProxy() http.HandlerFunc {
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

		domainXML, err := conn.GetDomainXML(req.Context(), libvirtUUID)
		if err != nil {
			r.logger.Error("get domain XML failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to get domain config")
			return
		}

		port, err := vncPortFromDomainXML(domainXML)
		if err != nil || port <= 0 {
			r.logger.Debug("no VNC port in domain", "host_id", hostID, "libvirt_uuid", libvirtUUID)
			writeJSONError(w, http.StatusBadGateway, "VNC not available")
			return
		}

		var host *config.Host
		for i := range r.config.Hosts {
			if r.config.Hosts[i].ID == hostID {
				host = &r.config.Hosts[i]
				break
			}
		}
		if host == nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}

		var vncConn net.Conn
		if isLocalLibvirtURI(host.URI) {
			vncAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
			vncConn, err = net.Dial("tcp", vncAddr)
			if err != nil {
				r.logger.Error("VNC backend unreachable", "host_id", hostID, "libvirt_uuid", libvirtUUID, "addr", vncAddr, "error", err)
				writeJSONError(w, http.StatusBadGateway, "VNC backend unreachable")
				return
			}
		} else {
			sshCfg, err := sshtunnel.ParseQemuSSH(host.URI)
			if err != nil {
				r.logger.Error("VNC proxy: invalid qemu+ssh uri", "host_id", hostID, "error", err)
				writeJSONError(w, http.StatusBadGateway, "remote host VNC: invalid SSH config")
				return
			}
			if sshCfg == nil {
				r.logger.Debug("VNC proxy: unsupported remote URI scheme", "host_id", hostID, "uri", host.URI)
				writeJSONError(w, http.StatusBadGateway, "remote host VNC not supported")
				return
			}
			if host.Keyfile != nil && *host.Keyfile != "" {
				sshCfg.Keyfile = *host.Keyfile
			}
			vncAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
			vncConn, err = sshtunnel.DialRemote(req.Context(), sshCfg, "tcp", vncAddr)
			if err != nil {
				r.logger.Error("VNC proxy: SSH tunnel failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
				writeJSONError(w, http.StatusBadGateway, "VNC backend unreachable")
				return
			}
		}
		defer vncConn.Close()

		wsConn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			r.logger.Error("WebSocket upgrade failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			return
		}
		defer wsConn.Close()

		r.proxyVNC(req.Context(), wsConn, vncConn, hostID, libvirtUUID)
	}
}

func (r *routerState) proxyVNC(ctx context.Context, ws *websocket.Conn, vnc net.Conn, hostID, libvirtUUID string) {
	var wg sync.WaitGroup
	wg.Add(2)

	// WebSocket -> VNC
	go func() {
		defer wg.Done()
		proxyWSToVNC(r.logger, ws, vnc, hostID, libvirtUUID)
	}()

	// VNC -> WebSocket
	go func() {
		defer wg.Done()
		proxyVNCToWS(r.logger, ws, vnc, hostID, libvirtUUID)
	}()

	wg.Wait()
}

func proxyWSToVNC(logger *slog.Logger, ws *websocket.Conn, vnc net.Conn, hostID, libvirtUUID string) {
	defer vnc.Close()
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return
			}
			logger.Debug("WebSocket read error in VNC proxy", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocketCloseVNCUnavailable, "proxy error"), time.Now().Add(time.Second))
			return
		}
		if _, err := vnc.Write(data); err != nil {
			logger.Debug("VNC write error", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocketCloseVNCUnavailable, "VNC backend closed"), time.Now().Add(time.Second))
			return
		}
	}
}

func proxyVNCToWS(logger *slog.Logger, ws *websocket.Conn, vnc net.Conn, hostID, libvirtUUID string) {
	defer vnc.Close()
	buf := make([]byte, 32*1024)
	for {
		n, err := vnc.Read(buf)
		if err != nil {
			if err != io.EOF {
				logger.Debug("VNC read error", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			}
			_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocketCloseVNCUnavailable, "VNC backend closed"), time.Now().Add(time.Second))
			return
		}
		if n == 0 {
			continue
		}
		if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
			logger.Debug("WebSocket write error in VNC proxy", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			return
		}
	}
}
