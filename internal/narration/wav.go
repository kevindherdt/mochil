package narration

import (
	"bytes"
	"encoding/binary"
)

// WAV wraps PCM audio in a playable WAV container.
func (a Audio) WAV() []byte {
	var out bytes.Buffer
	put := func(value any) { _ = binary.Write(&out, binary.LittleEndian, value) }
	size, rate := uint32(len(a.PCM)), uint32(a.SampleRate)
	out.WriteString("RIFF")
	put(36 + size)
	out.WriteString("WAVEfmt ")
	put(uint32(16))
	put(uint16(1)) // PCM
	put(uint16(1)) // mono
	put(rate)
	put(rate * 2)
	put(uint16(2))
	put(uint16(16))
	out.WriteString("data")
	put(size)
	out.Write(a.PCM)
	return out.Bytes()
}
