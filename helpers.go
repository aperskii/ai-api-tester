package main

import (
	"encoding/csv"
	"fmt"
	"os"
)

// CSVWriter handles writing the results to a CSV file.
type CSVWriter struct {
	file     *os.File
	writer   *csv.Writer
	filename string
}

// NewCSVWriter creates a new CSVWriter instance.
func NewCSVWriter(filename string) (*CSVWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV file: %v", err)
	}
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"ImageName", "Difference"}); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %v", err)
	}

	return &CSVWriter{
		file:     file,
		writer:   writer,
		filename: filename,
	}, nil
}

// WriteDifference writes a difference to the CSV file.
func (w *CSVWriter) WriteDifference(imageName, difference string) error {
	return w.writer.Write([]string{imageName, difference})
}

// Close closes the CSV file.
func (w *CSVWriter) Close() error {
	w.writer.Flush()
	return w.file.Close()
}

// Function to compare two Blocks struct and return the differences
func compareBlocks(resp1, resp2 string) []string {
	var differences []string

	// Compare the "Text" field in both responses
	if resp1 != resp2 {
		differences = append(differences, resp1, resp2)
	}
	return differences
}
