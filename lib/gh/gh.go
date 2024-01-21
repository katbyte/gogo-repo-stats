package gh

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/katbyte/gogo-repo-stats/lib/clog"
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

type Project struct {
	Owner  string
	Number int
	Token
}

func NewProject(owner string, number int, token string) Project {
	p := Project{
		Owner:  owner,
		Number: number,
		Token: Token{
			Token: nil,
		},
	}

	if token != "" {
		p.Token.Token = &token
	}

	return p
}

func (t Token) NewClient() (*github.Client, context.Context) {
	ctx := context.Background()

	// use retryablehttp to handle rate limiting
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 7
	retryClient.Logger = clog.Log

	// github is.. special using 403 instead of 429 for rate limiting so we need to handle that here :(
	retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		if resp != nil && resp.StatusCode == 403 {
			// get x-rate-limit-reset header
			reset := resp.Header.Get("x-ratelimit-reset")
			if reset != "" {
				i, err := strconv.ParseInt(reset, 10, 64)
				if err == nil {
					utime := time.Unix(i, 0)
					wait := utime.Sub(time.Now()) + time.Minute // add an extra min to be safe
					clog.Log.Errorf("ratelimited, parsed x-ratelimit-reset, waiting for %s", wait.String())
					return wait
				}
				clog.Log.Errorf("unable to parse x-ratelimit-reset header: %s", err)
			}
		}

		return retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
	}
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp.StatusCode == 403 {
			return true, nil
		}

		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	if t := t.Token; t != nil {
		t := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *t},
		)
		retryClient.HTTPClient = oauth2.NewClient(ctx, t)
	}

	return github.NewClient(retryClient.StandardClient()), ctx
}
