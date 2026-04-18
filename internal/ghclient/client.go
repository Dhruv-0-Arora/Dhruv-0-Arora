// Package ghclient is a tiny stdlib-only client for the small set of
// GitHub GraphQL queries this project needs.
//
// All queries are POSTed to https://api.github.com/graphql with a
// bearer token. Responses are decoded into shaped structs rather than
// generic maps so the call sites stay short and the type errors loud.
package ghclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const endpoint = "https://api.github.com/graphql"

// Client wraps the auth and provides one method per query.
// QueryCount tracks how many requests were made (debug parity with
// today.py's QUERY_COUNT dict).
type Client struct {
	HTTP       *http.Client
	Token      string
	UserAgent  string
	QueryCount map[string]int
}

func New(token string) *Client {
	return &Client{
		HTTP:       &http.Client{Timeout: 30 * time.Second},
		Token:      token,
		UserAgent:  "Dhruv-0-Arora-readme-generator",
		QueryCount: map[string]int{},
	}
}

func (c *Client) post(ctx context.Context, name, query string, vars map[string]any, out any) error {
	c.QueryCount[name]++
	body, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d: %s", name, resp.StatusCode, string(raw))
	}

	// Parse into a shape that captures both data and any GraphQL errors.
	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("%s: decode envelope: %w (body=%s)", name, err, string(raw))
	}
	if len(env.Errors) > 0 {
		msgs := ""
		for i, e := range env.Errors {
			if i > 0 {
				msgs += "; "
			}
			msgs += e.Message
		}
		return fmt.Errorf("%s: graphql errors: %s", name, msgs)
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("%s: decode data: %w", name, err)
	}
	return nil
}

// User holds the bits we care about from the user query.
type User struct {
	ID        string
	CreatedAt time.Time
}

