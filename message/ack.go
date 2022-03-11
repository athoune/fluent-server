package message

func (s *FluentSession) Ack(chunk string) error {
	return _map(s.encoder, "ack", chunk)
}
