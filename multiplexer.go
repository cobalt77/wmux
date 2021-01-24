package wmux

import (
	"bytes"
	"container/ring"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/vbauerster/mpb/cwriter"
)

type Dimensions struct {
	// Number of columns
	Columns int

	// Number of rows
	Rows int

	// Minimum horizontal split width
	MinColumnWidth int

	// Fixed height
	Height int
}

func (d Dimensions) requiredWidth() int {
	return d.Columns * d.MinColumnWidth
}

type Multiplexer struct {
	writer     *cwriter.Writer
	buffers    []*RingLineBuffer
	dimensions Dimensions
}

func splitWidths(dim Dimensions, width int) []int {
	width -= dim.Columns - 1 // Borders
	if width < 0 {
		panic("Bug: width < 0")
	}
	widths := make([]int, dim.Columns)
	for i := 0; i < dim.Columns; i++ {
		widths[i] = (width / dim.Columns)
	}
	// Add the remaining space to the last split
	widths[dim.Columns-1] += (width % dim.Columns)
	return widths
}

func NewMultiplexer(out io.Writer, dim Dimensions) (*Multiplexer, error) {
	m := &Multiplexer{
		writer:     cwriter.New(out),
		dimensions: dim,
	}

	m.buffers = make([]*RingLineBuffer, dim.Rows*dim.Columns)
	for i := 0; i < dim.Rows*dim.Columns; i++ {
		m.buffers[i] = NewRingLineBuffer(dim.Height)
	}
	return m, nil
}

func (m *Multiplexer) drawRowBorder() {
	w, _ := m.writer.GetWidth()
	color.New(color.FgBlue).
		Fprintln(m.writer, color.BlueString(strings.Repeat("-", w)))
}

func (m *Multiplexer) drawColumnBorder() {
	color.New(color.FgBlue).
		Fprint(m.writer, color.BlueString("|"))
}

var lastWidth int = -1

func (m *Multiplexer) render() error {
	numLines := (m.dimensions.Height * m.dimensions.Rows) +
		(m.dimensions.Rows + 1)

	w, _ := m.writer.GetWidth()
	if w < lastWidth {
		m.writer.Flush(numLines)
	}
	lastWidth = w

	columnWidths := m.calcColumnWidths()
	rings := make([]*ring.Ring, len(m.buffers))
	for i, b := range m.buffers {
		rings[i] = b.head
	}
	m.drawRowBorder()
	for row := 0; row < m.dimensions.Rows; row++ {
		for line := 0; line < m.dimensions.Height; line++ {
			for col := 0; col < m.dimensions.Columns; col++ {
				if columnWidths[col] == 0 {
					continue
				}
				lineBuf := bytes.Repeat([]byte{' '}, columnWidths[col])
				line := rings[col].Value.([]byte)
				copy(lineBuf, line)
				_, err := m.writer.Write(lineBuf)
				if err != nil {
					return err
				}
				if col < m.dimensions.Columns-1 {
					m.drawColumnBorder()
				}
				rings[col] = rings[col].Next()
			}
			m.writer.Write([]byte{'\n'})
		}
		m.drawRowBorder()
	}
	return m.writer.Flush(numLines)
}

func (m *Multiplexer) Run(done <-chan struct{}) error {
	timer := make(chan struct{})
	defer close(timer)

	go func() {
		for {
			time.Sleep(62 * time.Millisecond) // ~16hz
			timer <- struct{}{}
		}
	}()

	for {
		select {
		case <-timer:
			m.render()
		case <-done:
			return nil
		}
	}
}

func (m *Multiplexer) Writers() []io.Writer {
	writers := make([]io.Writer, len(m.buffers))
	for i, b := range m.buffers {
		writers[i] = b
	}
	return writers
}

func (m *Multiplexer) calcColumnWidths() []int {
	w, _ := m.writer.GetWidth()
	borders := m.dimensions.Columns - 1
	availableWidth := w - borders
	visibleColumns := m.dimensions.Columns
	for m.dimensions.MinColumnWidth*visibleColumns > availableWidth {
		visibleColumns--
	}
	widths := make([]int, m.dimensions.Columns)
	for i := 0; i < visibleColumns; i++ {
		widths[i] = availableWidth / visibleColumns
	}
	widths[visibleColumns-1] += availableWidth % visibleColumns

	for i := visibleColumns; i < m.dimensions.Columns; i++ {
		widths[i] = 0
	}
	return widths
}

func (m *Multiplexer) WriterAt(row, col int) io.Writer {
	return m.buffers[row*m.dimensions.Columns+col]
}
