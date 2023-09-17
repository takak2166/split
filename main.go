package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
)

type ErrorMsg string

const (
	InvalidByteSizeFormat       ErrorMsg = "Invalid byte size format"
	YouMustSpecifyOnlyOneOption ErrorMsg = "You must specify only one option"
	CannotDetermineFileSize     ErrorMsg = "Cannot determine file size"
	OverflowHasOccured          ErrorMsg = "Overflow has occured"
	InvalidSplitSize            ErrorMsg = "Invalid split size"
	InvalidIndex                ErrorMsg = "invalid index"
	DefaultCount                uint64   = 1000
)

func parseByteSize(input string) (uint64, error) {
	re := regexp.MustCompile(`^(\d+)([KMGTPkm]i?B?)?$`)
	matches := re.FindStringSubmatch(input)
	if len(matches) != 3 {
		return 0, fmt.Errorf("%s: %s", InvalidByteSizeFormat, input)
	}

	size, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0, err
	}

	unitVal := uint64(1)

	unit := matches[2]
	switch unit {
	case "KB", "kB":
		unitVal *= 1000
	case "MB", "mB":
		unitVal *= 1000 * 1000
	case "GB":
		unitVal *= 1000 * 1000 * 1000
	case "TB":
		unitVal *= 1000 * 1000 * 1000 * 1000
	case "PB":
		unitVal *= 1000 * 1000 * 1000 * 1000 * 1000
	case "K", "k", "KiB", "kiB":
		unitVal *= 1024
	case "M", "m", "MiB", "miB":
		unitVal *= 1024 * 1024
	case "G", "GiB":
		unitVal *= 1024 * 1024 * 1024
	case "T", "TiB":
		unitVal *= 1024 * 1024 * 1024 * 1024
	case "P", "PiB":
		unitVal *= 1024 * 1024 * 1024 * 1024 * 1024
	case "Ki", "Mi", "Gi", "Ti", "Pi", "ki", "mi":
		return 0, fmt.Errorf("%s: %s", InvalidByteSizeFormat, input)
	}

	result := size * unitVal

	if result/unitVal != size {
		return 0, fmt.Errorf("%s", OverflowHasOccured)
	}

	return result, nil
}

type SplitType int

const (
	ByBytes SplitType = iota
	ByLines
	ByFiles
)

type Splitter struct {
	splitType    SplitType
	count        uint64
	reader       io.Reader
	outputPrefix string
}

func NewSplitter(splitType SplitType, count uint64, reader io.Reader, outputPrefix string) *Splitter {
	return &Splitter{
		splitType:    splitType,
		count:        count,
		reader:       reader,
		outputPrefix: outputPrefix,
	}
}

func (s *Splitter) Split() error {
	switch s.splitType {
	case ByBytes:
		return s.splitByByte()
	case ByLines:
		return s.splitByLine()
	case ByFiles:
		return s.splitByFile()
	}

	return fmt.Errorf("%s", InvalidSplitSize)
}

