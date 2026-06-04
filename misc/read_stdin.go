package misc

import (
	"bufio"
	"os"

	"github.com/kaichao/gopkg/errors"
)

// ReadLinesFromStdin reads all lines from stdin. If stdin is a terminal
// (no pipe or redirection), it returns an error with message
// "no standard input detected".
func ReadLinesFromStdin() ([]string, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return []string{}, errors.WrapE(err, "failed to get stdin info")
	}
	if fi.Mode()&os.ModeCharDevice != 0 {
		return []string{},
			errors.E("no standard input detected")
	}

	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return []string{}, errors.WrapE(err, "failed to read standard input")
	}
	return lines, nil
}
