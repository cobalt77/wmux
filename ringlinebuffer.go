package wmux

import (
	"bufio"
	"bytes"
	"container/ring"
)

type RingLineBuffer struct {
	lineCount int
	head      *ring.Ring // Ring of byte slices
	current   *ring.Ring
}

func NewRingLineBuffer(lineCount int) *RingLineBuffer {
	wb := &RingLineBuffer{
		lineCount: lineCount,
		head:      ring.New(lineCount),
	}
	wb.current = wb.head
	for i := 0; i < lineCount; i++ {
		wb.head.Move(i).Value = make([]byte, 0)
	}
	return wb
}

func (wb *RingLineBuffer) writeLine(data []byte) int {
	wb.current.Value = data
	wb.current = wb.current.Next()
	if wb.current == wb.head {
		wb.head = wb.head.Next()
	}
	return len(data)
}

func (wb *RingLineBuffer) Write(data []byte) (int, error) {
	scan := bufio.NewScanner(bytes.NewReader(data))
	scan.Split(bufio.ScanLines)
	total := 0
	for scan.Scan() {
		total += wb.writeLine(scan.Bytes())
	}
	return total, nil
}
