package repo

import (
	"context"
	"net/http"

	"io"
	"net/url"
	"path"

	"github.com/mholt/archiver/v3"

	"gitPeek/src/peek"

	"github.com/google/go-github/v33/github"
)

type TarballLink struct {
	url     string
	success bool
	error   error
}

var tarballLinkChannel = make(chan TarballLink)

func fetchTarballURL(ctx context.Context, owner string, repo string, ref string) {
	defer close(tarballLinkChannel)
	repos := peek.GithubClient.Repositories
	var url *url.URL
	var _error error
	if ref == "" {
		url, _, _error = repos.GetArchiveLink(ctx, owner, repo, "tarball", nil, true)
	} else {
		opts := &github.RepositoryContentGetOptions{
			Ref: ref,
		}
		url, _, _error = repos.GetArchiveLink(ctx, owner, repo, "tarball", opts, true)
	}

	if _error == nil {
		tarballLinkChannel <- TarballLink{url: url.String(), success: true, error: _error}
	} else {
		tarballLinkChannel <- TarballLink{url: "", success: false, error: _error}
	}

}

func fetchTarballLink(query Query) TarballLink {
	owner := query.Params.owner
	repo := query.Params.repo

	// if query.exactRef {
	ctx := context.Background()
	go fetchTarballURL(ctx, owner, repo, query.refToUse())
	return <-tarballLinkChannel

	// } else {

	// parent := context.Background()
	// primaryCtx, cancelPrimary := context.WithTimeout(parent, time.Second*30)
	// secondaryContext, cancelSecondary := context.WithTimeout(parent, time.Second*30)
	// primaryRef := &github.RepositoryContentGetOptions{
	// 	Ref: query.refToUse(),
	// }

	// secondaryRef := &github.RepositoryContentGetOptions{
	// 	Ref: query.fallbackRef(),
	// }

	// repositories.GetArchiveLink(secondaryContext, owner, repo, github.Tarball, secondaryRef, true)

	// }

}

type CloneOperation struct {
	isFinished bool
	hadErrors  bool
}

func tarballClone(query Query, destination string) CloneOperation {
	link := fetchTarballLink(query)
	if !link.success {
		peek.Error(`Invalid repository link\nTried:\n- %v`, query.Pretty())
		return CloneOperation{hadErrors: true, isFinished: false}
	}

	peek.LogV("Fetched tarball link %v", link.url)

	client := http.Client{}
	ctx := context.Background()

	request, err := http.NewRequestWithContext(ctx, "GET", link.url, nil)

	if err != nil {
		peek.Error(`Failed to create request %v. Error:\n%v`, query.Pretty(), err)
		return CloneOperation{hadErrors: true, isFinished: false}
	}

	if peek.GithubToken != "" {
		request.Header.Add("Authorization", peek.GithubToken)
	}

	peek.LogV("Start request %v", link.url)
	response, err := client.Do(request)

	if err != nil {
		peek.Error(`Failed to fetch %v. Error:\n%v`, query.Pretty(), err)
		return CloneOperation{hadErrors: true, isFinished: false}
	}
	defer response.Body.Close()

	peek.Log("Extracting %v...", query.Pretty())
	tar := archiver.NewTarGz()

	tar.Open(response.Body, 0)

	defer tar.Close()

	repo := query.Params.repo

	// if the target ends up being a directory, then
	// we will continue walking and extracting files
	// until we are no longer within that directory
	var targetDirPath string
	targetDirPath = destination

	tar.StripComponents = 1

	var hadErrors bool = false

	var out string
	for {
		file, err := tar.Read()
		if err == io.EOF {
			break
		}

		name := path.Clean(file.Name())

		if file.IsDir() && repo == name {
			targetDirPath = path.Join(destination, name)
			out = targetDirPath
		} else {
			out = path.Join(targetDirPath, name)
		}

		if file.IsDir() {
			err = mkdir(out, file.Mode())
			if err != nil {
				peek.Error("Error mkdir at %v", out)
				hadErrors = true
			}
		} else {
			err = writeNewFile(out, file.ReadCloser, file.Mode())
			if err != nil {
				peek.Error("Error writing %v", out)
				hadErrors = true
			}
		}

		// if err != nil {
		// 	if t.ContinueOnError || IsIllegalPathError(err) {
		// 		log.Printf("[ERROR] Reading file in tar archive: %v", err)
		// 		continue
		// 	}
		// 	return fmt.Errorf("reading file in tar archive: %v", err)
		// }
	}

	if hadErrors {
		peek.Line("Finished extracting with errors")
	} else {
		peek.Line("Finished extracting!")
	}
	return CloneOperation{hadErrors: hadErrors, isFinished: true}

}

func TarballClone(query Query, destination string) CloneOperation {
	result := make(chan CloneOperation)

	go func(query Query, destination string) {
		result <- tarballClone(query, destination)
		close(result)
	}(query, destination)

	return <-result
}
