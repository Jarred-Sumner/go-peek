package repo

//go:generate go-enum -f=$GOFILE

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"github.com/ucarion/urlpath"
)

var patterns = [8]string{
	"/:owner/:repo",
	"/:query",
	"/:owner/:repo/tree/:ref",
	"/:owner/:repo/tree/:ref/*",
	"/:owner/:repo/blob/:ref/*",
	"/:owner/:repo/pull/:pullRequestID/files",
	"/:owner/:repo/pull/:pullRequestID/*",
	"/:owner/:repo/*",
}

var (
	rootRepoRoute        = urlpath.New(patterns[0])
	searchRepoRoute      = urlpath.New(patterns[1])
	treeBaseRoute        = urlpath.New(patterns[2])
	fileRoute            = urlpath.New(patterns[3])
	blobFileRoute        = urlpath.New(patterns[4])
	pullRequestCodeRoute = urlpath.New(patterns[5])
	catchAllPR           = urlpath.New(patterns[6])
	catchAllRepoRoute    = urlpath.New(patterns[7])
)

/*
ENUM(
clone
root
file
pullRequest
search
)
*/
type Kind int

var _PathKindMap = [8]Kind{
	KindSearch,
	KindRoot,
	KindRoot,
	KindFile,
	KindFile,
	KindPullRequest,
	KindPullRequest,
	KindRoot,
}

type Query struct {
	Kind   Kind
	GitHub bool
	CDN    bool
	Src    string
	Params KindParams
	Exact  bool
}

func (q Query) Pretty() string {
	return fmt.Sprintf(`%v/%v@%v`, q.Params.owner, q.Params.repo, q.refToUse())
}

func (q Query) PrettyDirname() string {
	return fmt.Sprintf(`%v@%v-*`, q.Params.owner, q.Params.repo)
}

const defaultRef = "main"

func (q Query) fallbackRef() string {
	if q.Exact {
		return ""
	}

	if q.Params.ref == defaultRef {
		return "master"
	} else {
		return "main"
	}
}

func (q Query) refToUse() string {
	if q.Exact {
		return q.Params.ref
	}

	if q.Params.ref == "" {
		return defaultRef
	} else {
		return q.Params.owner
	}
}

type KindParams struct {
	owner         string
	repo          string
	ref           string
	source        string
	path          string
	pullRequestID string
}

func findMatchingURL(path string) (*urlpath.Match, *urlpath.Path, Kind) {
	routes := [8]urlpath.Path{
		rootRepoRoute,
		searchRepoRoute,
		treeBaseRoute,
		fileRoute,
		blobFileRoute,
		pullRequestCodeRoute,
		catchAllPR,
		catchAllRepoRoute,
	}

	for index, _route := range routes {
		result, _match := _route.Match(path)
		if _match {
			return &result, &_route, _PathKindMap[index]
		}
	}

	return nil, nil, KindClone
}

func GetQuery(_url string) Query {
	url := _url

	gitHubDomain := viper.GetString("github_domain")
	query := Query{}
	query.Params = KindParams{}

	if strings.HasPrefix(url, "git-peek://") {
		url = url[len("git-peek://"):]
	}

	if strings.HasPrefix(url, "https://") {
		url = url[len("https://"):]
	}

	query.CDN = true

	if strings.Contains(url, "?") {
		if strings.Contains(url, "noCDN") {
			query.CDN = false
		}

		url = strings.Split(url, "?")[0]
	}

	if strings.HasPrefix(url, gitHubDomain) {
		query.GitHub = true
		url = url[len(gitHubDomain):]
	}

	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}

	match, _, kind := findMatchingURL(url)

	query.Kind = kind
	query.Src = url

	if kind == KindClone {
		return query
	}

	if kind == KindSearch {
		query.Src = strings.ReplaceAll(url, "/", "")
		return query
	}

	query.Params.owner = match.Params["owner"]
	query.Params.repo = match.Params["repo"]

	if match.Params["ref"] != "" {
		query.Params.ref = match.Params["ref"]
		query.Exact = true
	}

	if kind == KindPullRequest {
		query.Params.pullRequestID = match.Params["pullRequestID"]
	}

	if kind == KindFile {
		query.Params.path = match.Trailing
	}

	return query
}
