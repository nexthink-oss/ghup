package local

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/nexthink-oss/ghup/internal/util"
)

var (
	InvalidSpecError        = errors.New("invalid spec")
	NoBranchSpecError       = errors.New("empty branch spec")
	InvalidBranchNameError  = errors.New("invalid branch spec")
	EmptySourceSpecError    = errors.New("empty source spec")
	EmptyTargetSpecError    = errors.New("empty target spec")
	SourceEqualsTargetError = errors.New("source and target files are the same")
)

// ParseCopySpec parses a file specification into remote source andtarget file paths.
// The separator is used to split the source and target file paths.
// All file paths are cleaned before being returned.
func ParseCopySpec(spec, separator string) (branch, source, target string, err error) {
	parts := strings.Split(spec, separator)
	errs := make([]error, 0)

	switch len(parts) {
	case 2:
		source = filepath.Clean(parts[0])
		target = filepath.Clean(parts[1])

	case 3:
		branch = parts[0]
		if err := util.IsValidRefName(branch); err != nil {
			errs = append(errs, InvalidBranchNameError)
		}

		source = filepath.Clean(parts[1])
		target = filepath.Clean(parts[2])

	default:
		errs = append(errs, InvalidSpecError)
	}

	if source == "" {
		errs = append(errs, EmptySourceSpecError)
	}

	if target == "" {
		errs = append(errs, EmptyTargetSpecError)
	}

	if source == target {
		errs = append(errs, SourceEqualsTargetError)
	}

	if len(errs) > 0 {
		err = fmt.Errorf("copy-spec %q: %w", spec, errors.Join(errs...))
	}

	return branch, source, target, err
}

// ParseUpdateSpec parses a file specification into source and target file paths.
// The separator is used to split the source and target file paths, if present.
// All file paths are cleaned before being returned.
func ParseUpdateSpec(spec, separator string) (source, target string, err error) {
	files := strings.SplitN(spec, separator, 2)
	errs := make([]error, 0)

	switch len(files) {
	case 1:
		clean := filepath.Clean(files[0])
		source = clean
		target = clean

	case 2:
		source = filepath.Clean(files[0])
		target = filepath.Clean(files[1])

	default:
		errs = append(errs, InvalidSpecError)
	}

	if source == "" {
		errs = append(errs, EmptySourceSpecError)
	}

	if target == "" {
		errs = append(errs, EmptyTargetSpecError)
	}

	if len(errs) > 0 {
		err = fmt.Errorf("update-spec %q: %w", spec, errors.Join(errs...))
	}

	return source, target, err
}
