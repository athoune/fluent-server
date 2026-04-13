package message

import (
	"fmt"

	"github.com/athoune/fluent-server/wire"
)

func Ack(wire *wire.Wire, chunk string) error {
	err := wire.Encoder.EncodeMapLen(1)
	if err != nil {
		return fmt.Errorf("failed to encode ack map length: %w", err)
	}
	err = wire.Encoder.EncodeString("ack")
	if err != nil {
		return fmt.Errorf("failed to encode ack key: %w", err)
	}
	err = wire.Encoder.EncodeString(chunk)
	if err != nil {
		return fmt.Errorf("failed to encode ack chunk: %w", err)
	}
	wire.Debug("< ACK")
	return wire.Flush()
}
