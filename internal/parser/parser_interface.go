package parser

import (
	"github.com/zisuu/github-actions-digest-pinner/pgk/types"
)

type DefaultParser struct{}

func (d DefaultParser) ParseWorkflowActions(content []byte) ([]types.ActionRef, error) {
	return ParseWorkflowActions(content)
}
