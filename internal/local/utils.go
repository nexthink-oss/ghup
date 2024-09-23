package local

import (
	"fmt"
	"os"
	"strings"
)

// GetLocalFileContent loads the content of a file and returns the target path and its contents
func GetLocalFileContent(arg string, separator string) (target string, content []byte, err error) {
	var source string

	files := strings.SplitN(arg, separator, 2)

	switch {
	case len(files) < 1:
		err = fmt.Errorf("invalid file parameter")
		return
	case files[0] == "":
		err = fmt.Errorf("no source file specified")
		return
	case len(files) == 1:
		source = files[0]
		target = files[0]
	case files[1] == "":
		err = fmt.Errorf("no target file specified")
		return
	default:
		source = files[0]
		target = files[1]
	}

	content, err = os.ReadFile(source)
	return
}
