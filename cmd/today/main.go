// today updates dark_mode.svg and light_mode.svg with fresh stats
// pulled from the GitHub GraphQL API.
//
// Required env vars:
//
//	ACCESS_TOKEN  GitHub PAT with read:user, repo, read:org
//	USER_NAME     login (e.g. "Dhruv-0-Arora") -- optional, falls back to profile.Login
//
// On success, the two SVGs are written in place. Per-repo LOC is
// cached in cache/<sha256(login)>.txt so reruns only re-scan repos
// whose commit count changed.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Dhruv-0-Arora/Dhruv-0-Arora/internal/age"
	"github.com/Dhruv-0-Arora/Dhruv-0-Arora/internal/ghclient"
	"github.com/Dhruv-0-Arora/Dhruv-0-Arora/internal/loccache"
	"github.com/Dhruv-0-Arora/Dhruv-0-Arora/internal/profile"
	"github.com/Dhruv-0-Arora/Dhruv-0-Arora/internal/svg"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "today: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	token := os.Getenv("ACCESS_TOKEN")
	if token == "" {
		return fmt.Errorf("ACCESS_TOKEN env var is required")
	}
	login := os.Getenv("USER_NAME")
	if login == "" {
		login = profile.Login
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	cli := ghclient.New(token)
	start := time.Now()

	// 1. Identity (needed to attribute commits in LOC walker).
	user, err := timed("user", func() (ghclient.User, error) { return cli.GetUser(ctx, login) })
	if err != nil {
		return err
	}

	// 2. Age, follower count, owned repo+star totals, contrib repo total.
	ageStr := age.Since(profile.Birthday, time.Now().UTC()).String()

	followers, err := timed("followers", func() (int, error) { return cli.Followers(ctx, login) })
	if err != nil {
		return err
	}
	ownedRepos, stars, err := timed2("owned repos+stars", func() (int, int, error) {
		return cli.ReposAndStars(ctx, login, []string{"OWNER"})
	})
	if err != nil {
		return err
	}
	contribRepos, _, err := timed2("contributed repos", func() (int, int, error) {
		return cli.ReposAndStars(ctx, login,
			[]string{"OWNER", "COLLABORATOR", "ORGANIZATION_MEMBER"})
	})
	if err != nil {
		return err
	}

	// 3. LOC pipeline. List every repo's commit-count, then refresh
	//    only those whose count changed since the last cache write.
	repos, err := timed("repo list (commit counts)", func() ([]ghclient.RepoCommitCount, error) {
		return cli.ListRepoCommitCounts(ctx, login,
			[]string{"OWNER", "COLLABORATOR", "ORGANIZATION_MEMBER"})
	})
	if err != nil {
		return err
	}
	totals, err := timed("loc cache refresh", func() (loccache.Totals, error) {
		return loccache.Refresh(ctx, cli, "cache", login, user.ID, repos)
	})
	if err != nil {
		return err
	}

	// 4. Rewrite both SVGs.
	stats := svg.Stats{
		Age:       ageStr,
		Repos:     ownedRepos,
		Contrib:   contribRepos,
		Stars:     stars,
		Commits:   totals.MyCommits,
		Followers: followers,
		LOCAdd:    totals.Added,
		LOCDel:    totals.Deleted,
	}
	for _, path := range []string{"dark_mode.svg", "light_mode.svg"} {
		if err := svg.Rewrite(path, stats); err != nil {
			return fmt.Errorf("rewrite %s: %w", path, err)
		}
	}

	cacheNote := "(cache miss)"
	if totals.CacheHit {
		cacheNote = "(cache hit)"
	}
	fmt.Printf("done in %s %s\n", time.Since(start).Round(time.Millisecond), cacheNote)
	fmt.Printf("query counts: %v\n", cli.QueryCount)
	return nil
}

// timed runs fn and prints its duration.
func timed[T any](label string, fn func() (T, error)) (T, error) {
	start := time.Now()
	v, err := fn()
	fmt.Printf("  %-28s %8s\n", label+":", time.Since(start).Round(time.Millisecond))
	return v, err
}

// timed2 is the two-result variant of timed.
func timed2[A, B any](label string, fn func() (A, B, error)) (A, B, error) {
	start := time.Now()
	a, b, err := fn()
	fmt.Printf("  %-28s %8s\n", label+":", time.Since(start).Round(time.Millisecond))
	return a, b, err
}
