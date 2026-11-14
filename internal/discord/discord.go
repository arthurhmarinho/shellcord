// Package discord implementa o cliente do protocolo IPC local do Discord
// (ADR-0001): cabeçalho de 8 bytes (opcode e tamanho, little-endian) seguido
// de JSON.
package discord

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	opHandshake = 0
	opFrame     = 1
	opClose     = 2
	opPing      = 3
	opPong      = 4

	maxFrame = 64 << 10
	// replyTimeout limita a espera pelo handshake e pelas respostas. Com o
	// socat do Flatpak no meio, um Discord travado pode aceitar a conexão e
	// nunca responder (ADR-0007).
	replyTimeout = 5 * time.Second
)

// Activity é a atividade exibida no card. Campos vazios são omitidos.
type Activity struct {
	Details    string      `json:"details,omitempty"`
	State      string      `json:"state,omitempty"`
	Timestamps *Timestamps `json:"timestamps,omitempty"`
	Assets     *Assets     `json:"assets,omitempty"`
}

// Timestamps controla o cronômetro do card. Start é em segundos Unix.
type Timestamps struct {
	Start int64 `json:"start,omitempty"`
}

// Assets são as imagens do card, cadastradas no Developer Portal.
type Assets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"` // círculo no canto da imagem grande
	SmallText  string `json:"small_text,omitempty"`
}

// DefaultDirs são os diretórios onde procurar discord-ipc-N: o padrão e o do
// Flatpak (ADR-0007).
func DefaultDirs() []string {
	rt := os.Getenv("XDG_RUNTIME_DIR")
	if rt == "" {
		rt = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	return []string{rt, filepath.Join(rt, "app", "com.discordapp.Discord")}
}

type frame struct {
	Cmd   string          `json:"cmd"`
	Evt   string          `json:"evt"`
	Nonce string          `json:"nonce"`
	Data  json.RawMessage `json:"data"`
}

// Client é uma conexão aberta e autenticada com o Discord.
type Client struct {
	conn    net.Conn
	wmu     sync.Mutex
	nonce   atomic.Int64
	replies chan frame
	done    chan struct{}
	once    sync.Once
	err     error
}

// Dial procura discord-ipc-0 a discord-ipc-9 em cada diretório e devolve a
// primeira conexão em que o handshake funciona. O arquivo existir não basta:
// só conta se conectar e o Discord responder READY.
func Dial(clientID string, dirs []string) (*Client, error) {
	var lastErr error
	for _, d := range dirs {
		for i := 0; i < 10; i++ {
			p := filepath.Join(d, "discord-ipc-"+strconv.Itoa(i))
			conn, err := net.DialTimeout("unix", p, time.Second)
			if err != nil {
				if lastErr == nil || !errors.Is(err, os.ErrNotExist) {
					lastErr = err
				}
				continue
			}
			c, err := handshake(conn, clientID)
			if err != nil {
				conn.Close()
				lastErr = fmt.Errorf("%s: %w", p, err)
				continue
			}
			return c, nil
		}
	}
	return nil, fmt.Errorf("Discord indisponível: %w", lastErr)
}

func handshake(conn net.Conn, clientID string) (*Client, error) {
	conn.SetDeadline(time.Now().Add(replyTimeout))
	body, _ := json.Marshal(map[string]any{"v": 1, "client_id": clientID})
	if err := writeFrame(conn, opHandshake, body); err != nil {
		return nil, err
	}
	op, resp, err := readFrame(conn)
	if err != nil {
		return nil, err
	}
	if op == opClose {
		return nil, fmt.Errorf("handshake recusado: %s", resp)
	}
	var f frame
	if op != opFrame || json.Unmarshal(resp, &f) != nil || f.Evt != "READY" {
		return nil, fmt.Errorf("resposta inesperada ao handshake (op=%d): %s", op, resp)
	}
	conn.SetDeadline(time.Time{})
	c := &Client{conn: conn, replies: make(chan frame, 8), done: make(chan struct{})}
	go c.readLoop()
	return c, nil
}

func (c *Client) readLoop() {
	for {
		op, body, err := readFrame(c.conn)
		if err != nil {
			c.shutdown(err)
			return
		}
		switch op {
		case opPing:
			c.write(opPong, body)
		case opClose:
			c.shutdown(fmt.Errorf("Discord fechou a conexão: %s", body))
			return
		case opFrame:
			var f frame
			if json.Unmarshal(body, &f) == nil && f.Nonce != "" {
				select {
				case c.replies <- f:
				default: // ninguém esperando; descarta
				}
			}
		}
	}
}

func (c *Client) shutdown(err error) {
	c.once.Do(func() {
		c.err = err
		close(c.done)
		c.conn.Close()
	})
}

func (c *Client) write(op uint32, body []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(replyTimeout))
	return writeFrame(c.conn, op, body)
}

// Done fecha quando a conexão cai. Err diz o motivo.
func (c *Client) Done() <-chan struct{} { return c.done }

// Err devolve o motivo do fechamento; só tem valor depois de Done fechar.
func (c *Client) Err() error { return c.err }

// Close fecha a conexão. O Discord limpa a presença ao perceber o fechamento.
func (c *Client) Close() { c.shutdown(errors.New("conexão fechada pelo shellcord")) }

// SetActivity publica a atividade e espera a confirmação. Com a == nil, limpa
// a presença.
func (c *Client) SetActivity(a *Activity) error {
	nonce := strconv.FormatInt(c.nonce.Add(1), 10)
	args := map[string]any{"pid": os.Getpid()}
	if a != nil {
		args["activity"] = a
	}
	body, err := json.Marshal(map[string]any{"cmd": "SET_ACTIVITY", "nonce": nonce, "args": args})
	if err != nil {
		return err
	}
	if err := c.write(opFrame, body); err != nil {
		c.shutdown(err)
		return err
	}
	timer := time.NewTimer(replyTimeout)
	defer timer.Stop()
	for {
		select {
		case f := <-c.replies:
			if f.Nonce != nonce {
				continue
			}
			if f.Evt == "ERROR" {
				return fmt.Errorf("SET_ACTIVITY recusado: %s", f.Data)
			}
			return nil
		case <-c.done:
			return c.err
		case <-timer.C:
			return errors.New("Discord não respondeu ao SET_ACTIVITY")
		}
	}
}

func writeFrame(w io.Writer, op uint32, body []byte) error {
	buf := make([]byte, 8+len(body))
	binary.LittleEndian.PutUint32(buf[0:4], op)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(body)))
	copy(buf[8:], body)
	_, err := w.Write(buf)
	return err
}

func readFrame(r io.Reader) (uint32, []byte, error) {
	var hdr [8]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, err
	}
	op := binary.LittleEndian.Uint32(hdr[0:4])
	n := binary.LittleEndian.Uint32(hdr[4:8])
	if n > maxFrame {
		return 0, nil, fmt.Errorf("frame de %d bytes passa do limite", n)
	}
	body := make([]byte, n)
	_, err := io.ReadFull(r, body)
	return op, body, err
}