func (s *Splitter) splitByByte() error {
	buffer := make([]byte, s.count)
	for i := uint64(0); ; i++ {
		n, err := s.reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		outputFile, err := s.createOutputFile(i)
		if err != nil {
			return err
		}
		defer outputFile.Close()

		if _, err := outputFile.Write(buffer[:n]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Splitter) splitByLine() error {
	fileIndex := uint64(0)
	lineCount := uint64(0)
	outputFile, err := s.createOutputFile(uint64(fileIndex))
	if err != nil {
		return err
	}
	defer outputFile.Close()

	buffer := bufio.NewReader(s.reader)
	for i := 0; ; i++ {
		line, err := buffer.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				outputFile.Write(line)
				break
			} else {
				return err
			}
		}

		if _, err := outputFile.Write(line); err != nil {
			return err
		}
		lineCount++

		if lineCount%s.count == 0 {
			outputFile.Close()
			fileIndex++
			outputFile, err = s.createOutputFile(fileIndex)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Splitter) splitByFile() error {
	fileBuf := new(bytes.Buffer)
	fileSize, err := io.Copy(fileBuf, s.reader)
	if err != nil {
		return err
	}
	byteCount := uint64(fileSize) / s.count
	byteRemain := uint64(fileSize) % s.count
	buffer := make([]byte, byteCount)
	for i := uint64(0); i < s.count; i++ {
		if i == s.count-1 {
			buffer = make([]byte, byteCount+byteRemain)
		}
		n, err := fileBuf.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		outputFile, err := s.createOutputFile(i)
		if err != nil {
			return err
		}
		defer outputFile.Close()

		if _, err := outputFile.Write(buffer[:n]); err != nil {
			return err
		}
	}

	return nil
}

func (s *Splitter) createOutputFile(index uint64) (*os.File, error) {
	outputFileName, err := genFileName(s.outputPrefix, index, s.count, s.splitType)
	if err != nil {
		return nil, err
	}

	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return outputFile, err
	}
	return outputFile, nil
}

// 生成されるファイル名の命名規則
func genFileName(prefix string, index uint64, fileCount uint64, splitType SplitType) (string, error) {
	if splitType == ByFiles && index+1 > fileCount {
		return "", fmt.Errorf("%s", InvalidIndex)
	}
	if prefix == "" {
		prefix = "x"
	}
	if splitType != ByFiles {
		fileCount = 26
		for i := 1; i < 14; i++ {
			if uint64(math.Pow(26, float64(i)))-26 <= index && index < uint64(math.Pow(26, float64(i+1)))-26 {
				index -= uint64(math.Pow(26, float64(i))) - 26
				fileCount = uint64(math.Pow(26, float64(i))) + 1
				for j := 1; j < i; j++ {
					prefix += "z"
				}
				break
			}
		}
	}
	tmp := ""
	if fileCount < 27 {
		tmp = fmt.Sprintf("%c", 'a'+(index%26))
		index /= 26
	}
	fileCountF := float64(fileCount)
	for fileCountF >= 1 {
		tmp = fmt.Sprintf("%c", 'a'+(index%26)) + tmp
		if fileCountF/26 == 1 {
			break
		}
		fileCountF /= 26
		index /= 26
	}
	return prefix + tmp, nil
}

type CLI struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (cli *CLI) Run(args []string) error {
	splitFlag := flag.NewFlagSet(args[0], flag.ExitOnError)
	splitFlag.Usage = func() {
		fmt.Fprintf(cli.Stderr, "Usage: %s [options...] <file> [prefix]\n", args[0])
		splitFlag.PrintDefaults()
	}
	// コマンドライン引数
	byteCountStr := splitFlag.String("b", "0", "Bytes per output file")
	lineCountP := splitFlag.Uint64("l", 0, "Number of lines per output file")
	fileCountP := splitFlag.Uint64("n", 0, "Number of output files")

	splitFlag.Parse(args[1:])

	byteCount, err := parseByteSize(*byteCountStr)
	if err != nil {
		return err
	}
	lineCount := *lineCountP
	fileCount := *fileCountP

	// 複数の分割方法は指定不可
	if byteCount > 0 && (lineCount > 0 || fileCount > 0) {
		return fmt.Errorf("%s", YouMustSpecifyOnlyOneOption)
	} else if lineCount > 0 && fileCount > 0 {
		return fmt.Errorf("%s", YouMustSpecifyOnlyOneOption)
	}

	var reader io.Reader
	outputPrefix := "x"

	// ファイルの指定がない場合やFILEが"-"の場合、 標準入力から読み込みを行う
	if splitFlag.Arg(0) == "" || splitFlag.Arg(0) == "-" {
		if fileCount > 0 {
			return fmt.Errorf("%s", CannotDetermineFileSize)
		}
		reader = cli.Stdin
	} else {
		inputFilePath := splitFlag.Arg(0)
		outputPrefix = splitFlag.Arg(1)

		inputFile, err := os.Open(inputFilePath)
		if err != nil {
			return err
		}
		defer inputFile.Close()

		reader = inputFile
	}

	splitter := NewSplitter(ByLines, DefaultCount, reader, outputPrefix)

	if byteCount > 0 {
		splitter = NewSplitter(ByBytes, byteCount, reader, outputPrefix)

	} else if lineCount > 0 {
		splitter = NewSplitter(ByLines, lineCount, reader, outputPrefix)

	} else if fileCount > 0 {
		splitter = NewSplitter(ByFiles, fileCount, reader, outputPrefix)
	}

	if err = splitter.Split(); err != nil {
		return err
	}

	return nil
}

func main() {
	cli := &CLI{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
	}

	if err := cli.Run(os.Args); err != nil {
		panic(err)
	}
}

func getFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func compareFileHashes(filePath1, filePath2 string) (bool, error) {
	hash1, err := getFileHash(filePath1)
	if err != nil {
		return false, err
	}

	hash2, err := getFileHash(filePath2)
	if err != nil {
		return false, err
	}

	return hash1 == hash2, nil
}
