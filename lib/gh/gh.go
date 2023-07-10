package gh

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v45/github"
	common "github.com/katbyte/gogo-repo-stats/lib/chttp"
	"github.com/katbyte/gogo-repo-stats/lib/pointer"
	"golang.org/x/oauth2"
)

type Token struct {
	Token *string
}

type Repo struct {
	Owner string
	Name  string
	Token
}

func NewRepo(repo, token string) (*Repo, error) {
	parts := strings.Split(repo, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format, expected owner/name got %q", repo)
	}

	return pointer.To(NewRepoOwnerName(parts[0], parts[1], token)), nil
}

func NewRepoOwnerName(owner, name, token string) Repo {
	r := Repo{
		Owner: owner,
		Name:  name,
		Token: Token{
			Token: nil,
		},
	}

	if token != "" {
		r.Token.Token = &token
	}

	return r
}

// TODO retry on throttle
// check out go-retryablehttp ? roll own?

func (t Token) NewClient() (*github.Client, context.Context) {
	ctx := context.Background()
	httpClient := common.NewHTTPClient("GitHub")

	if t := t.Token; t != nil {
		t := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *t},
		)
		httpClient = oauth2.NewClient(ctx, t)
	}

	httpClient.Transport = common.NewTransport("GitHub", httpClient.Transport)

	return github.NewClient(httpClient), ctx
}
