package packet

import (
	"bytes"
	"io"
)

type PINGRESP struct {
	*FixedHeader `json:"FixedHeader,omitempty"`
}

func (pkt *PINGRESP) Kind() byte {
	return 0xD
}
func (pkt *PINGRESP) Pack(w io.Writer) error {
	return pkt.FixedHeader.Pack(w)
}
func (pkt *PINGRESP) Unpack(_ *bytes.Buffer) error {
	return nil
}
