// Package discordtest é um Discord falso para testes: escuta em
// <dir>/discord-ipc-0, responde ao handshake e confirma cada SET_ACTIVITY.
package discordtest

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Server é o Discord falso.
type Server struct {
	Dir      string
	ClientID string

	ln    net.Listener
	acts  chan json.RawMessage
	pings chan json.RawMessage
	mu    sync.Mutex
	conns []net.Conn
}

// New sobe o servidor num diretório temporário e o derruba no fim do teste.
func New(t testing.TB, clientID string) *Server {
	t.Helper()
	dir := t.TempDir()
	ln, err := net.Listen("unix", filepath.Join(dir, "discord-ipc-0"))
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{Dir: dir, ClientID: clientID, ln: ln,
		acts: make(chan json.RawMessage, 32), pings: make(chan json.RawMessage, 8)}
	go s.serve()
	t.Cleanup(func() {
		ln.Close()
		s.Disconnect()
	})
	return s
}

// NextActivity espera o próximo SET_ACTIVITY e devolve args.activity, que é
// "null" quando a presença é limpa.
func (s *Server) NextActivity(t testing.TB, timeout time.Duration) string {
	t.Helper()
	select {
	case a := <-s.acts:
		return string(a)
	case <-time.After(timeout):
		t.Fatal("nenhum SET_ACTIVITY recebido")
		return ""
	}
}

// NoActivity falha se chegar algum SET_ACTIVITY dentro do prazo.
func (s *Server) NoActivity(t testing.TB, wait time.Duration) {
	t.Helper()
	select {
	case a := <-s.acts:
		t.Fatalf("SET_ACTIVITY inesperado: %s", a)
	case <-time.After(wait):
	}
}

// Ping manda um PING para todas as conexões e espera o PONG com o mesmo corpo.
func (s *Server) Ping(t testing.TB, body string) {
	t.Helper()
	s.mu.Lock()
	for _, c := range s.conns {
		write(c, 3, []byte(body))
	}
	s.mu.Unlock()
	select {
	case got := <-s.pings:
		if string(got) != body {
			t.Fatalf("PONG com corpo %s, quero %s", got, body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("nenhum PONG recebido")
	}
}

// Disconnect manda CLOSE e fecha todas as conexões, como o Discord ao fechar.
func (s *Server) Disconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.conns {
		write(c, 2, []byte(`{"code":1000,"message":"tchau"}`))
		c.Close()
	}
	s.conns = nil
}

func (s *Server) serve() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(c)
	}
}

func (s *Server) handle(c net.Conn) {
	op, body, err := read(c)
	if err != nil || op != 0 {
		c.Close()
		return
	}
	var hs struct {
		ClientID string `json:"client_id"`
	}
	json.Unmarshal(body, &hs)
	if hs.ClientID != s.ClientID {
		write(c, 2, []byte(`{"code":4000,"message":"Invalid Client ID"}`))
		c.Close()
		return
	}
	s.mu.Lock()
	s.conns = append(s.conns, c)
	s.mu.Unlock()
	write(c, 1, []byte(`{"cmd":"DISPATCH","evt":"READY","data":{"v":1},"nonce":null}`))
	for {
		op, body, err := read(c)
		if err != nil {
			return
		}
		if op == 4 {
			s.pings <- body
			continue
		}
		var req struct {
			Cmd   string `json:"cmd"`
			Nonce string `json:"nonce"`
			Args  struct {
				Activity json.RawMessage `json:"activity"`
			} `json:"args"`
		}
		json.Unmarshal(body, &req)
		act := req.Args.Activity
		if act == nil {
			act = json.RawMessage("null")
		}
		resp, _ := json.Marshal(map[string]any{"cmd": req.Cmd, "nonce": req.Nonce, "evt": nil, "data": act})
		s.acts <- act
		write(c, 1, resp)
	}
}

func write(w io.Writer, op uint32, body []byte) {
	buf := make([]byte, 8+len(body))
	binary.LittleEndian.PutUint32(buf[0:4], op)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(body)))
	copy(buf[8:], body)
	w.Write(buf)
}

func read(r io.Reader) (uint32, []byte, error) {
	var hdr [8]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, err
	}
	body := make([]byte, binary.LittleEndian.Uint32(hdr[4:8]))
	_, err := io.ReadFull(r, body)
	return binary.LittleEndian.Uint32(hdr[0:4]), body, err
}
