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
- class_b
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
    reviewers:
    - go_b
`),

		"src/dir/workers/OWNERS": []byte(`
approvers:
- workers_a
reviewers:
- workers_b
files:
  "\\.go$":
    approvers:
    - go_a
`),

		"src/dir/doc/OWNERS": []byte(`
options:
  no_parent_owners: true
files:
  "\\.md$":
    approvers:
    - md_a
    reviewers:
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

func TestRepoOwnerInfo(t *testing.T) {
	testRepoOwnerInfo(localgit.New, t)
}

func TestRepoOwnerInfoV2(t *testing.T) {
	testRepoOwnerInfo(localgit.NewV2, t)
}

func testRepoOwnerInfo(clients localgit.Clients, t *testing.T) {
	for _, test := range genTestCaseOfRepoOwnerInfo() {
		t.Logf("Running scenario %q", test.name)
		client, cleanup, err := getTestClientWrapper(testRepo, false, true, false, false, nil, nil, nil, nil, clients)
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

		check := func(expected, got testCaseOfRepoOwnerInfo) {
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

		check(test, testCaseOfRepoOwnerInfo{
			topLevelApprovers:         ro.TopLevelApprovers(),
			expectedApprovers:         ro.Approvers(test.name),
			expectedReviewers:         ro.Reviewers(test.name),
			expectedLeafApprovers:     ro.LeafApprovers(test.name),
			expectedLeafReviewers:     ro.LeafReviewers(test.name),
			expectedApproverOwnerFile: ro.FindApproverOwnersForFile(test.name),
			expectedReviewerOwnerFile: ro.FindReviewersOwnersForFile(test.name),
		})
	}
}

type testCaseOfRepoOwnerInfo struct {
	name                      string
	path                      string
	topLevelApprovers         sets.String
	expectedApprovers         sets.String
	expectedReviewers         sets.String
	expectedLeafApprovers     sets.String
	expectedLeafReviewers     sets.String
	expectedApproverOwnerFile string
	expectedReviewerOwnerFile string
}

func genTestCaseOfRepoOwnerInfo() []testCaseOfRepoOwnerInfo {
	return []testCaseOfRepoOwnerInfo{
		{
			name:                      "root directory",
			path:                      "1.txt",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("root_a"),
			expectedReviewers:         sets.NewString("root_b"),
			expectedLeafApprovers:     sets.NewString("root_a"),
			expectedLeafReviewers:     sets.NewString("root_b"),
			expectedApproverOwnerFile: "",
			expectedReviewerOwnerFile: "",
		},

		{
			name:                      "src/OWNERS",
			path:                      "src/1.txt",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("root_a", "src_a"),
			expectedReviewers:         sets.NewString("root_b", "src_b"),
			expectedLeafApprovers:     sets.NewString("src_a"),
			expectedLeafReviewers:     sets.NewString("src_b"),
			expectedApproverOwnerFile: "src",
			expectedReviewerOwnerFile: "src",
		},

		{
			name:                      "tool/OWNERS",
			path:                      "tool/1.txt",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("root_a", "tool_a"),
			expectedReviewers:         sets.NewString("root_b"),
			expectedLeafApprovers:     sets.NewString("tool_a"),
			expectedLeafReviewers:     sets.NewString("root_b"),
			expectedApproverOwnerFile: "tool",
			expectedReviewerOwnerFile: "",
		},

		{
			name:                      "pkg/OWNERS",
			path:                      "pkg/1.txt",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("root_a"),
			expectedReviewers:         sets.NewString("root_b", "pkg_b"),
			expectedLeafApprovers:     sets.NewString("root_a"),
			expectedLeafReviewers:     sets.NewString("pkg_b"),
			expectedApproverOwnerFile: "",
			expectedReviewerOwnerFile: "pkg",
		},

		{
			name:                      "vendor",
			path:                      "vendor/1.txt",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("root_a"),
			expectedReviewers:         sets.NewString("root_b"),
			expectedLeafApprovers:     sets.NewString("root_a"),
			expectedLeafReviewers:     sets.NewString("root_b"),
			expectedApproverOwnerFile: "",
			expectedReviewerOwnerFile: "",
		},

		{
			name:                      "src/dir/OWNERS",
			path:                      "src/dir/1.txt",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("dir_a"),
			expectedReviewers:         sets.NewString("dir_b"),
			expectedLeafApprovers:     sets.NewString("dir_a"),
			expectedLeafReviewers:     sets.NewString("dir_b"),
			expectedApproverOwnerFile: "src/dir",
			expectedReviewerOwnerFile: "src/dir",
		},

		{
			name:                      "src/conformance/OWNERS",
			path:                      "src/conformance/1.txt",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("conformance_a"),
			expectedReviewers:         sets.NewString("root_b", "src_b"),
			expectedLeafApprovers:     sets.NewString("conformance_a"),
			expectedLeafReviewers:     sets.NewString("src_b"),
			expectedApproverOwnerFile: "src/conformance",
			expectedReviewerOwnerFile: "src",
		},

		{
			name:                      "src/class/OWNERS",
			path:                      "src/class/1.txt",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("root_a", "src_a"),
			expectedReviewers:         sets.NewString("class_b"),
			expectedLeafApprovers:     sets.NewString("src_a"),
			expectedLeafReviewers:     sets.NewString("class_b"),
			expectedApproverOwnerFile: "src",
			expectedReviewerOwnerFile: "src/class",
		},

		{
			name:                      "src/test/OWNERS",
			path:                      "src/test/1.txt",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("root_a", "src_a"),
			expectedReviewers:         sets.NewString("root_b", "src_b"),
			expectedLeafApprovers:     sets.NewString("src_a"),
			expectedLeafReviewers:     sets.NewString("src_b"),
			expectedApproverOwnerFile: "src",
			expectedReviewerOwnerFile: "src",
		},

		{
			name:                      "src/dir/subdir/1.txt",
			path:                      "src/dir/subdir/1.txt",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("dir_a", "subdir_a"),
			expectedReviewers:         sets.NewString("dir_b", "subdir_b"),
			expectedLeafApprovers:     sets.NewString("subdir_a"),
			expectedLeafReviewers:     sets.NewString("subdir_b"),
			expectedApproverOwnerFile: "src/dir/subdir",
			expectedReviewerOwnerFile: "src/dir/subdir",
		},

		{
			name:                      "src/dir/subdir/1.go",
			path:                      "src/dir/subdir/1.go",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("go_a"),
			expectedReviewers:         sets.NewString("go_b"),
			expectedLeafApprovers:     sets.NewString("go_a"),
			expectedLeafReviewers:     sets.NewString("go_b"),
			expectedApproverOwnerFile: "src/dir/subdir",
			expectedReviewerOwnerFile: "src/dir/subdir",
		},

		{
			name:                      "src/dir/subdir/agent/1.go",
			path:                      "src/dir/subdir/agent/1.go",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("dir_a", "subdir_a"),
			expectedReviewers:         sets.NewString("dir_b", "subdir_b"),
			expectedLeafApprovers:     sets.NewString("subdir_a"),
			expectedLeafReviewers:     sets.NewString("subdir_b"),
			expectedApproverOwnerFile: "src/dir/subdir",
			expectedReviewerOwnerFile: "src/dir/subdir",
		},

		{
			name:                      "src/dir/workers/1.go",
			path:                      "src/dir/workers/1.go",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("go_a"),
			expectedReviewers:         sets.NewString("dir_b", "workers_b"),
			expectedLeafApprovers:     sets.NewString("go_a"),
			expectedLeafReviewers:     sets.NewString("workers_b"),
			expectedApproverOwnerFile: "src/dir/workers",
			expectedReviewerOwnerFile: "src/dir/workers",
		},

		{
			name:                      "src/dir/doc/1.txt",
			path:                      "src/dir/doc/1.txt",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("dir_a"),
			expectedReviewers:         sets.NewString("dir_b"),
			expectedLeafApprovers:     sets.NewString("dir_a"),
			expectedLeafReviewers:     sets.NewString("dir_b"),
			expectedApproverOwnerFile: "src/dir",
			expectedReviewerOwnerFile: "src/dir",
		},

		{
			name:                      "src/dir/doc/1.md",
			path:                      "src/dir/doc/1.md",
			topLevelApprovers:         sets.NewString("root_a"),
			expectedApprovers:         sets.NewString("md_a"),
			expectedReviewers:         sets.NewString("md_b"),
			expectedLeafApprovers:     sets.NewString("md_a"),
			expectedLeafReviewers:     sets.NewString("md_b"),
			expectedApproverOwnerFile: "src/dir/doc",
			expectedReviewerOwnerFile: "src/dir/doc",
		},
	}
}
