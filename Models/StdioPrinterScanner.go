package Models

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type StdioPrinterScanner struct {
}

func (s StdioPrinterScanner) GetAnswer(string2 string) string {
	_, err := fmt.Fprintln(os.Stdout, string2)
	if err != nil {
		s.DisplayMessage(err.Error())
		return ""
	}
	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	answer = strings.TrimSpace(answer)
	if err != nil {
		s.DisplayMessage(answer)
		s.DisplayMessage(err.Error())
	}
	return answer
}

func (s StdioPrinterScanner) DisplayMessage(string2 string) {
	_, err := fmt.Fprintln(os.Stdout, string2)
	if err != nil {
		return
	}
}
