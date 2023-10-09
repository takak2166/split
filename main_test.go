package main

import (
	"fmt"
	"os"
	"testing"
)

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		valid    bool
		input    string
		expected uint64
	}{
		{true, "1KB", 1000},
		{true, "1kB", 1000},
		{true, "20MB", 20 * 1000 * 1000},
		{true, "20mB", 20 * 1000 * 1000},
		{true, "300GB", 300 * 1000 * 1000 * 1000},
		{true, "4000TB", 4000 * 1000 * 1000 * 1000 * 1000},
		{true, "16383PB", 16383 * 1000 * 1000 * 1000 * 1000 * 1000},
		{true, "1K", 1024},
		{true, "1k", 1024},
		{true, "1KiB", 1024},
		{true, "1kiB", 1024},
		{true, "20M", 20 * 1024 * 1024},
		{true, "20MiB", 20 * 1024 * 1024},
		{true, "20miB", 20 * 1024 * 1024},
		{true, "300G", 300 * 1024 * 1024 * 1024},
		{true, "4000T", 4000 * 1024 * 1024 * 1024 * 1024},
		{true, "16383P", 16383 * 1024 * 1024 * 1024 * 1024 * 1024},
		{true, "12345", 12345},
		{false, "16384P", 0},
		{false, "KB", 0},
		{false, "K", 0},
		{false, "Ki", 0},
		{false, "ki", 0},
		{false, "10B", 0},
		{false, "10KA", 0},
		{false, "10Ki", 0},
		{false, "10ki", 0},
		{false, "10g", 0},
		{false, "10giB", 0},
		{false, "10gB", 0},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := parseByteSize(test.input)
			if err != nil && test.valid {
				t.Errorf("Error parsing byte size for input %s: %v", test.input, err)
			}
			if err == nil && !test.valid {
				t.Errorf("Unexpected result : error is nil, result is %v", result)
			}
			if result != test.expected && test.valid {
				t.Errorf("Unexpected result for input %s: got %d, expected %d", test.input, result, test.expected)
			}
		})
	}
}

func TestGenFileName(t *testing.T) {
	tests := []struct {
		valid     bool
		prefix    string
		index     uint64
		fileCount uint64
		splitType SplitType
		expected  string
	}{
		{true, "prefix", 0, 1, ByFiles, "prefixaa"},
		{true, "prefix", 1, 2, ByFiles, "prefixab"},
		{true, "", 0, 26, ByFiles, "xaa"},
		{true, "", 0, 676, ByFiles, "xaa"},
		{true, "", 25, 27, ByFiles, "xaz"},
		{true, "", 26, 27, ByFiles, "xba"},
		{true, "", 675, 676, ByFiles, "xzz"},
		{true, "", 0, 677, ByFiles, "xaaa"},
		{true, "", 676, 677, ByFiles, "xbaa"},
		{true, "", 17575, 17576, ByFiles, "xzzz"},
		{true, "", 0, 17577, ByFiles, "xaaaa"},
		{true, "", 17576, 17577, ByFiles, "xbaaa"},
		{false, "", 0, 0, ByFiles, ""},
		{false, "", 1, 0, ByFiles, ""},
		{true, "prefix", 0, 0, ByBytes, "prefixaa"},
		{true, "", 1, 0, ByBytes, "xab"},
		{true, "", 649, 0, ByBytes, "xyz"},
		{true, "", 650, 0, ByBytes, "xzaaa"},
		{true, "", 651, 0, ByBytes, "xzaab"},
		{true, "", 676, 0, ByBytes, "xzaba"},
		{true, "", 17549, 0, ByBytes, "xzyzz"},
		{true, "", 17550, 0, ByBytes, "xzzaaaa"},
		{true, "", 456949, 0, ByBytes, "xzzyzzz"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("splitType: %v index: %d fileCount: %d", test.splitType, test.index, test.fileCount), func(t *testing.T) {
			fileName, err := genFileName(test.prefix, test.index, test.fileCount, test.splitType)
			if err != nil && test.valid {
				t.Errorf("Unexpected error: %s", err)
			}
			if fileName != test.expected && test.valid {
				t.Errorf("Unexpected result: got %s, expected %s", fileName, test.expected)
			}
		})
	}
}

func TestSplitterSplit(t *testing.T) {
	wd, _ := os.Getwd()
	wd += "/"
	inputTestFileDir := wd + "testfiles/input/"
	outputTestFileDir := wd + "testfiles/output/"
	tests := []struct {
		name                string
		splitType           SplitType
		count               uint64
		inputFileName       string
		outputFilePrefix    string
		expectedOutputNames []string
	}{
		{
			name:                "split by bytes",
			splitType:           ByBytes,
			count:               50,
			inputFileName:       "sample.txt",
			outputFilePrefix:    "bytes_",
			expectedOutputNames: []string{"bytes_aa", "bytes_ab", "bytes_ac", "bytes_ad", "bytes_ae", "bytes_af", "bytes_ag", "bytes_ah"},
		},
		{
			name:                "split by lines",
			splitType:           ByLines,
			count:               2,
			inputFileName:       "sample.txt",
			outputFilePrefix:    "lines_",
			expectedOutputNames: []string{"lines_aa", "lines_ab", "lines_ac"},
		},
		{
			name:                "split by files",
			splitType:           ByFiles,
			count:               3,
			inputFileName:       "sample.txt",
			outputFilePrefix:    "files_",
			expectedOutputNames: []string{"files_aa", "files_ab", "files_ac"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inputFile, _ := os.Open(inputTestFileDir + test.inputFileName)
			defer inputFile.Close()

			splitter := NewSplitter(test.splitType, test.count, inputFile, test.outputFilePrefix)
			splitter.Split()

			for _, expectedOutputName := range test.expectedOutputNames {
				expectedOutputFilePath := outputTestFileDir + expectedOutputName
				outputFilePath := wd + expectedOutputName
				ok, err := func() (bool, error) {
					defer os.Remove(outputFilePath)
					return compareFileHashes(expectedOutputFilePath, outputFilePath)
				}()

				if err != nil {
					t.Errorf("Unexpected error: %s", err)
					return
				}
				if !ok {
					t.Errorf("The result of comparing %v and %v is not as expected.", expectedOutputFilePath, outputFilePath)
					return
				}
			}
		})
	}
}
