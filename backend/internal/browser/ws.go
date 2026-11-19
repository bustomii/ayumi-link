package browser

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// Klien WebSocket minimal (RFC 6455) hanya untuk bicara ke DevTools Chromium di loopback.
// Sengaja ditulis sendiri supaya tidak menambah dependensi.

const maxWSMessage = 32 << 20 // screenshot base64 bisa beberapa MB

type wsConn struct {
	c   net.Conn
	br  *bufio.Reader
	wmu sync.Mutex
}

func wsDial(hostport, path string) (*wsConn, error) {
	c, err := net.DialTimeout("tcp", hostport, 5*time.Second)
	if err != nil {
		return nil, err
	}
	keyRaw := make([]byte, 16)
	if _, err := rand.Read(keyRaw); err != nil {
		c.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyRaw)
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", path, hostport, key)
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.Write([]byte(req)); err != nil {
		c.Close()
		return nil, err
	}
	br := bufio.NewReaderSize(c, 64<<10)
	res, err := http.ReadResponse(br, nil)
	if err != nil {
		c.Close()
		return nil, err
	}
	if res.StatusCode != http.StatusSwitchingProtocols {
		c.Close()
		return nil, fmt.Errorf("websocket ditolak: %s", res.Status)
	}
	_ = c.SetDeadline(time.Time{})
	return &wsConn{c: c, br: br}, nil
}

func (w *wsConn) close() { _ = w.c.Close() }

func (w *wsConn) writeFrame(op byte, payload []byte) error {
	w.wmu.Lock()
	defer w.wmu.Unlock()
	hdr := []byte{0x80 | op}
	n := len(payload)
	switch {
	case n < 126:
		hdr = append(hdr, 0x80|byte(n))
	case n <= 0xFFFF:
		hdr = append(hdr, 0x80|126, byte(n>>8), byte(n))
	default:
		hdr = append(hdr, 0x80|127)
		var l [8]byte
		binary.BigEndian.PutUint64(l[:], uint64(n))
		hdr = append(hdr, l[:]...)
	}
	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return err
	}
	hdr = append(hdr, mask[:]...)
	masked := make([]byte, n)
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}
	_ = w.c.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if _, err := w.c.Write(hdr); err != nil {
		return err
	}
	_, err := w.c.Write(masked)
	return err
}

func (w *wsConn) writeText(b []byte) error { return w.writeFrame(0x1, b) }

// readMessage membaca satu pesan utuh (menyambung fragmen, membalas ping).
func (w *wsConn) readMessage() ([]byte, error) {
	var msg []byte
	for {
		var h [2]byte
		if _, err := io.ReadFull(w.br, h[:]); err != nil {
			return nil, err
		}
		fin := h[0]&0x80 != 0
		op := h[0] & 0x0F
		masked := h[1]&0x80 != 0
		n := uint64(h[1] & 0x7F)
		switch n {
		case 126:
			var l [2]byte
			if _, err := io.ReadFull(w.br, l[:]); err != nil {
				return nil, err
			}
			n = uint64(binary.BigEndian.Uint16(l[:]))
		case 127:
			var l [8]byte
			if _, err := io.ReadFull(w.br, l[:]); err != nil {
				return nil, err
			}
			n = binary.BigEndian.Uint64(l[:])
		}
		if n > maxWSMessage || uint64(len(msg))+n > maxWSMessage {
			return nil, errors.New("pesan websocket terlalu besar")
		}
		var mask [4]byte
		if masked {
			if _, err := io.ReadFull(w.br, mask[:]); err != nil {
				return nil, err
			}
		}
		payload := make([]byte, n)
		if _, err := io.ReadFull(w.br, payload); err != nil {
			return nil, err
		}
		if masked {
			for i := range payload {
				payload[i] ^= mask[i%4]
			}
		}
		switch op {
		case 0x8:
			return nil, io.EOF
		case 0x9:
			_ = w.writeFrame(0xA, payload)
			continue
		case 0xA:
			continue
		}
		msg = append(msg, payload...)
		if fin {
			return msg, nil
		}
	}
}
