package table

import (
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type KeyMap struct {
	LineUp   key.Binding
	LineDown key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	ColRight key.Binding
	ColLeft  key.Binding
}

type Model struct {
	KeyMap KeyMap

	cols       []string
	rows       [][]string
	paddings   []int
	cursor_row int
	cursor_col int

	viewport viewport.Model
	start    int
	end      int
}

func DefaultKeyMap() KeyMap {
	const spacebar = " "
	return KeyMap{
		LineUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		LineDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("b", "pgup"),
			key.WithHelp("b/pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("f", "pgdown", spacebar),
			key.WithHelp("f/pgdn", "page down"),
		),
		ColRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("right", "right"),
		),
		ColLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("left", "left"),
		),
	}
}

// Option is used to set options in New. For example:
//
//	table := New(WithColumns([]Column{{Title: "ID", Width: 10}}))
type Option func(*Model)

func New(opts ...Option) Model {
	m := Model{
		cursor_row: 0,
		cursor_col: 0,
		KeyMap:     DefaultKeyMap(),
	}

	for _, opt := range opts {
		opt(&m)
	}
	m.viewport = viewport.New(0, min(30, len(m.rows)))
	m.paddings = m.calc_paddings()

	m.UpdateViewport()

	return m
}

func (m Model) View() string {
	return m.headersView() + "\n" + m.viewport.View()
}

func WithColumns(cols []string) Option {
	return func(m *Model) {
		m.cols = cols
	}
}

// WithRows sets the table rows (data).
func WithRows(rows [][]string) Option {
	return func(m *Model) {
		m.rows = rows
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.LineUp):
			m.MoveUp(1)
		case key.Matches(msg, m.KeyMap.LineDown):
			m.MoveDown(1)
		case key.Matches(msg, m.KeyMap.PageUp):
			m.MoveUp(m.viewport.Height)
		case key.Matches(msg, m.KeyMap.PageDown):
			m.MoveDown(m.viewport.Height)
		case key.Matches(msg, m.KeyMap.ColLeft):
			m.MoveLeft(1)
		case key.Matches(msg, m.KeyMap.ColRight):
			m.MoveRight(1)
		}

	}

	return m, nil
}

func (m Model) headersView() string {
	s := make([]string, 0, len(m.cols)+1)
	s = append(s, lipgloss.NewStyle().
		Width(max(len(strconv.Itoa(len(m.rows))), 2)).
		Align(lipgloss.Right).
		Foreground(lipgloss.Color("#ffffff")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Render(" "))

	for i := m.cursor_col; i < len(m.cols); i++ {
		value := m.cols[i]
		width := m.paddings[i]
		textStyle := lipgloss.NewStyle().
			Inline(true).
			Bold(true).
			MaxWidth(width).
			Width(width)

		cellStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Padding(0, 1)

		s = append(s, cellStyle.Render(textStyle.Render(value)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, s...)
}

func (m *Model) UpdateViewport() {
	renderedRows := make([]string, 0, len(m.rows))

	if m.cursor_row >= 0 {
		m.start = clamp(m.cursor_row-m.viewport.Height, 0, m.cursor_row)
	} else {
		m.start = 0
	}
	m.end = clamp(m.cursor_row+m.viewport.Height, m.cursor_row, len(m.rows))

	for i := m.start; i < m.end; i++ {
		renderedRows = append(renderedRows, m.renderRow(i))
	}

	m.viewport.SetContent(
		lipgloss.JoinVertical(lipgloss.Left, renderedRows...),
	)
}

func (m *Model) renderRow(r int) string {
	s := make([]string, 0, len(m.cols)+1)
	s = append(s, lipgloss.NewStyle().
		Width(max(len(strconv.Itoa(len(m.rows))), 2)).
		Align(lipgloss.Right).
		Align(lipgloss.Right).
		Render(strconv.FormatInt(int64(r), 10)))

	for i := m.cursor_col; i < len(m.rows[r]); i++ {
		value := m.rows[r][i]
		width := m.paddings[i]
		textStyle := lipgloss.NewStyle().
			Width(width).
			MaxWidth(width).
			Inline(true)
		cellStyle := lipgloss.NewStyle().
			Padding(0, 1)
		s = append(s, cellStyle.Render(textStyle.Render(value)))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, s...)

	if r == m.cursor_row {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Render(row)
	}

	return row
}

func (m *Model) calc_paddings() []int {
	var elements [][]string
	elements = append(elements, m.cols)
	elements = append(elements, m.rows...)
	paddings := make([]int, len(m.cols))

	for _, record := range elements {
		for j, value := range record {
			paddings[j] = max(paddings[j], len(value))
		}
	}
	for i := range paddings {
		paddings[i] = paddings[i] + 1
	}
	return paddings
}

// MoveUp moves the selection up by any number of rows.
// It can not go above the first row.
func (m *Model) MoveUp(n int) {
	m.cursor_row = clamp(m.cursor_row-n, 0, len(m.rows)-1)
	switch {
	case m.start == 0:
		m.viewport.SetYOffset(clamp(m.viewport.YOffset, 0, m.cursor_row))
	case m.start < m.viewport.Height:
		m.viewport.YOffset = (clamp(clamp(m.viewport.YOffset+n, 0, m.cursor_row), 0, m.viewport.Height))
	case m.viewport.YOffset >= 1:
		m.viewport.YOffset = clamp(m.viewport.YOffset+n, 1, m.viewport.Height)
	}
	m.UpdateViewport()
}

// MoveDown moves the selection down by any number of rows.
// It can not go below the last row.
func (m *Model) MoveDown(n int) {
	m.cursor_row = clamp(m.cursor_row+n, 0, len(m.rows)-1)
	m.UpdateViewport()

	switch {
	case m.end == len(m.rows) && m.viewport.YOffset > 0:
		m.viewport.SetYOffset(clamp(m.viewport.YOffset-n, 1, m.viewport.Height))
	case m.cursor_row > (m.end-m.start)/2 && m.viewport.YOffset > 0:
		m.viewport.SetYOffset(clamp(m.viewport.YOffset-n, 1, m.cursor_row))
	case m.viewport.YOffset > 1:
	case m.cursor_row > m.viewport.YOffset+m.viewport.Height-1:
		m.viewport.SetYOffset(clamp(m.viewport.YOffset+1, 0, 1))
	}
}

func (m *Model) MoveRight(n int) {
	m.cursor_col = clamp(m.cursor_col+n, 0, len(m.cols)-1)
	m.UpdateViewport()
}

func (m *Model) MoveLeft(n int) {
	m.cursor_col = clamp(m.cursor_col-n, 0, len(m.cols)-1)
	m.UpdateViewport()
}

func clamp(v, low, high int) int {
	return min(max(v, low), high)
}
