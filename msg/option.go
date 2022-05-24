package msg

type Option struct {
	Size       int
	Chunk      string
	Compressed string
	Stuff      map[string]interface{}
}
