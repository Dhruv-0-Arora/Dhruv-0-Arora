<a href="https://github.com/Dhruv-0-Arora/Dhruv-0-Arora">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/Dhruv-0-Arora/Dhruv-0-Arora/main/dark_mode.svg">
    <img alt="Dhruv Arora's GitHub Profile README" src="https://raw.githubusercontent.com/Dhruv-0-Arora/Dhruv-0-Arora/main/light_mode.svg">
  </picture>
</a>

---

## How this README works

A small Go program (`cmd/today`) runs daily on GitHub Actions, queries the
GitHub GraphQL API for fresh stats (repos, stars, commits, lines of code,
followers, age), and rewrites the dynamic `<tspan>`s in `dark_mode.svg` and
`light_mode.svg` in place. The face is an ASCII portrait baked into both SVGs
from `ASCII_art.txt`.

### Layout

```
.
├── ASCII_art.txt              # source of truth for the face
├── dark_mode.svg              # generated template (face baked, stats id'd)
├── light_mode.svg             # same, light palette
├── cache/<sha256(login)>.txt  # per-repo LOC cache
├── cmd/
│   ├── build-templates/       # regenerates the two SVGs from ASCII_art.txt
│   └── today/                 # the daily updater
├── internal/
│   ├── age/        # calendar diff (matches dateutil.relativedelta)
│   ├── face/       # loads & cleans ASCII_art.txt
│   ├── ghclient/   # stdlib-only GraphQL client
│   ├── loccache/   # incremental LOC cache
│   ├── profile/    # static personal info
│   └── svg/        # template builder + in-place rewriter
└── .github/workflows/build.yaml
```

### Local development

```bash
go test ./...
go run ./cmd/build-templates           # regen SVGs after editing the face / profile
ACCESS_TOKEN=ghp_xxx USER_NAME=Dhruv-0-Arora go run ./cmd/today
```

### Required secrets (in repo settings)

| Name           | Value                                                     |
|----------------|-----------------------------------------------------------|
| `ACCESS_TOKEN` | A GitHub PAT with `read:user`, `repo`, `read:org` scopes. |
| `USER_NAME`    | Your GitHub login (e.g. `Dhruv-0-Arora`).                 |

Inspired by [Andrew6rant/Andrew6rant](https://github.com/Andrew6rant/Andrew6rant).
