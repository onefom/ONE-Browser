package backend

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// Juggler uses NUL-delimited JSON over two private inherited handles. No TCP
// debugging listener or Python/Node installation is needed for manual browsing.
type camoufoxPipe struct {
	read, write           *os.File
	childRead, childWrite *os.File
	writeMu               sync.Mutex
	mu                    sync.Mutex
	pending               map[uint64]chan camoufoxReply
	seq                   atomic.Uint64
	done                  chan struct{}
	once                  sync.Once
}
type camoufoxReply struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func newCamoufoxPipe(cmd *exec.Cmd) (*camoufoxPipe, error) {
	cr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	pr, cw, err := os.Pipe()
	if err != nil {
		cr.Close()
		pw.Close()
		return nil, err
	}
	p := &camoufoxPipe{read: pr, write: pw, childRead: cr, childWrite: cw, pending: map[uint64]chan camoufoxReply{}, done: make(chan struct{})}
	if err = configureCamoufoxPipe(cmd, cr, cw); err != nil {
		p.close()
		p.closeChild()
		return nil, err
	}
	return p, nil
}
func (p *camoufoxPipe) closeChild() { p.childRead.Close(); p.childWrite.Close() }
func (p *camoufoxPipe) close()      { p.once.Do(func() { close(p.done); p.read.Close(); p.write.Close() }) }
func (p *camoufoxPipe) readLoop() {
	defer p.close()
	reader := bufio.NewReader(p.read)
	for {
		// Bound frames to protect the manager from unbounded remote output.
		var frame []byte
		for {
			part, err := reader.ReadSlice(0)
			frame = append(frame, part...)
			if len(frame) > 8*1024*1024 {
				return
			}
			if err == bufio.ErrBufferFull {
				continue
			}
			if err != nil {
				return
			}
			break
		}
		var reply camoufoxReply
		if json.Unmarshal(frame[:len(frame)-1], &reply) != nil || reply.ID == 0 {
			continue
		}
		p.mu.Lock()
		ch := p.pending[reply.ID]
		p.mu.Unlock()
		if ch != nil {
			select {
			case ch <- reply:
			default:
			}
		}
	}
}
func (p *camoufoxPipe) send(id uint64, method string, params any) error {
	message, err := json.Marshal(map[string]any{"id": id, "method": method, "params": params})
	if err != nil {
		return err
	}
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	select {
	case <-p.done:
		return io.EOF
	default:
	}
	_, err = p.write.Write(append(message, 0))
	return err
}
func (p *camoufoxPipe) call(method string, params any, timeout time.Duration) (json.RawMessage, error) {
	id := p.seq.Add(1)
	ch := make(chan camoufoxReply, 1)
	p.mu.Lock()
	p.pending[id] = ch
	p.mu.Unlock()
	defer func() { p.mu.Lock(); delete(p.pending, id); p.mu.Unlock() }()
	if err := p.send(id, method, params); err != nil {
		return nil, err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case r := <-ch:
		if r.Error != nil {
			return nil, fmt.Errorf("%s", r.Error.Message)
		}
		return r.Result, nil
	case <-p.done:
		return nil, io.EOF
	case <-timer.C:
		return nil, fmt.Errorf("Firefox 控制请求超时：%s", method)
	}
}
