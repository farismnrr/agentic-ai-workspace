package egressproxy

import (
	"io"
	"net"
	"time"
)

const copyBuffer = 32 << 10

// pipe copies src to dst until either side ends, refreshing an idle deadline on each read and write and never
// going past the grant's end.
func pipe(dst, srcConn net.Conn, src io.Reader, idle time.Duration, hard time.Time) {
	buf := make([]byte, copyBuffer)

	for {
		_ = srcConn.SetReadDeadline(earliest(hard, time.Now().Add(idle)))

		n, err := src.Read(buf)
		if n > 0 {
			_ = dst.SetWriteDeadline(earliest(hard, time.Now().Add(idle)))

			if _, werr := dst.Write(buf[:n]); werr != nil {
				return
			}
		}

		if err != nil {
			return
		}
	}
}

func earliest(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}

	return b
}
