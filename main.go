package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-github/v79/github"
	"golang.org/x/oauth2"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	user := os.Getenv("GITHUB_USER")
	if token == "" || user == "" {
		fmt.Println("Set GITHUB_TOKEN and GITHUB_USER environment variables")
		return
	}

	baseDir := "./github_zips"
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		fmt.Printf("Cannot create base dir: %v\n", err)
		return
	}

	ctx := context.Background()
	client := createGithubClient(ctx, token)

	repos, err := listRepositories(ctx, client, user)
	if err != nil {
		fmt.Printf("Failed to list repositories: %v\n", err)
		return
	}

	for _, repo := range repos {
		fmt.Println("Processing repo:", repo)
		processRepository(ctx, client, user, repo, baseDir)
	}
}

// createGithubClient sets up an authenticated GitHub client
func createGithubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

// listRepositories fetches all repos for a user
func listRepositories(ctx context.Context, client *github.Client, user string) ([]string, error) {
	var all []string
	opt := &github.RepositoryListOptions{ListOptions: github.ListOptions{PerPage: 50}}

	for {
		repos, resp, err := client.Repositories.List(ctx, user, opt)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			all = append(all, r.GetName())
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return all, nil
}

// processRepository downloads branches and tags for a single repo
func processRepository(ctx context.Context, client *github.Client, owner, repo, baseDir string) {
	repoDir := filepath.Join(baseDir, repo)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		fmt.Printf("Cannot create repo dir: %v\n", err)
		return
	}

	// Branches
	branchDir := filepath.Join(repoDir, "branches")
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		fmt.Printf("Cannot create branch dir: %v\n", err)
		return
	}
	branches, err := listBranches(ctx, client, owner, repo)
	if err == nil {
		for _, br := range branches {
			out := filepath.Join(branchDir, fmt.Sprintf("%s-branch-%s.zip", repo, br))
			downloadArchive(ctx, client, owner, repo, "heads/"+br, out)
		}
	}

	// Tags
	tagDir := filepath.Join(repoDir, "tags")
	if err := os.MkdirAll(tagDir, 0755); err != nil {
		fmt.Printf("Cannot create tag dir: %v\n", err)
		return
	}
	tags, err := listTags(ctx, client, owner, repo)
	if err == nil {
		for _, tg := range tags {
			out := filepath.Join(tagDir, fmt.Sprintf("%s-tag-%s.zip", repo, tg))
			downloadArchive(ctx, client, owner, repo, "tags/"+tg, out)
		}
	}
}

// listBranches fetches all branch names for a repo
func listBranches(ctx context.Context, client *github.Client, owner, repo string) ([]string, error) {
	var all []string
	opt := &github.BranchListOptions{ListOptions: github.ListOptions{PerPage: 50}}

	for {
		branches, resp, err := client.Repositories.ListBranches(ctx, owner, repo, opt)
		if err != nil {
			return nil, err
		}
		for _, br := range branches {
			all = append(all, br.GetName())
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return all, nil
}

// listTags fetches all tag names for a repo
func listTags(ctx context.Context, client *github.Client, owner, repo string) ([]string, error) {
	var all []string
	opt := &github.ListOptions{PerPage: 50}

	for {
		tags, resp, err := client.Repositories.ListTags(ctx, owner, repo, opt)
		if err != nil {
			return nil, err
		}
		for _, tg := range tags {
			all = append(all, tg.GetName())
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return all, nil
}

// downloadArchive fetches a ZIP for a branch or tag
func downloadArchive(ctx context.Context, client *github.Client, owner, repo, ref, out string) {
	url, _, err := client.Repositories.GetArchiveLink(ctx, owner, repo, github.Zipball, &github.RepositoryContentGetOptions{Ref: ref}, 0)
	if err != nil {
		fmt.Printf("GetArchiveLink failed %s/%s %s: %v\n", owner, repo, ref, err)
		return
	}

	fmt.Printf("Downloading %s …\n", out)
	resp, err := http.Get(url.String())
	if err != nil {
		fmt.Printf("HTTP download error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	f, err := os.Create(out)
	if err != nil {
		fmt.Printf("File create error: %v\n", err)
		return
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		fmt.Printf("Write error: %v\n", err)
		return
	}
	fmt.Printf("Saved %s\n", out)
}
