package repoowners

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"
	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/util/sets"
)

type ownersConfig struct {
	SimpleConfig
	Files map[string]Config `json:"files,omitempty"`
}

func normalConfig(c *Config) *Config {
	return &Config{
		Approvers:         NormLogins(c.Approvers).List(),
		Reviewers:         NormLogins(c.Reviewers).List(),
		RequiredReviewers: NormLogins(c.RequiredReviewers).List(),
		Labels:            c.Labels,
	}
}

func parseYaml(path string, r interface{}) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, r)
}

type RepoOwnerInfo struct {
	dirOwners    map[string]map[*regexp.Regexp]SimpleConfig
	fileOwners   map[string]map[*regexp.Regexp]Config
	baseDir      string
	dirBlacklist []*regexp.Regexp
	log          *logrus.Entry
}

var _ RepoOwner = (*RepoOwnerInfo)(nil)

func (o *RepoOwnerInfo) applyDirConfigToPath(path string, re *regexp.Regexp, config *SimpleConfig) {
	if _, ok := o.dirOwners[path]; !ok {
		o.dirOwners[path] = make(map[*regexp.Regexp]SimpleConfig)
	}

	o.dirOwners[path][re] = SimpleConfig{
		Config:  *normalConfig(&config.Config),
		Options: config.Options,
	}
}

func (o *RepoOwnerInfo) applyFileConfigToPath(path string, re *regexp.Regexp, config *Config) {
	if _, ok := o.fileOwners[path]; !ok {
		o.fileOwners[path] = make(map[*regexp.Regexp]Config)
	}

	o.fileOwners[path][re] = *normalConfig(config)
}

func (o *RepoOwnerInfo) parseOwnerConfig(path, relPathDir string, log *logrus.Entry) error {
	c := new(ownersConfig)
	if err := parseYaml(path, c); err != nil {
		return err
	}

	for pattern, config := range c.Files {
		if pattern != ".*" {
			continue
		}

		if re, err := regexp.Compile(pattern); err != nil {
			log.WithError(err).Errorf("Invalid regexp %q.", pattern)
		} else {
			o.applyFileConfigToPath(relPathDir, re, &config)
		}
	}

	if !c.SimpleConfig.Empty() {
		o.applyDirConfigToPath(relPathDir, nil, &c.SimpleConfig)
	}
	return nil
}

func (o *RepoOwnerInfo) walkFunc(path string, info os.FileInfo, err error) error {
	log := o.log.WithField("path", path)
	if err != nil {
		log.WithError(err).Error("Error while walking OWNERS files.")
		return nil
	}

	filename := filepath.Base(path)
	relPath, err := filepath.Rel(o.baseDir, path)
	if err != nil {
		log.WithError(err).Errorf("Unable to find relative path between baseDir: %q and path: %q.", o.baseDir, path)
		return err
	}
	relPathDir := canonicalize(filepath.Dir(relPath))

	if info.Mode().IsDir() {
		for _, re := range o.dirBlacklist {
			if re.MatchString(relPath) {
				return filepath.SkipDir
			}
		}
	}
	if !info.Mode().IsRegular() {
		return nil
	}

	if filename != ownersFileName {
		return nil
	}

	// if path is in a blacklisted directory, ignore it
	dir := filepath.Dir(path)
	for _, re := range o.dirBlacklist {
		if re.MatchString(dir) {
			return filepath.SkipDir
		}
	}

	if err := o.parseOwnerConfig(path, relPathDir, log); err != nil {
		log.WithError(err).Errorf("Failed to parse OWNERS %s.", path)
	}
	return nil
}

// findOwnersForFile returns the OWNERS file path furthest down the tree for a specified file
// using ownerMap to check for entries
func (o *RepoOwnerInfo) findOwnersForFile(path string, getValue func(*Config) []string) string {
	//TODO: is it a bug for original
	d := canonicalize(filepath.Dir(path))

	for re := range o.fileOwners[d] {
		if re != nil && re.MatchString(path) {
			return d
		}
	}

	for ; d != baseDirConvention; d = canonicalize(filepath.Dir(d)) {
		if m, ok := o.dirOwners[d]; ok {
			// TODO: m[(*regexp.Regexp)(nil)]
			if s, ok := m[nil]; ok {
				if s.Options.NoParentOwners || len(getValue(&s.Config)) > 0 {
					return d
				}

			}
		}
	}
	return baseDirConvention
}

// FindApproverOwnersForFile returns the OWNERS file path furthest down the tree for a specified file
// that contains an approvers section
func (o *RepoOwnerInfo) FindApproverOwnersForFile(path string) string {
	return o.findOwnersForFile(path, func(c *Config) []string {
		return c.Approvers
	})
}

// FindReviewersOwnersForFile returns the OWNERS file path furthest down the tree for a specified file
// that contains a reviewers section
func (o *RepoOwnerInfo) FindReviewersOwnersForFile(path string) string {
	return o.findOwnersForFile(path, func(c *Config) []string {
		return c.Reviewers
	})
}

// FindLabelsForFile returns a set of labels which should be applied to PRs
// modifying files under the given path.
func (o *RepoOwnerInfo) FindLabelsForFile(path string) sets.String {
	return o.entriesForFile(path, false, func(c *Config) []string {
		return c.Labels
	})
}

// IsNoParentOwners checks if an OWNERS file path refers to an OWNERS file with NoParentOwners enabled.
func (o *RepoOwnerInfo) IsNoParentOwners(path string) bool {
	if m, ok := o.dirOwners[path]; ok {
		// TODO: m[(*regexp.Regexp)(nil)]
		if s, ok := m[nil]; ok {
			return s.Options.NoParentOwners
		}
	}
	return false
}

