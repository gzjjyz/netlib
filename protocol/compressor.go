package protocol

// Compressor defines a common compression interface.
type Compressor interface {
	Zip([]byte) ([]byte, error)
	Unzip([]byte) ([]byte, error)
}
