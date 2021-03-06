package table

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/grokify/simplego/encoding/csvutil"
	"github.com/grokify/simplego/encoding/jsonutil"
	"github.com/grokify/simplego/type/stringsutil"
	"github.com/pkg/errors"
)

func NewTableFilesSimple(filenames []string, sep string, hasHeader, trimSpace bool) (Table, error) {
	tbl := NewTable()
	for i, filename := range filenames {
		filename = strings.TrimSpace(filename)
		if len(filename) == 0 {
			continue
		}
		tblx, err := NewTableFileSimple(filename, sep, hasHeader, trimSpace)
		if err != nil {
			return tbl, err
		}
		if len(tbl.Columns) == 0 {
			tbl.Columns = tblx.Columns
		} else {
			curCols := strings.Join(tbl.Columns, ",")
			nowCols := strings.Join(tblx.Columns, ",")
			if curCols != nowCols {
				if i == 0 {
					// if len(tbl.Columns) > 0, i has to be > 0
					panic("E_BAD_FILE_COUNTER_TABLE_COLUMNS")
				}
				return tbl, fmt.Errorf("CSV table definition mismatch [%s] AND [%s] for FILES [%s]",
					curCols, nowCols,
					filenames[i-1]+","+filename)
			}
		}
		if len(tblx.Records) > 0 {
			tbl.Records = append(tbl.Records, tblx.Records...)
		}
	}
	return tbl, nil
}

func NewTableFileSimple(path string, sep string, hasHeader, trimSpace bool) (Table, error) {
	tbl := NewTable()
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return tbl, err
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if trimSpace {
			line = strings.TrimSpace(line)
		}
		parts := strings.Split(line, sep)
		parts = stringsutil.SliceTrimSpace(parts, false)
		if hasHeader && i == 0 {
			tbl.Columns = parts
		} else {
			tbl.Records = append(tbl.Records, parts)
		}
	}
	return tbl, nil
}

// ReadFile reads in a delimited file and returns a `Table` struct.
func ReadFile(path string, comma rune, hasHeader, stripBom bool) (Table, error) {
	tbl := NewTable()
	csvReader, f, err := csvutil.NewReader(path, comma, stripBom)
	if err != nil {
		return tbl, err
	}
	defer f.Close()
	if DebugReadCSV {
		i := -1
		for {
			line, err := csvReader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				return tbl, err
			}
			i++
			if i == 0 && hasHeader {
				tbl.Columns = line
				continue
			}
			tbl.Records = append(tbl.Records, line)
			if i > 2500 {
				fmt.Printf("[%v] %v\n", i, strings.Join(line, ","))
			}
		}
	} else {
		lines, err := csvReader.ReadAll()
		if err != nil {
			return tbl, err
		}
		if hasHeader {
			tbl.LoadMergedRows(lines)
		} else {
			tbl.Records = lines
		}
	}
	return tbl, nil
}

func ReadMergeFilterCSVFiles(inPaths []string, outPath string, inComma rune, inStripBom bool, andFilter map[string]stringsutil.MatchInfo) (DocumentsSet, error) {
	//data := JsonRecordsInfo{Records: []map[string]string{}}
	data := NewDocumentsSet()

	for _, inPath := range inPaths {
		reader, inFile, err := csvutil.NewReader(inPath, inComma, inStripBom)
		if err != nil {
			return data, err
		}

		csvHeader := csvutil.CSVHeader{}
		j := -1

		for {
			line, err := reader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				return data, err
			}
			j++

			if j == 0 {
				csvHeader.Columns = line
				continue
			}
			match, err := csvHeader.RecordMatch(line, andFilter)
			if err != nil {
				return data, err
			}
			if !match {
				continue
			}

			mss := csvHeader.RecordToMSS(line)
			data.Documents = append(data.Documents, mss)
		}
		err = inFile.Close()
		if err != nil {
			return data, err
		}
	}
	data.Inflate()
	return data, nil
}

func MergeFilterCSVFilesToJSON(inPaths []string, outPath string, inComma rune, inStripBom bool, perm os.FileMode, andFilter map[string]stringsutil.MatchInfo) error {
	data, err := ReadMergeFilterCSVFiles(inPaths, outPath, inComma, inStripBom, andFilter)
	if err != nil {
		return err
	}
	bytes, err := jsonutil.MarshalSimple(data, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(outPath, bytes, perm)
}

func ReadCSVFilesSingleColumnValuesString(files []string, sep string, hasHeader, trimSpace bool, col uint, condenseUniqueSort bool) ([]string, error) {
	values := []string{}
	for _, file := range files {
		fileValues, err := ReadCSVFileSingleColumnValuesString(
			file, sep, hasHeader, trimSpace, col, false)
		if err != nil {
			return values, err
		}
		values = append(values, fileValues...)
	}
	if condenseUniqueSort {
		values = stringsutil.SliceCondenseSpace(values, true, true)
	}
	return values, nil
}

func ReadCSVFileSingleColumnValuesString(filename, sep string, hasHeader, trimSpace bool, col uint, condenseUniqueSort bool) ([]string, error) {
	tbl, err := NewTableFileSimple(filename, sep, hasHeader, trimSpace)
	if err != nil {
		return []string{}, err
	}
	values := []string{}
	for _, row := range tbl.Records {
		if len(row) > int(col) {
			values = append(values, row[col])
		}
	}
	if condenseUniqueSort {
		values = stringsutil.SliceCondenseSpace(values, true, true)
	}
	return values, nil
}

func ParseBytes(data []byte, delimiter rune, hasHeaderRow bool) (Table, error) {
	return ParseReader(bytes.NewReader(data), delimiter, hasHeaderRow)
}

func ParseReader(reader io.Reader, delimiter rune, hasHeaderRow bool) (Table, error) {
	tbl := NewTable()
	csvReader := csv.NewReader(reader)
	csvReader.Comma = delimiter
	csvReader.TrimLeadingSpace = true
	idx := -1
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return tbl, err
		}
		idx++
		if idx == 0 && hasHeaderRow {
			tbl.Columns = record
			continue
		}
		tbl.Records = append(tbl.Records, record)
	}
	return tbl, nil
}

// Unmarshal is a convenience function to provide a simple interface to
// unmarshal table contents into any desired output.
func (tbl *Table) Unmarshal(funcRecord func(record []string) error) error {
	for i, rec := range tbl.Records {
		err := funcRecord(rec)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error on Record Index [%d]", i))
		}
	}
	return nil
}
