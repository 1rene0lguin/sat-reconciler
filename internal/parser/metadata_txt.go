package parser

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/1rene0lguin/sat-reconciler/internal/adapters/sat"
)

// ParseMetadataTxt procesa el archivo .txt separado por ~ del SAT.
func ParseMetadataTxt(reader io.Reader) ([]sat.Metadata, error) {
	scanner := bufio.NewScanner(reader)
	var results []sat.Metadata
	lineCount := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Basic cleanup
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Official separator is ~
		fields := strings.Split(line, sat.MetadataSeparator)

		// Validation of minimal structure (at least until UUID and RFCs)
		if len(fields) < 5 {
			// If it's the first line and fails, it's likely a header or garbage
			if lineCount == 1 {
				continue
			}
			return nil, fmt.Errorf("malformed line %d: %d fields found", lineCount, len(fields))
		}

		// Detect and skip header if exists
		if strings.HasPrefix(fields[0], "Uuid") || strings.HasPrefix(fields[0], "UUID") {
			continue
		}

		// Object construction
		meta := sat.Metadata{
			UUID:         fields[0],
			RfcIssuer:    fields[1],
			NameIssuer:   fields[2],
			RfcReceiver:  fields[3],
			NameReceiver: fields[4],
			// fields[5] is RfcPac (not mapped in simple struct)
			// Dates (indices 6 and 7) processed below
			// Amount (index 8)
			// Effect (index 9)
			// Status (index 10)
		}

		// Date Parsing (Defensive)
		if len(fields) > 6 {
			meta.DateEmission = parseSatDate(fields[6])
		}
		if len(fields) > 7 {
			meta.DateCertification = parseSatDate(fields[7])
		}

		// Amount Parsing
		if len(fields) > 8 {
			if val, err := strconv.ParseFloat(strings.TrimSpace(fields[8]), 64); err == nil {
				meta.Total = val
			}
		}

		// Effect and Status Parsing
		if len(fields) > 10 {
			meta.TypeVoucher = fields[9]
			// SAT returns "1" for Vigente (Active), "0" for Cancelado (Cancelled)
			if fields[10] == sat.StatusVigente {
				meta.Status = "Vigente"
			} else {
				meta.Status = "Cancelado"
			}
		}

		// Cancellation Date (if exists)
		if len(fields) > 11 && strings.TrimSpace(fields[11]) != "" {
			cancelDate := parseSatDate(fields[11])
			meta.DateCancellation = &cancelDate
		}

		results = append(results, meta)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// parseSatDate intenta interpretar la fecha en los múltiples formatos que el SAT usa.
func parseSatDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}
	}

	// Lista de formatos conocidos del SAT (prioridad al más común)
	layouts := []string{
		"02/01/2006 15:04:05", // dd/mm/yyyy HH:mm:ss (Común en TXT)
		"2006-01-02T15:04:05", // ISO8601 (Común en XML)
		"02/01/2006",          // Solo fecha
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t
		}
	}

	// Si falla, retornamos Zero Time (0001-01-01) en lugar de error para no romper el batch.
	// En un log real, aquí pondríamos un warning.
	return time.Time{}
}
