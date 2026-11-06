package browserhost

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// ReadMessage reads one native-messaging frame (uint32 LE length + JSON body).
func ReadMessage(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	n := binary.LittleEndian.Uint32(lenBuf[:])
	if n == 0 {
		return nil, fmt.Errorf("empty message length")
	}
	if n > MaxMessageBytes {
		return nil, fmt.Errorf("message too large: %d bytes", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// WriteMessage writes one native-messaging frame.
func WriteMessage(w io.Writer, payload []byte) error {
	if len(payload) > MaxMessageBytes {
		return fmt.Errorf("response too large")
	}
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(payload)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// WriteJSON marshals v and writes it as a native-messaging frame.
func WriteJSON(w io.Writer, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return WriteMessage(w, raw)
}
