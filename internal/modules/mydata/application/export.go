// Package application lets a person take a copy of everything they have here.
package application

import (
	"context"
	"fmt"
	"io"
)

// Archive is where an export is written, one file at a time.
type Archive interface {
	// Create starts a file at path; the caller writes its content to the result.
	Create(path string) (io.Writer, error)
}

// Section writes one part of a person's data into the archive.
type Section interface {
	Write(ctx context.Context, userID string, out Archive) error
}

// Exporter gathers every section into one archive.
type Exporter struct {
	sections []Section
}

// NewExporter builds an exporter that runs the sections in order.
func NewExporter(sections ...Section) *Exporter {
	return &Exporter{sections: sections}
}

// Export writes all of the person's data to out.
func (e *Exporter) Export(ctx context.Context, userID string, out Archive) error {
	for _, section := range e.sections {
		if err := section.Write(ctx, userID, out); err != nil {
			return fmt.Errorf("export: %w", err)
		}
	}

	return nil
}
