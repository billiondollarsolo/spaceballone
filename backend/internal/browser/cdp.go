package browser

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

// CDPClient is a simple Chrome DevTools Protocol client over WebSocket.
type CDPClient struct {
	conn      *websocket.Conn
	mu        sync.Mutex
	nextID    atomic.Int64
	callbacks sync.Map // id -> chan json.RawMessage
	eventCh   chan CDPEvent
	done      chan struct{}
}

// CDPEvent represents an incoming CDP event (method notification).
type CDPEvent struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// cdpResponse represents a CDP JSON-RPC response.
type cdpResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *cdpError       `json:"error,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

type cdpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewCDPClient connects to a CDP WebSocket endpoint and starts reading.
func NewCDPClient(wsURL string) (*CDPClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cdp: failed to connect: %w", err)
	}

	c := &CDPClient{
		conn:    conn,
		eventCh: make(chan CDPEvent, 64),
		done:    make(chan struct{}),
	}

	go c.readLoop()
	return c, nil
}

// Events returns a channel that receives CDP events (notifications).
func (c *CDPClient) Events() <-chan CDPEvent {
	return c.eventCh
}

// Send sends a CDP command and waits for a response.
func (c *CDPClient) Send(method string, params interface{}) (json.RawMessage, error) {
	id := c.nextID.Add(1)

	msg := map[string]interface{}{
		"id":     id,
		"method": method,
	}
	if params != nil {
		msg["params"] = params
	}

	ch := make(chan json.RawMessage, 1)
	c.callbacks.Store(id, ch)
	defer c.callbacks.Delete(id)

	c.mu.Lock()
	err := c.conn.WriteJSON(msg)
	c.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("cdp: failed to send: %w", err)
	}

	select {
	case result := <-ch:
		return result, nil
	case <-c.done:
		return nil, fmt.Errorf("cdp: connection closed")
	}
}

// Close closes the CDP WebSocket connection.
func (c *CDPClient) Close() error {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	return c.conn.Close()
}

func (c *CDPClient) readLoop() {
	defer func() {
		select {
		case <-c.done:
		default:
			close(c.done)
		}
		close(c.eventCh)
	}()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			select {
			case <-c.done:
			default:
				log.Printf("cdp: read error: %v", err)
			}
			return
		}

		var resp cdpResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			log.Printf("cdp: unmarshal error: %v", err)
			continue
		}

		// If it has an ID, it's a response to a command
		if resp.ID > 0 {
			if ch, ok := c.callbacks.Load(resp.ID); ok {
				if resp.Error != nil {
					// Send nil to indicate error (caller gets empty result)
					ch.(chan json.RawMessage) <- nil
				} else {
					ch.(chan json.RawMessage) <- resp.Result
				}
			}
			continue
		}

		// Otherwise it's an event
		if resp.Method != "" {
			select {
			case c.eventCh <- CDPEvent{Method: resp.Method, Params: resp.Params}:
			default:
				// Drop event if channel is full
			}
		}
	}
}

// PageNavigate sends Page.navigate.
func (c *CDPClient) PageNavigate(url string) error {
	_, err := c.Send("Page.navigate", map[string]interface{}{
		"url": url,
	})
	return err
}

// PageStartScreencast starts a page screencast.
func (c *CDPClient) PageStartScreencast(format string, quality int, maxWidth, maxHeight int) error {
	_, err := c.Send("Page.startScreencast", map[string]interface{}{
		"format":    format,
		"quality":   quality,
		"maxWidth":  maxWidth,
		"maxHeight": maxHeight,
	})
	return err
}

// PageScreencastFrameAck acknowledges a screencast frame.
func (c *CDPClient) PageScreencastFrameAck(sessionID int) error {
	_, err := c.Send("Page.screencastFrameAck", map[string]interface{}{
		"sessionId": sessionID,
	})
	return err
}

// InputDispatchMouseEvent dispatches a mouse event.
func (c *CDPClient) InputDispatchMouseEvent(eventType string, x, y float64, button string, clickCount int) error {
	params := map[string]interface{}{
		"type": eventType,
		"x":    x,
		"y":    y,
	}
	if button != "" {
		params["button"] = button
	}
	if clickCount > 0 {
		params["clickCount"] = clickCount
	}
	_, err := c.Send("Input.dispatchMouseEvent", params)
	return err
}

// InputDispatchKeyEvent dispatches a key event.
func (c *CDPClient) InputDispatchKeyEvent(eventType, key, code, text string) error {
	params := map[string]interface{}{
		"type": eventType,
	}
	if key != "" {
		params["key"] = key
	}
	if code != "" {
		params["code"] = code
	}
	if text != "" {
		params["text"] = text
	}
	_, err := c.Send("Input.dispatchKeyEvent", params)
	return err
}
