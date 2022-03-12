package message

func (s *FluentSession) Ack(chunk string) error {
	err := s.encoder.EncodeMapLen(1)
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString("ack")
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString(chunk)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	return s.Flush()
}
