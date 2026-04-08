package api

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	sshmanager "github.com/spaceballone/backend/internal/ssh"
	"gorm.io/gorm"
)

type ProxyHandler struct {
	DB  *gorm.DB
	SSH *sshmanager.Manager
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineId")
	portStr := chi.URLParam(r, "port")

	_, client, ok := requireConnectedMachine(h.DB, h.SSH, machineID, w)
	if !ok {
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		writeError(w, http.StatusBadRequest, "invalid port")
		return
	}

	remPath := chi.URLParam(r, "*")
	if remPath == "" {
		remPath = "/"
	}
	if !strings.HasPrefix(remPath, "/") {
		remPath = "/" + remPath
	}

	remoteAddr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to connect to "+remoteAddr+": "+err.Error())
		return
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(60 * time.Second))

	targetURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, remPath)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create proxy request")
		return
	}

	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}
	proxyReq.Host = remoteAddr
	proxyReq.Header.Del("Cookie")
	proxyReq.Header.Del("Referer")
	proxyReq.Header.Del("Origin")
	proxyReq.Header.Del("Sec-Fetch-Mode")
	proxyReq.Header.Del("Sec-Fetch-Dest")
	proxyReq.Header.Del("Sec-Fetch-Site")

	if err := proxyReq.Write(conn); err != nil {
		writeError(w, http.StatusBadGateway, "failed to send request to remote: "+err.Error())
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), proxyReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to read response from remote: "+err.Error())
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.Header().Del("X-Frame-Options")
	w.Header().Del("Content-Security-Policy")

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
