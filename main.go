package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"tablespy/table"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xuri/excelize/v2"
)

var NO_DELIMITER_SET_VALUE rune = 0

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table table.Model
}

type command_args struct {
	filename     string
	file_type    string
	delimiter    rune
	table_height int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
			//tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func main() {
	args := parseArgs()

	columns, rows := readFile(args)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
	)

	m := model{t}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	//	print_records(records)
}

func readFile(args command_args) ([]string, [][]string) {
	content, err := os.ReadFile(args.filename)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file '%s': %v\n", args.filename, err)
		os.Exit(1)
	}

	contentString := string(content)

	switch args.file_type {
	case "csv":
		return parseCsv(contentString, args)
	case "excel":
		return parseXlsx(contentString)
	default:
		if strings.HasSuffix(args.filename, ".xlsx") || strings.HasSuffix(args.filename, ".xls") {
			return parseXlsx(contentString)
		} else {
			return parseCsv(contentString, args)
		}
	}
}

func parseCsv(content string, args command_args) ([]string, [][]string) {
	var delimiter = args.delimiter
	if args.delimiter == NO_DELIMITER_SET_VALUE {
		delimiter = guessDelimiter(content)
	}

	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = delimiter
	records, err := reader.ReadAll()

	if err != nil {
		fmt.Println("Error reading CSV from string:", err)
		os.Exit(1)
	}

	records = trim(records)

	columns, rows := records[0], records[1:]
	return columns, rows
}

func guessDelimiter(content string) rune {
	semi_count := 0
	comma_count := 0
	for _, char := range content {
		if char == ';' {
			semi_count += 1
		} else if char == ',' {
			comma_count += 1
		}
	}

	if semi_count > comma_count {
		return ';'
	} else {
		return ','
	}
}

func parseXlsx(content string) ([]string, [][]string) {
	f, err := excelize.OpenReader(strings.NewReader(content))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sheet := f.GetSheetList()[0]

	rows, err := f.GetRows(sheet)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	records := trim(rows)

	columns, rows := records[0], records[1:]
	return columns, rows
}

// func print_records(records [][]string) {
// 	paddings := calc_paddings(records)

// 	for i := 0; i < len(records); i++ {
// 		record := records[i]

// 		for j := 0; j < len(record); j++ {
// 			padding := paddings[j]
// 			fmt.Printf("%-*s", padding, record[j])
// 		}
// 		fmt.Println()
// 	}
// }

func trim(records [][]string) [][]string {
	for i := 0; i < len(records); i++ {
		record := records[i]
		for j := 0; j < len(record); j++ {
			record[j] = strings.Trim(record[j], " ")
		}
	}
	return records
}

func parseArgs() command_args {
	flag.Usage = printUsage

	fileTypePtr := flag.String("file_type", "auto", "force specific filetype, values: 'excel' or 'csv'")
	delimiterPtr := flag.String("delimiter", "auto", "char delimiter for when parsing csv, like ',' or ';'")
	heightPtr := flag.Int("height", 20, "max height for the table in rows")

	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "filename is required")
		os.Exit(0)
	}

	filename := args[0]

	allowedFileTypes := []string{"csv", "excel", "auto"}
	if !slices.Contains(allowedFileTypes, *fileTypePtr) {
		fmt.Fprintln(os.Stderr, "file_type can only be the following types: ", allowedFileTypes)
		os.Exit(0)
	}

	var delimiter rune
	if *delimiterPtr == "auto" {
		delimiter = NO_DELIMITER_SET_VALUE
	} else if len(*delimiterPtr) != 1 {
		fmt.Fprintln(os.Stderr, "delimiter can only be a single char")
		os.Exit(0)
	} else {
		delimiter, _ = utf8.DecodeRuneInString(*delimiterPtr)
	}

	return command_args{
		filename:     filename,
		file_type:    *fileTypePtr,
		table_height: *heightPtr,
		delimiter:    delimiter,
	}
}

func printUsage() {
	progname := filepath.Base(os.Args[0])

	fmt.Fprintf(os.Stderr, `Read table data from a file

Usage of %s:
	%s [flags] filename

Flags:
`, progname, progname)
	flag.PrintDefaults()

}
