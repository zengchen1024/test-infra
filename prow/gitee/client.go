package gitee

import (
	"context"
	"fmt"
	"sync"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/antihax/optional"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/github"
)

var _ Client = (*client)(nil)

type client struct {
	ac *sdk.APIClient

	mut      sync.Mutex // protects botName and email
	userData *sdk.User
}

func NewClient(getToken func() []byte) Client {
	token := string(getToken())

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	conf := sdk.NewConfiguration()
	conf.HTTPClient = oauth2.NewClient(context.Background(), ts)

	c := sdk.NewAPIClient(conf)
	return &client{ac: c}
}

// BotName returns the login of the authenticated identity.
func (c *client) BotName() (string, error) {
	err := c.getUserData()
	if err != nil {
		return "", err
	}

	return c.userData.Login, nil
}

func (c *client) Email() (string, error) {
	err := c.getUserData()
	if err != nil {
		return "", err
	}

	return c.userData.Email, nil
}

func (c *client) BotUser() (*github.User, error) {
	err := c.getUserData()
	if err != nil {
		return nil, err
	}

	d := c.userData
	u := github.User{
		Login: d.Login,
		Name:  d.Name,
		Email: d.Email,
		ID:    int(d.Id),
	}

	return &u, nil
}

func (c *client) CreatePullRequest(org, repo, title, body, head, base string, canModify bool) (sdk.PullRequest, error) {
	opts := sdk.CreatePullRequestParam{
		Title:             title,
		Head:              head,
		Base:              base,
		Body:              body,
		PruneSourceBranch: true,
	}

	pr, _, err := c.ac.PullRequestsApi.PostV5ReposOwnerRepoPulls(
		context.Background(), org, repo, opts)

	return pr, err
}

func (c *client) GetPullRequests(org, repo, state, head, base string) ([]sdk.PullRequest, error) {
	opts := sdk.GetV5ReposOwnerRepoPullsOpts{
		State: optional.NewString(state),
		Head:  optional.NewString(head),
		Base:  optional.NewString(base),
	}
	prs, _, err := c.ac.PullRequestsApi.GetV5ReposOwnerRepoPulls(context.Background(), org, repo, &opts)
	return prs, err
}

func (c *client) UpdatePullRequest(org, repo string, number int32, title, body, state, labels string) (sdk.PullRequest, error) {
	opts := sdk.PullRequestUpdateParam{
		Title:  title,
		Body:   body,
		State:  state,
		Labels: labels,
	}
	pr, _, err := c.ac.PullRequestsApi.PatchV5ReposOwnerRepoPullsNumber(context.Background(), org, repo, number, opts)
	return pr, err
}

func (c *client) getUserData() error {
	if c.userData == nil {
		c.mut.Lock()
		defer c.mut.Unlock()

		if c.userData == nil {
			u, _, err := c.ac.UsersApi.GetV5User(context.Background(), nil)
			if err != nil {
				return fmt.Errorf("fetching bot name from Gitee: %v", err)
			}
			c.userData = &u
		}
	}
	return nil
}

func (c *client) ListCollaborators(org, repo string) ([]github.User, error) {
	return nil, nil
}

func (c *client) GetRef(org, repo, ref string) (string, error) {
	return "", nil
}

func (c *client) GetPullRequest(org, repo string, number int) (*github.PullRequest, error) {
	return nil, nil
}

func (c *client) GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error) {
	return nil, nil
}

func (c *client) GetIssueLabels(org, repo string, number int) ([]github.Label, error) {
	return nil, nil
}

func (c *client) ListIssueComments(org, repo string, number int) ([]github.IssueComment, error) {
	return nil, nil
}

func (c *client) ListReviews(org, repo string, number int) ([]github.Review, error) {
	return nil, nil
}

func (c *client) ListPullRequestComments(org, repo string, number int) ([]github.ReviewComment, error) {
	return nil, nil
}

func (c *client) DeleteComment(org, repo string, ID int) error {
	return nil
}

func (c *client) CreateComment(org, repo string, number int, comment string) error {
	return nil
}

func (c *client) AddLabel(org, repo string, number int, label string) error {
	return nil
}

func (c *client) RemoveLabel(org, repo string, number int, label string) error {
	return nil
}

func (c *client) ListIssueEvents(org, repo string, num int) ([]github.ListedIssueEvent, error) {
	return nil, nil
}
