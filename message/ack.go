package message

import "github.com/athoune/fluent-server/wire"

func Ack(wire *wire.Wire, chunk string) error {
	err := wire.Encoder.EncodeMapLen(1)
	if err != nil {
		return err
	}
	err = wire.Encoder.EncodeString("ack")
	if err != nil {
		return err
	}
	err = wire.Encoder.EncodeString(chunk)
	if err != nil {
		return err
	}
	wire.Debug("< ACK")
	return wire.Flush()
}