// GetUser returns id + createdAt. Used to (a) format the account
// creation date for stats and (b) filter commit authorship in the
// LOC walker.
func (c *Client) GetUser(ctx context.Context, login string) (User, error) {
	const q = `query($login:String!){ user(login:$login){ id createdAt } }`
	var resp struct {
		User struct {
			ID        string    `json:"id"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"user"`
	}
	if err := c.post(ctx, "GetUser", q, map[string]any{"login": login}, &resp); err != nil {
		return User{}, err
	}
	return User{ID: resp.User.ID, CreatedAt: resp.User.CreatedAt}, nil
}

// Followers returns the user's follower count.
func (c *Client) Followers(ctx context.Context, login string) (int, error) {
	const q = `query($login:String!){ user(login:$login){ followers{ totalCount } } }`
	var resp struct {
		User struct {
			Followers struct {
				TotalCount int `json:"totalCount"`
			} `json:"followers"`
		} `json:"user"`
	}
	if err := c.post(ctx, "Followers", q, map[string]any{"login": login}, &resp); err != nil {
		return 0, err
	}
	return resp.User.Followers.TotalCount, nil
}

// RepoStarsPage is one page of repos with their star counts.
type RepoStarsPage struct {
	TotalCount int
	Repos      []RepoStars
	EndCursor  string
	HasNext    bool
}

type RepoStars struct {
	NameWithOwner string
	Stars         int
}

// ReposAndStars walks all pages and returns the total repo count and
// the sum of stargazers across every returned repo.
//
// affiliations is a list like ["OWNER"] or ["OWNER","COLLABORATOR","ORGANIZATION_MEMBER"].
func (c *Client) ReposAndStars(ctx context.Context, login string, affiliations []string) (totalRepos, totalStars int, err error) {
	const q = `
query($aff:[RepositoryAffiliation], $login:String!, $cursor:String){
  user(login:$login){
    repositories(first:100, after:$cursor, ownerAffiliations:$aff){
      totalCount
      edges{ node{ ... on Repository { nameWithOwner stargazers{ totalCount } } } }
      pageInfo{ endCursor hasNextPage }
    }
  }
}`
	var cursor *string
	for {
		var resp struct {
			User struct {
				Repositories struct {
					TotalCount int `json:"totalCount"`
					Edges      []struct {
						Node struct {
							NameWithOwner string `json:"nameWithOwner"`
							Stargazers    struct {
								TotalCount int `json:"totalCount"`
							} `json:"stargazers"`
						} `json:"node"`
					} `json:"edges"`
					PageInfo struct {
						EndCursor   string `json:"endCursor"`
						HasNextPage bool   `json:"hasNextPage"`
					} `json:"pageInfo"`
				} `json:"repositories"`
			} `json:"user"`
		}
		vars := map[string]any{"aff": affiliations, "login": login, "cursor": cursor}
		if err := c.post(ctx, "ReposAndStars", q, vars, &resp); err != nil {
			return 0, 0, err
		}
		totalRepos = resp.User.Repositories.TotalCount
		for _, e := range resp.User.Repositories.Edges {
			totalStars += e.Node.Stargazers.TotalCount
		}
		if !resp.User.Repositories.PageInfo.HasNextPage {
			return totalRepos, totalStars, nil
		}
		c2 := resp.User.Repositories.PageInfo.EndCursor
		cursor = &c2
	}
}

// RepoCommitCount is one entry returned by ListRepoCommitCounts.
type RepoCommitCount struct {
	NameWithOwner string
	TotalCommits  int // 0 if the repo has no default branch (empty repo)
}

// ListRepoCommitCounts returns every accessible repo plus the total
// commit count on its default branch. Pagination is 60/page to
// sidestep the 502 timeouts that today.py documents.
func (c *Client) ListRepoCommitCounts(ctx context.Context, login string, affiliations []string) ([]RepoCommitCount, error) {
	const q = `
query($aff:[RepositoryAffiliation], $login:String!, $cursor:String){
  user(login:$login){
    repositories(first:60, after:$cursor, ownerAffiliations:$aff){
      edges{ node{ ... on Repository {
        nameWithOwner
        defaultBranchRef{ target{ ... on Commit { history{ totalCount } } } }
      } } }
      pageInfo{ endCursor hasNextPage }
    }
  }
}`
	var out []RepoCommitCount
	var cursor *string
	for {
		var resp struct {
			User struct {
				Repositories struct {
					Edges []struct {
						Node struct {
							NameWithOwner    string `json:"nameWithOwner"`
							DefaultBranchRef *struct {
								Target struct {
									History struct {
										TotalCount int `json:"totalCount"`
									} `json:"history"`
								} `json:"target"`
							} `json:"defaultBranchRef"`
						} `json:"node"`
					} `json:"edges"`
					PageInfo struct {
						EndCursor   string `json:"endCursor"`
						HasNextPage bool   `json:"hasNextPage"`
					} `json:"pageInfo"`
				} `json:"repositories"`
			} `json:"user"`
		}
		vars := map[string]any{"aff": affiliations, "login": login, "cursor": cursor}
		if err := c.post(ctx, "ListRepoCommitCounts", q, vars, &resp); err != nil {
			return nil, err
		}
		for _, e := range resp.User.Repositories.Edges {
			r := RepoCommitCount{NameWithOwner: e.Node.NameWithOwner}
			if e.Node.DefaultBranchRef != nil {
				r.TotalCommits = e.Node.DefaultBranchRef.Target.History.TotalCount
			}
			out = append(out, r)
		}
		if !resp.User.Repositories.PageInfo.HasNextPage {
			return out, nil
		}
		c2 := resp.User.Repositories.PageInfo.EndCursor
		cursor = &c2
	}
}

// RepoLOC is the LOC contribution attributable to a specific user.
type RepoLOC struct {
	MyCommits int
	Additions int
	Deletions int
}

// RepoLOCForUser walks every commit on the default branch (paginated
// 100 at a time) and sums additions/deletions/commits where the
// commit author's user.id matches userID.
//
// Returns a zero RepoLOC if the repo has no default branch.
func (c *Client) RepoLOCForUser(ctx context.Context, owner, repo, userID string) (RepoLOC, error) {
	const q = `
query($owner:String!, $repo:String!, $cursor:String){
  repository(owner:$owner, name:$repo){
    defaultBranchRef{ target{ ... on Commit {
      history(first:100, after:$cursor){
        edges{ node{ ... on Commit {
          additions deletions
          author{ user{ id } }
        } } }
        pageInfo{ endCursor hasNextPage }
      }
    } } }
  }
}`
	var out RepoLOC
	var cursor *string
	for {
		var resp struct {
			Repository struct {
				DefaultBranchRef *struct {
					Target struct {
						History struct {
							Edges []struct {
								Node struct {
									Additions int `json:"additions"`
									Deletions int `json:"deletions"`
									Author    struct {
										User *struct {
											ID string `json:"id"`
										} `json:"user"`
									} `json:"author"`
								} `json:"node"`
							} `json:"edges"`
							PageInfo struct {
								EndCursor   string `json:"endCursor"`
								HasNextPage bool   `json:"hasNextPage"`
							} `json:"pageInfo"`
						} `json:"history"`
					} `json:"target"`
				} `json:"defaultBranchRef"`
			} `json:"repository"`
		}
		vars := map[string]any{"owner": owner, "repo": repo, "cursor": cursor}
		if err := c.post(ctx, "RepoLOCForUser", q, vars, &resp); err != nil {
			return RepoLOC{}, err
		}
		if resp.Repository.DefaultBranchRef == nil {
			return RepoLOC{}, nil
		}
		hist := resp.Repository.DefaultBranchRef.Target.History
		for _, e := range hist.Edges {
			if e.Node.Author.User != nil && e.Node.Author.User.ID == userID {
				out.MyCommits++
				out.Additions += e.Node.Additions
				out.Deletions += e.Node.Deletions
			}
		}
		if !hist.PageInfo.HasNextPage {
			return out, nil
		}
		c2 := hist.PageInfo.EndCursor
		cursor = &c2
	}
}
