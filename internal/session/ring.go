package session

import "sync"

// Ring is a fixed-size circular byte buffer for PTY output replay.
type Ring struct {
	mu   sync.Mutex
	buf  []byte
	size int
	// start is index of oldest byte; length is how many bytes are valid.
	start  int
	length int
}

func NewRing(size int) *Ring {
	if size <= 0 {
		size = 512 * 1024
	}
	return &Ring{
		buf:  make([]byte, size),
		size: size,
	}
}

func (r *Ring) Write(p []byte) {
	if len(p) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// If payload alone fills or overflows the ring, keep only the tail.
	if len(p) >= r.size {
		copy(r.buf, p[len(p)-r.size:])
		r.start = 0
		r.length = r.size
		return
	}

	// Drop oldest bytes if needed.
	overflow := r.length + len(p) - r.size
	if overflow > 0 {
		r.start = (r.start + overflow) % r.size
		r.length -= overflow
	}

	// Append p at the logical end.
	end := (r.start + r.length) % r.size
	n := copy(r.buf[end:], p)
	if n < len(p) {
		copy(r.buf, p[n:])
	}
	r.length += len(p)
}

// Bytes returns a copy of the buffer in chronological order.
func (r *Ring) Bytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]byte, r.length)
	if r.length == 0 {
		return out
	}
	n := copy(out, r.buf[r.start:])
	if n < r.length {
		copy(out[n:], r.buf[:r.length-n])
	}
	return out
}

func (r *Ring) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.length
}
