package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// cdpClient: satu koneksi Chrome DevTools Protocol ke satu halaman (tanpa session id).
type cdpClient struct {
	ws      *wsConn
	mu      sync.Mutex
	nextID  int
	pending map[int]chan cdpResp
	closed  bool
	done    chan struct{}
}

type cdpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cdpResp struct {
	Result json.RawMessage `json:"result"`
	Error  *cdpError       `json:"error"`
}

func newCDP(ws *wsConn) *cdpClient {
	c := &cdpClient{ws: ws, pending: map[int]chan cdpResp{}, done: make(chan struct{})}
	go c.readLoop()
	return c
}

func (c *cdpClient) readLoop() {
	defer close(c.done)
	for {
		raw, err := c.ws.readMessage()
		if err != nil {
			c.fail()
			return
		}
		var m struct {
			ID *int `json:"id"`
			cdpResp
		}
		if json.Unmarshal(raw, &m) != nil || m.ID == nil {
			continue // event; tidak dipakai
		}
		c.mu.Lock()
		ch := c.pending[*m.ID]
		delete(c.pending, *m.ID)
		c.mu.Unlock()
		if ch != nil {
			ch <- m.cdpResp
		}
	}
}

func (c *cdpClient) fail() {
	c.mu.Lock()
	c.closed = true
	for id, ch := range c.pending {
		ch <- cdpResp{Error: &cdpError{Code: -1, Message: "koneksi browser terputus"}}
		delete(c.pending, id)
	}
	c.mu.Unlock()
}

func (c *cdpClient) alive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.closed
}

func (c *cdpClient) close() {
	c.ws.close()
	<-c.done
}

// call mengirim satu perintah dan menunggu jawabannya.
func (c *cdpClient) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, errors.New("koneksi browser terputus")
	}
	c.nextID++
	id := c.nextID
	ch := make(chan cdpResp, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	if params == nil {
		params = map[string]any{}
	}
	body, err := json.Marshal(map[string]any{"id": id, "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	if err := c.ws.writeText(body); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}
	select {
	case r := <-ch:
		if r.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, r.Error.Message)
		}
		return r.Result, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	}
}
