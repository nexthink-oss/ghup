package local

import (
	"fmt"
	"strings"
)

func SplitUpdateSpec(spec, separator string) (source, target string, err error) {
	files := strings.SplitN(spec, separator, 2)

	switch {
	case len(files) < 1:
		return "", "", fmt.Errorf("invalid file parameter")
	case files[0] == "":
		return "", "", fmt.Errorf("no source file specified")
	case len(files) == 1:
		source = files[0]
		target = files[0]
	case files[1] == "":
		return "", "", fmt.Errorf("no target file specified")
	default:
		source = files[0]
		target = files[1]
	}

	return source, target, nil
}
