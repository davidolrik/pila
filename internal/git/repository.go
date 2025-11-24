package git

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
)

type LocalRepository struct {
	Type       string
	Repository *git.Repository
}

func GetLocalRepository() (*LocalRepository, error) {
	repo := &LocalRepository{
		Type: "unknown",
	}

	r, err := git.PlainOpen(".")
	if err != nil {
		panic(err)
	}
	repo.Repository = r

	// repo.detectType()

	return repo, nil
}

func (r *LocalRepository) Note(format string, arg ...any) {
	format = fmt.Sprintf("\n# %s", format)
	fmt.Println(color.HiBlackString(fmt.Sprintf(format, arg...)))
}

func (r *LocalRepository) Warn(format string, arg ...any) {
	format = fmt.Sprintf("\n# %s", format)
	fmt.Println(color.HiYellowString(fmt.Sprintf(format, arg...)))
}

func (r *LocalRepository) Err(format string, arg ...any) {
	format = fmt.Sprintf("\n# %s", format)
	fmt.Println(color.HiRedString(fmt.Sprintf(format, arg...)))
}

func (r *LocalRepository) ExecuteGitCommand(arg ...string) (string, error) {
	fmt.Println(color.CyanString("$ git %s", strings.Join(arg, " ")))

	return r.ExecuteGitCommandQuiet(arg...)
}

func (r *LocalRepository) ExecuteGitCommandQuiet(arg ...string) (string, error) {
	cmd := exec.Command("git", arg...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stderr.String(), err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Returns branch name of merge or empty string
func (r *LocalRepository) OngoingMergeBranchName() (string, error) {
	mergeHeadShaBytes, err := os.ReadFile(".git/MERGE_HEAD")
	if err != nil {
		return "", err
	}
	branchName, err := r.ExecuteGitCommandQuiet("name-rev", "--name-only", strings.TrimSpace(string(mergeHeadShaBytes)))
	if err != nil {
		return "", err
	}

	return branchName, nil
}

func (r *LocalRepository) detectType() error {
	fmt.Println("Detecting type")

	// Look at origin
	cfg, err := r.Repository.ConfigScoped(config.GlobalScope)
	if err != nil {
		panic(err)
	}
	// spew.Dump(cfg)
	fmt.Println(cfg.Init.DefaultBranch)
	fmt.Println(cfg.Remotes["origin"].URLs[0])

	gitOriginUrl, err := url.Parse(cfg.Remotes["origin"].URLs[0])
	if err != nil {
		panic(err)
	}

	hostname := gitOriginUrl.Hostname()
	fmt.Println(gitOriginUrl.Hostname())

	if hostname == "gitlab.com" {
		r.Type = "gitlab"
		return nil
	}

	if hostname == "github.com" {
		r.Type = "github"
		return nil
	}

	var possibleGitLabHttpURLs []string
	switch gitOriginUrl.Scheme {
	case "ssh":
		possibleGitLabHttpURLs = []string{
			"https://" + hostname + "/",
			"http://" + hostname + "/",
		}

	case "http":
		possibleGitLabHttpURLs = []string{
			"http://" + hostname + "/",
		}

	case "https":
		possibleGitLabHttpURLs = []string{
			"https://" + hostname + "/",
		}
	default:
		panic("Unknown scheme")
	}

	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   2 * time.Second, // Connect timeout
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   2 * time.Second,
			ResponseHeaderTimeout: 2 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Return an error to stop following redirects
			return http.ErrUseLastResponse
		},
	}

	for _, possibleURL := range possibleGitLabHttpURLs {
		req, err := http.NewRequest(http.MethodHead, possibleURL, nil)
		if err != nil {
			panic("unable to create HEAD request")
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		fmt.Println(resp.StatusCode)
		if resp.Header.Get("X-Gitlab-Meta") != "" {
			r.Type = "gitlab"
			return nil
		}
	}

	return nil
}
