package local

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/nexthink-oss/ghup/internal/util"
)

var (
	ErrInvalidSpec        = errors.New("invalid spec")
	ErrNoBranchSpec       = errors.New("empty branch spec")
	ErrInvalidBranchName  = errors.New("invalid branch spec")
	ErrEmptySourceSpec    = errors.New("empty source spec")
	ErrEmptyTargetSpec    = errors.New("empty target spec")
	ErrSourceEqualsTarget = errors.New("source and target files are the same")
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
			errs = append(errs, ErrInvalidBranchName)
		}

		source = filepath.Clean(parts[1])
		target = filepath.Clean(parts[2])

	default:
		errs = append(errs, ErrInvalidSpec)
	}

	if source == "" {
		errs = append(errs, ErrEmptySourceSpec)
	}

	if target == "" {
		errs = append(errs, ErrEmptyTargetSpec)
	}

	if source == target {
		errs = append(errs, ErrSourceEqualsTarget)
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
		source = files[0]
		target = files[0]

	case 2:
		source = files[0]
		target = files[1]

	default:
		errs = append(errs, ErrInvalidSpec)
	}

	if source == "" {
		errs = append(errs, ErrEmptySourceSpec)
	}

	if target == "" {
		errs = append(errs, ErrEmptyTargetSpec)
	}

	if len(errs) > 0 {
		source = ""
		target = ""
		err = fmt.Errorf("update-spec %q: %w", spec, errors.Join(errs...))
	} else {
		source = filepath.Clean(source)
		target = filepath.Clean(target)
	}

	return source, target, err
}
