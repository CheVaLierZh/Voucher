package vouchercode

import (
	"bytes"
	"github.com/sigurn/crc8"
	"math/rand"
	"strconv"
	"time"
)


var table *crc8.Table

func init() {
	table = crc8.MakeTable(crc8.CRC8_MAXIM)
}

// Generator 码值生成器
type Generator struct {
	counter map[uint64]bool
}

func New() *Generator {
	return &Generator{counter: make(map[uint64]bool)}
}

func (g *Generator) Generate(seq int, n int) []string {
	rand.Seed(time.Now().UnixNano())
	res := make([]uint64, 0, n)
	for len(res) < n {
		r := uint64(rand.Int63())
		if ok := g.counter[r]; !ok {
			g.counter[r] = true
			res = append(res, r)
		}
	}

	resEnc := make([]string, 0, n)
	for i := 0; i < len(res); i++ {
		resEnc = append(resEnc, encode(seq, res[i]))
	}
	return resEnc
}

const (
	Uint64Mask = 0xaaaaaaaaaaaaaaaa
	Uint32Mask = 0xaaaaaaaa
)

func encode(seq int, x uint64) string {
	var buf bytes.Buffer
	seqEnc := strconv.FormatUint(uint64(seq) ^ Uint32Mask, 36)
	buf.WriteString(seqEnc)
	codeEnc := strconv.FormatUint(x ^ Uint64Mask, 36)
	buf.WriteString(codeEnc)

	crc := crc8.Checksum([]byte(seqEnc + codeEnc), table)
	buf.WriteString(strconv.FormatUint(uint64(crc), 36))
	return buf.String()
}

func Encode(seq int, x string) string {
	var buf bytes.Buffer
	seqEnc := strconv.FormatUint(uint64(seq) ^ Uint32Mask, 36)
	buf.WriteString(seqEnc)
	buf.WriteString(x)

	crc := crc8.Checksum([]byte(seqEnc + x), table)
	buf.WriteString(strconv.FormatUint(uint64(crc), 36))
	return buf.String()
}

func Decode(x string) (int, string, bool) {
	crc := crc8.Checksum([]byte(x[:20]), table)
	c, err := strconv.ParseUint(x[20:], 36, 8)
	if err != nil {
		return 0, "", false
	}
	if crc != uint8(c) {
		return 0, "", false
	}

	seq, _ := strconv.ParseUint(x[:7], 36, 64)
	seq = seq ^ Uint32Mask

	return int(seq), x[:20], true
}

