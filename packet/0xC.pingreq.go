package packet

import (
	"bytes"
	"io"
)

type PINGREQ struct {
	*FixedHeader `json:"FixedHeader,omitempty"`
}

func (pkt *PINGREQ) Kind() byte {
	return 0xC
}
func (pkt *PINGREQ) Pack(w io.Writer) error {
	return pkt.FixedHeader.Pack(w)
}
func (pkt *PINGREQ) Unpack(_ *bytes.Buffer) error {
	return nil
}
