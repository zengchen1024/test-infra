/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package approvers

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

const ownersFileName = "OWNERS"

// File in an interface for files
type File interface {
	String() string
}

// Approval has the information about each approval on a PR
type Approval struct {
	Login     string // Login of the approver (can include uppercase)
	How       string // How did the approver approved
	Reference string // Where did the approver approved
	NoIssue   bool   // Approval also accepts missing associated issue
}

// String creates a link for the approval. Use `Login` if you just want the name.
func (a Approval) String() string {
	return fmt.Sprintf(
		`*<a href="%s" title="%s">%s</a>*`,
		a.Reference,
		a.How,
		a.Login,
	)
}

// ApprovedFile contains the information of a an approved file.
type ApprovedFile struct {
	baseURL  *url.URL
	filepath string
	// approvers is the set of users that approved this file change.
	approvers sets.String
	branch    string
}

func (a ApprovedFile) String() string {
	fullOwnersPath := filepath.Join(a.filepath, ownersFileName)
	if strings.HasSuffix(a.filepath, ".md") {
		fullOwnersPath = a.filepath
	}
	link := fmt.Sprintf("%s/blob/%s/%v",
		a.baseURL.String(),
		a.branch,
		fullOwnersPath,
	)
	return fmt.Sprintf("- ~~[%s](%s)~~ [%v]\n", fullOwnersPath, link, strings.Join(a.approvers.List(), ","))
}

// UnapprovedFile contains the information of a an unapproved file.
type UnapprovedFile struct {
	baseURL  *url.URL
	filepath string
	branch   string
}

func (ua UnapprovedFile) String() string {
	fullOwnersPath := filepath.Join(ua.filepath, ownersFileName)
	if strings.HasSuffix(ua.filepath, ".md") {
		fullOwnersPath = ua.filepath
	}
	link := fmt.Sprintf("%s/blob/%s/%v",
		ua.baseURL.String(),
		ua.branch,
		fullOwnersPath,
	)
	return fmt.Sprintf("- **[%s](%s)**\n", fullOwnersPath, link)
}
