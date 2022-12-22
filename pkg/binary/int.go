package binary

import (
	"encoding/binary"

	"golang.org/x/exp/constraints"
)

func From[T constraints.Integer](i T) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(i))
	return b
}

func ToUint64(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}
