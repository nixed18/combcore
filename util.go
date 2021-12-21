package main

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