// entriesForFile returns a set of users who are assignees to the
// requested file. The path variable should be a full path to a filename
// and not directory as the final directory will be discounted if enableMDYAML is true
// leafOnly indicates whether only the OWNERS deepest in the tree (closest to the file)
// should be returned or if all OWNERS in filepath should be returned
func (o *RepoOwnerInfo) entriesForFile(path string, leafOnly bool, getValue func(*Config) []string) sets.String {
	//TODO: is it a bug for original
	d := canonicalize(filepath.Dir(path))

	for re, s := range o.fileOwners[d] {
		if re != nil && re.MatchString(path) {
			return sets.NewString(getValue(&s)...)
		}
	}

	out := sets.NewString()
	for {
		if m, ok := o.dirOwners[d]; ok {
			// TODO: m[(*regexp.Regexp)(nil)]
			if s, ok := m[nil]; ok {
				out.Insert(getValue(&s.Config)...)

				if s.Options.NoParentOwners {
					break
				}
			}
		}

		if leafOnly && out.Len() > 0 {
			break
		}
		if d == baseDirConvention {
			break
		}
		d = filepath.Dir(d)
		d = canonicalize(d)
	}
	return out
}

// LeafApprovers returns a set of users who are the closest approvers to the
// requested file. If pkg/OWNERS has user1 and pkg/util/OWNERS has user2 this
// will only return user2 for the path pkg/util/sets/file.go
func (o *RepoOwnerInfo) LeafApprovers(path string) sets.String {
	return o.entriesForFile(path, true, func(c *Config) []string {
		return c.Approvers
	})
}

// Approvers returns ALL of the users who are approvers for the
// requested file (including approvers in parent dirs' OWNERS).
// If pkg/OWNERS has user1 and pkg/util/OWNERS has user2 this
// will return both user1 and user2 for the path pkg/util/sets/file.go
func (o *RepoOwnerInfo) Approvers(path string) sets.String {
	return o.entriesForFile(path, false, func(c *Config) []string {
		return c.Approvers
	})
}

// LeafReviewers returns a set of users who are the closest reviewers to the
// requested file. If pkg/OWNERS has user1 and pkg/util/OWNERS has user2 this
// will only return user2 for the path pkg/util/sets/file.go
func (o *RepoOwnerInfo) LeafReviewers(path string) sets.String {
	return o.entriesForFile(path, true, func(c *Config) []string {
		return c.Reviewers
	})
}

// Reviewers returns ALL of the users who are reviewers for the
// requested file (including reviewers in parent dirs' OWNERS).
// If pkg/OWNERS has user1 and pkg/util/OWNERS has user2 this
// will return both user1 and user2 for the path pkg/util/sets/file.go
func (o *RepoOwnerInfo) Reviewers(path string) sets.String {
	return o.entriesForFile(path, false, func(c *Config) []string {
		return c.Reviewers
	})
}

// RequiredReviewers returns ALL of the users who are required_reviewers for the
// requested file (including required_reviewers in parent dirs' OWNERS).
// If pkg/OWNERS has user1 and pkg/util/OWNERS has user2 this
// will return both user1 and user2 for the path pkg/util/sets/file.go
func (o *RepoOwnerInfo) RequiredReviewers(path string) sets.String {
	return o.entriesForFile(path, false, func(c *Config) []string {
		return c.RequiredReviewers
	})
}

func (o *RepoOwnerInfo) TopLevelApprovers() sets.String {
	return o.entriesForFile(".", false, func(c *Config) []string {
		return c.Approvers
	})
}

func (o *RepoOwnerInfo) AllReviewers() sets.String {
	r := sets.NewString()

	for _, v := range o.dirOwners {
		for _, v1 := range v {
			r.Insert(v1.Approvers...)
			r.Insert(v1.Reviewers...)
		}
	}

	for _, v := range o.fileOwners {
		for _, v1 := range v {
			r.Insert(v1.Approvers...)
			r.Insert(v1.Reviewers...)
		}
	}

	return r
}

// ParseSimpleConfig will unmarshal the content of the OWNERS file at the path into a SimpleConfig.
// Returns an error if the content cannot be unmarshalled.
func (o *RepoOwnerInfo) ParseSimpleConfig(path string) (SimpleConfig, error) {
	c := new(ownersConfig)
	err := parseYaml(path, c)
	return c.SimpleConfig, err
}

// ParseFullConfig will unmarshal the content of the OWNERS file at the path into a FullConfig.
// Returns an error if the content cannot be unmarshalled.
func (o *RepoOwnerInfo) ParseFullConfig(path string) (FullConfig, error) {
	c := new(ownersConfig)
	if err := parseYaml(path, c); err != nil {
		return FullConfig{}, err
	}

	return FullConfig{
		Filters: c.Files,
	}, nil
}

func loadOwners(baseDir string, mdYaml bool, aliases RepoAliases, dirBlacklist []*regexp.Regexp, log *logrus.Entry) (RepoOwner, error) {
	o := &RepoOwnerInfo{
		dirOwners:    make(map[string]map[*regexp.Regexp]SimpleConfig),
		fileOwners:   make(map[string]map[*regexp.Regexp]Config),
		baseDir:      baseDir,
		dirBlacklist: dirBlacklist,
		log:          log,
	}

	return o, filepath.Walk(o.baseDir, o.walkFunc)
}

func (o *RepoOwnerInfo) filterCollaborators(toKeep []github.User) RepoOwner {
	return o
}

func (o *RepoOwnerInfo) isEnableMDYAML() bool {
	return false
}
