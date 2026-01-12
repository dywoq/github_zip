package github

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

type Client struct {
	client *github.Client
}

func NewClient(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return &Client{
		client: github.NewClient(tc),
	}
}

func (c *Client) ListRepositories(ctx context.Context, user string) ([]string, error) {
	var all []string
	opt := &github.RepositoryListOptions{ListOptions: github.ListOptions{PerPage: 50}}

	for {
		repos, resp, err := c.client.Repositories.List(ctx, user, opt)
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

func (c *Client) ListBranches(ctx context.Context, owner, repo string) ([]string, error) {
	var all []string
	opt := &github.BranchListOptions{ListOptions: github.ListOptions{PerPage: 50}}

	for {
		branches, resp, err := c.client.Repositories.ListBranches(ctx, owner, repo, opt)
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

func (c *Client) ListTags(ctx context.Context, owner, repo string) ([]string, error) {
	var all []string
	opt := &github.ListOptions{PerPage: 50}

	for {
		tags, resp, err := c.client.Repositories.ListTags(ctx, owner, repo, opt)
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

func (c *Client) DownloadArchive(ctx context.Context, owner, repo, ref, out string) error {
	url, _, err := c.client.Repositories.GetArchiveLink(ctx, owner, repo, github.Zipball, &github.RepositoryContentGetOptions{Ref: ref}, 0)
	if err != nil {
		return fmt.Errorf("GetArchiveLink failed %s/%s %s: %w", owner, repo, ref, err)
	}

	resp, err := http.Get(url.String())
	if err != nil {
		return fmt.Errorf("HTTP download error: %w", err)
	}
	defer resp.Body.Close()

	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("File create error: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("Write error: %w", err)
	}
	return nil
}

func (c *Client) ProcessRepository(ctx context.Context, owner, repo, baseDir string) error {
	repoDir := filepath.Join(baseDir, repo)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("Cannot create repo dir: %w", err)
	}

	branchDir := filepath.Join(repoDir, "branches")
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		return fmt.Errorf("Cannot create branch dir: %w", err)
	}
	branches, err := c.ListBranches(ctx, owner, repo)
	if err == nil {
		for _, br := range branches {
			out := filepath.Join(branchDir, fmt.Sprintf("%s-branch-%s.zip", repo, br))
			if err := c.DownloadArchive(ctx, owner, repo, "heads/"+br, out); err != nil {
				return err
			}
		}
	}

	tagDir := filepath.Join(repoDir, "tags")
	if err := os.MkdirAll(tagDir, 0755); err != nil {
		return fmt.Errorf("Cannot create tag dir: %w", err)
	}
	tags, err := c.ListTags(ctx, owner, repo)
	if err == nil {
		for _, tg := range tags {
			out := filepath.Join(tagDir, fmt.Sprintf("%s-tag-%s.zip", repo, tg))
			if err := c.DownloadArchive(ctx, owner, repo, "tags/"+tg, out); err != nil {
				return err
			}
		}
	}
	return nil
}
