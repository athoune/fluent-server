package message

import (
	"fmt"
)

func (s *FluentSession) doPing() error {
	fmt.Println("PING")
	return nil
}
