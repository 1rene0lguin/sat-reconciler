package parser

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/i4ene0lguin/sat-reconcilier/internal/sat"
)

// ParseMetadataTxt procesa el archivo .txt separado por ~ del SAT.
func ParseMetadataTxt(reader io.Reader) ([]sat.Metadata, error) {
	scanner := bufio.NewScanner(reader)
	var results []sat.Metadata
	lineCount := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Limpieza básica
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// El separador oficial es ~
		fields := strings.Split(line, "~")

		// Validación de estructura mínima (al menos hasta el UUID y RFCs)
		if len(fields) < 5 {
			// Si es la primera línea y falla, probablemente sea un header o basura
			if lineCount == 1 {
				continue
			}
			return nil, fmt.Errorf("linea %d mal formada: %d campos encontrados", lineCount, len(fields))
		}

		// Detectar y saltar cabecera si existe
		if strings.HasPrefix(fields[0], "Uuid") || strings.HasPrefix(fields[0], "UUID") {
			continue
		}

		// Construcción del objeto
		meta := sat.Metadata{
			UUID:           fields[0],
			RfcEmisor:      fields[1],
			NombreEmisor:   fields[2],
			RfcReceptor:    fields[3],
			NombreReceptor: fields[4],
			// fields[5] es RfcPac (no mapeado en struct simple)
			// Fechas (indices 6 y 7) se procesan abajo
			// Monto (indice 8)
			// Efecto (indice 9)
			// Estatus (indice 10)
		}

		// Parseo de Fechas (Defensivo)
		if len(fields) > 6 {
			meta.FechaEmision = parseSatDate(fields[6])
		}
		if len(fields) > 7 {
			meta.FechaCertificacion = parseSatDate(fields[7])
		}

		// Parseo de Monto
		if len(fields) > 8 {
			if val, err := strconv.ParseFloat(strings.TrimSpace(fields[8]), 64); err == nil {
				meta.Total = val
			}
		}

		// Parseo de Efecto y Estatus
		if len(fields) > 10 {
			meta.TipoComprobante = fields[9]
			// SAT devuelve "1" para Vigente, "0" para Cancelado
			if fields[10] == "1" {
				meta.Estatus = "Vigente"
			} else {
				meta.Estatus = "Cancelado"
			}
		}

		// Fecha Cancelación (si existe)
		if len(fields) > 11 && strings.TrimSpace(fields[11]) != "" {
			fechaCancel := parseSatDate(fields[11])
			meta.FechaCancelacion = &fechaCancel
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
