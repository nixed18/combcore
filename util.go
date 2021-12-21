package main

import "fmt"

func Hex(x byte) byte {
	return 7*(x/10) + x + '0'
}

func checkHEX(b string, length int) bool {
	if len(b) != 2*length {
		return false
	}
	for i := 0; i < 2*length; i++ {
		if ((b[i] >= '0') && (b[i] <= '9')) || ((b[i] >= 'A') && (b[i] <= 'F')) || ((b[i] >= 'a') && (b[i] <= 'f')) {
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
	return [32]byte{13, 14, 15, 0, 0, 10, 11, 12, 0, 0, 0, 0, 0, 0, 0, 0, 3, 2, 1, 0, 7, 6, 5, 4, 0, 0, 9, 8, 0, 0, 0, 0}[(hex^(hex>>4))&31]
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
