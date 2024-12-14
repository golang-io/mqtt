package packet

import (
	"bytes"
	"io"
)

type RESERVED struct {
	*FixedHeader
}

func (pkt *RESERVED) Kind() byte {
	return pkt.FixedHeader.Kind
}
func (pkt *RESERVED) Pack(io.Writer) error {
	return nil
}
func (pkt *RESERVED) Unpack(*bytes.Buffer) error {
	return nil
}
