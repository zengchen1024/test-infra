package repoowners

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/git/localgit"
)

var (
	testRepo = map[string][]byte{
		"OWNERS": []byte(`
approvers:
- root_a
reviewers:
- root_b
`),
		"src/OWNERS": []byte(`
approvers:
- src_a
reviewers:
- src_b
`),

		"tool/OWNERS": []byte(`
approvers:
- tool_a
`),

		"pkg/OWNERS": []byte(`
reviewers:
- pkg_b
`),

		"vendor": []byte(`
`),

		"src/dir/OWNERS": []byte(`
approvers:
- dir_a
reviewers:
- dir_b
options:
  no_parent_owners: true
`),

		"src/conformance/OWNERS": []byte(`
approvers:
- conformance_a
options:
  no_parent_owners: true
`),

		"src/class/OWNERS": []byte(`
reviewers:
- class_a
options:
  no_parent_owners: true
`),

		"src/test/OWNERS": []byte(`
options:
  no_parent_owners: true
`),

		"src/dir/subdir/OWNERS": []byte(`
approvers:
- subdir_a
reviewers:
- subdir_b
files:
  "\\.go$":
    approvers:
    - go_a
    reviers:
    - go_b
`),

		"src/dir/doc/OWNERS": []byte(`
options:
  no_parent_owners: true
files:
  "\\.md$":
    approvers:
    - md_a
    reviers:
    - md_b
`),
	}
)

func getTestClientWrapper(
	files map[string][]byte,
	enableMdYaml,
	skipCollab,
	includeAliases bool,
	ignorePreconfiguredDefaults bool,
	ownersDirBlacklistDefault []string,
	ownersDirBlacklistByRepo map[string][]string,
	extraBranchesAndFiles map[string]map[string][]byte,
	cacheOptions *cacheOptions,
	clients localgit.Clients,
) (*Client, func(), error) {
	c, f, err := getTestClient(
		files, enableMdYaml, skipCollab, includeAliases, ignorePreconfiguredDefaults,
		ownersDirBlacklistDefault, ownersDirBlacklistByRepo, extraBranchesAndFiles,
		cacheOptions, clients,
	)
	if err == nil {
		c.loadOwnersFunc = loadOwners
	}
	return c, f, err
}

func TestRepoOwnerInfo(clients localgit.Clients, t *testing.T) {
	type testCase struct {
		name                      string
		expectedApprovers         sets.String
		expectedReviewers         sets.String
		expectedLeafApprovers     sets.String
		expectedLeafReviewers     sets.String
		expectedApproverOwnerFile string
		expectedReviewerOwnerFile string
	}

	tests := []testCase{
		{
			name:              "top level approvers",
			expectedApprovers: sets.NewString("root_a"),
		},
	}

	for _, test := range tests {
		t.Logf("Running scenario %q", test.name)
		client, cleanup, err := getTestClientWrapper(testFiles, false, true, false, false, nil, nil, nil, nil, clients)
		if err != nil {
			t.Errorf("Error creating test client: %v.", err)
			continue
		}
		defer cleanup()

		r, err := client.LoadRepoOwners("org", "repo", "master")
		if err != nil {
			t.Errorf("Unexpected error loading RepoOwners: %v.", err)
			continue
		}
		ro := r.(*RepoOwnerInfo)
		if ro.baseDir == "" {
			t.Errorf("Expected 'baseDir' to be populated.")
			continue
		}

		check := func(expected, got testCase) {
			do := func(item string, v1, v2 sets.String) {
				if !v1.Equal(v2) {
					t.Errorf("Run test case:%s, Expected %s to be:\n%#v\ngot:\n%#v.", expected.name, item, v1, v2)
				}
			}
			do("approvers", expected.expectedApprovers, got.expectedApprovers)
			do("reviewers", expected.expectedReviewers, got.expectedReviewers)
			do("leaf approvers", expected.expectedLeafApprovers, got.expectedLeafApprovers)
			do("leaf reviewers", expected.expectedLeafReviewers, got.expectedLeafReviewers)
			do("apprver owner file", sets.NewString(expected.expectedApproverOwnerFile), sets.NewString(got.expectedApproverOwnerFile))
			do("reviewer owner file", sets.NewString(expected.expectedReviewerOwnerFile), sets.NewString(got.expectedReviewerOwnerFile))
		}

		check(test, testCase{
			expectedApprovers: ro.TopLevelApprovers(),
		})
	}
}
