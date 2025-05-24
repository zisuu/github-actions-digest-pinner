package finder

import "io/fs"

type DefaultFinder struct{}

func (d DefaultFinder) FindWorkflowFiles(fsys fs.FS) ([]string, error) {
	return FindWorkflowFiles(fsys)
}
