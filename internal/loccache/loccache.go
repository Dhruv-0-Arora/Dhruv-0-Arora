// Package loccache maintains the per-repo lines-of-code cache and
// drives incremental refresh against the GitHub API.
//
// On-disk format (one repo per line, whitespace separated):
//
//	<sha256(nameWithOwner)> <total_commits> <my_commits> <added_loc> <deleted_loc>
//
// A repo is re-scanned only when its `total_commits` differs from the
// cached value. This is the same trick today.py uses to keep the daily
// cron under the GitHub anti-abuse limits.
package loccache

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Dhruv-0-Arora/Dhruv-0-Arora/internal/ghclient"
)

// Totals is the aggregate result returned by Refresh.
type Totals struct {
	Added       int
	Deleted     int
	Net         int  // Added - Deleted
	MyCommits   int  // sum of my_commits across all cached repos
	CacheHit    bool // true if no repo needed re-scanning
}

// Refresh loads the cache (creating it if missing), updates any repos
// whose commit count has changed, writes the cache back, and returns
// the aggregate totals.
//
// dir is the cache directory; the file is named <sha256(login)>.txt
// inside it.
func Refresh(
	ctx context.Context,
	cli *ghclient.Client,
	dir, login, userID string,
	repos []ghclient.RepoCommitCount,
) (Totals, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Totals{}, err
	}
	path := filepath.Join(dir, sha(login)+".txt")

	entries, err := load(path)
	if err != nil {
		return Totals{}, err
	}

	// Index existing entries by repo hash for O(1) lookup.
	byHash := make(map[string]*entry, len(entries))
	for i := range entries {
		byHash[entries[i].repoHash] = &entries[i]
	}

	// Rebuild in the order returned by GraphQL so the file stays stable.
	rebuilt := make([]entry, 0, len(repos))
	cacheHit := true

	for _, r := range repos {
		h := sha(r.NameWithOwner)
		e := entry{repoHash: h, totalCommits: r.TotalCommits}

		if prev, ok := byHash[h]; ok && prev.totalCommits == r.TotalCommits && r.TotalCommits > 0 {
			// Unchanged repo: reuse previous numbers.
			e.myCommits = prev.myCommits
			e.added = prev.added
			e.deleted = prev.deleted
		} else if r.TotalCommits == 0 {
			// Empty repo, nothing to count.
			cacheHit = cacheHit && (ok && prev.totalCommits == 0)
		} else {
			// Either new repo or commit count changed -> re-scan.
			cacheHit = false
			owner, name, ok := splitNameWithOwner(r.NameWithOwner)
			if !ok {
				return Totals{}, fmt.Errorf("malformed nameWithOwner: %q", r.NameWithOwner)
			}
			loc, err := cli.RepoLOCForUser(ctx, owner, name, userID)
			if err != nil {
				// Persist what we have so partial progress isn't lost,
				// then surface the error.
				_ = save(path, rebuilt)
				return Totals{}, fmt.Errorf("scan %s: %w", r.NameWithOwner, err)
			}
			e.myCommits = loc.MyCommits
			e.added = loc.Additions
			e.deleted = loc.Deletions
		}
		rebuilt = append(rebuilt, e)
	}

	if err := save(path, rebuilt); err != nil {
		return Totals{}, err
	}

	var t Totals
	t.CacheHit = cacheHit
	for _, e := range rebuilt {
		t.Added += e.added
		t.Deleted += e.deleted
		t.MyCommits += e.myCommits
	}
	t.Net = t.Added - t.Deleted
	return t, nil
}

type entry struct {
	repoHash     string
	totalCommits int
	myCommits    int
	added        int
	deleted      int
}

func load(path string) ([]entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 5 {
			return nil, fmt.Errorf("malformed cache line: %q", line)
		}
		e := entry{repoHash: parts[0]}
		var perr error
		if e.totalCommits, perr = strconv.Atoi(parts[1]); perr != nil {
			return nil, fmt.Errorf("totalCommits: %w", perr)
		}
		if e.myCommits, perr = strconv.Atoi(parts[2]); perr != nil {
			return nil, fmt.Errorf("myCommits: %w", perr)
		}
		if e.added, perr = strconv.Atoi(parts[3]); perr != nil {
			return nil, fmt.Errorf("added: %w", perr)
		}
		if e.deleted, perr = strconv.Atoi(parts[4]); perr != nil {
			return nil, fmt.Errorf("deleted: %w", perr)
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

func save(path string, entries []entry) error {
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "%s %d %d %d %d\n",
			e.repoHash, e.totalCommits, e.myCommits, e.added, e.deleted)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func splitNameWithOwner(s string) (owner, name string, ok bool) {
	i := strings.IndexByte(s, '/')
	if i < 0 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}
