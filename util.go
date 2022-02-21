package main

import (
	"errors"
	"fmt"
	"strings"
)

func checkHEX(b string, length int) bool {
	if len(b) != 2*length {
		return false
	}
	for i := 0; i < 2*length; i++ {
		if ((b[i] >= '0') && (b[i] <= '9')) || ((b[i] >= 'A') && (b[i] <= 'F')) {
		} else {

			return false
		}
	}
	return true
}
func checkHEX32(b string) error {
	if checkHEX(b, 32) {
		return nil
	}
	return fmt.Errorf("not a 32byte hex identifier")
}
func hex2byte32(hex []byte) (out [32]byte) {
	for i := range out {
		out[i] = (x2b(hex[i<<1]) << 4) | x2b(hex[i<<1|1])
	}
	return out
}
func hex2byte(hex []byte) (out []byte) {
	out = make([]byte, len(hex)/2)
	for i := range out {
		out[i] = (x2b(hex[i<<1]) << 4) | x2b(hex[i<<1|1])
	}
	return out
}
func x2b(hex byte) (lo byte) {
	return (hex & 15) + 9*(hex>>6)
}
func parse_hex(hex string) (raw [32]byte, err error) {
	if len(hex) < 64 {
		err = errors.New("hex too short")
		return raw, err
	}
	if len(hex) > 64 {
		err = errors.New("hex too long")
		return raw, err
	}

	hex = strings.ToUpper(hex)

	if err = checkHEX32(hex); err != nil {
		return raw, err
	}

	raw = hex2byte32([]byte(hex))
	return raw, nil
}

func stringify_hex(hex [32]byte) (str string) {
	return fmt.Sprintf("%X", hex)
}

func uint64_to_bytes(in uint64) (out [8]byte) {
	out[7] = byte((in >> 0) % 256)
	out[6] = byte((in >> 8) % 256)
	out[5] = byte((in >> 16) % 256)
	out[4] = byte((in >> 24) % 256)
	out[3] = byte((in >> 32) % 256)
	out[2] = byte((in >> 40) % 256)
	out[1] = byte((in >> 48) % 256)
	out[0] = byte((in >> 56) % 256)
	return out
}
func bytes_to_uint64(in [8]byte) (out uint64) {
	out = 0
	out = (out + uint64(in[0])) << 8
	out = (out + uint64(in[1])) << 8
	out = (out + uint64(in[2])) << 8
	out = (out + uint64(in[3])) << 8
	out = (out + uint64(in[4])) << 8
	out = (out + uint64(in[5])) << 8
	out = (out + uint64(in[6])) << 8
	out = (out + uint64(in[7]))
	return out
}

func uint16_to_bytes(in uint16) (out [2]byte) {
	out[1] = byte(in % 256)
	out[0] = byte((in >> 8) % 256)
	return out
}
func bytes_to_uint16(in [2]byte) (out uint16) {
	out = 0
	out = (out + uint16(in[0])) << 8
	out = (out + uint16(in[1]))
	return out
}

func swap_endian(data [32]byte) [32]byte {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
	return data
}
