package gh

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/google/go-github/v45/github"
	"github.com/katbyte/gogo-repo-stats/lib/clog"
)

func (r Repo) IssueURL(pr int) string {
	return "https://github.com/" + r.Owner + "/" + r.Name + "/issues/" + strconv.Itoa(pr)
}

func (r Repo) ListAllIssues(state string, cb func([]*github.Issue, *github.Response) error) error {
	client, ctx := r.NewClient()

	opts := &github.IssueListByRepoOptions{
		State: state,
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	for {
		clog.Log.Debugf("Listing all issues for %s/%s (Page %d)...", r.Owner, r.Name, opts.ListOptions.Page)
		prs, resp, err := client.Issues.ListByRepo(ctx, r.Owner, r.Name, opts)
		if err != nil {

			return fmt.Errorf("unable to list issues for %s/%s (Page %d): %w", r.Owner, r.Name, opts.ListOptions.Page, err)

		}

		if err = cb(prs, resp); err != nil {
			return fmt.Errorf("callback failed for %s/%s (Page %d): %w", r.Owner, r.Name, opts.ListOptions.Page, err)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil
}

func (r Repo) GetAllIssues(state string) (*[]github.Issue, error) {
	var allIssues []github.Issue

	err := r.ListAllIssues(state, func(issues []*github.Issue, resp *github.Response) error {
		for i, p := range issues {
			if p.IsPullRequest() {
				clog.Log.Debugf("issues[%d] is a PR, skipping", i)
				continue
			}

			if p == nil {
				clog.Log.Debugf("issues[%d] was nil, skipping", i)
				continue
			}

			n := p.GetNumber()
			if n == 0 {
				clog.Log.Debugf("issues[%d].Number was nil/0, skipping", i)
				continue
			}

			allIssues = append(allIssues, *p)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get all prs for %s/%s: %w", r.Owner, r.Name, err)
	}

	sort.Slice(allIssues, func(i, j int) bool {
		return allIssues[i].GetNumber() < allIssues[j].GetNumber()
	})

	return &allIssues, nil
}
