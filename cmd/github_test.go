package cmd

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextPageLink(t *testing.T) {
	type testCase struct {
		tName            string
		linkHeader       string
		addLink          bool
		expectedNextPage string
	}
	testCases := []testCase{
		{
			tName:            "existing next page",
			linkHeader:       `<https://api.github.com/organizations/16915932/repos?page=2>; rel="next", <https://api.github.com/organizations/16915932/repos?page=12>; rel="last"`,
			addLink:          true,
			expectedNextPage: `https://api.github.com/organizations/16915932/repos?page=2`,
		},
		{
			tName:            "no next page",
			linkHeader:       `<https://api.github.com/organizations/16915932/repos?page=11&wper_page=2>; rel="prev", <https://api.github.com/organizations/16915932/repos?page=1&wper_page=2>; rel="first"`,
			addLink:          true,
			expectedNextPage: "",
		},
		{
			tName:            "no next links",
			linkHeader:       ``,
			addLink:          false,
			expectedNextPage: "",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.tName, func(t *testing.T) {
			t.Parallel()
			h := http.Header{}
			if tc.addLink {
				h.Add("Link", tc.linkHeader)
			}
			assert.Equal(t, tc.expectedNextPage, getNextPageLink(h))
		})
	}
}

func TestRepoActions(t *testing.T) {
	type testCase struct {
		tName           string
		repo            repo
		dirList         []string
		expectedAction  action
		expectedDirList []string
	}
	testCases := []testCase{
		{
			tName: "repo to archive",
			repo: repo{
				Name:     "archivedRepo",
				Archived: true,
				SSHURL:   "git@giturl",
			},
			dirList:         []string{"archivedRepo", "syncRepo", "deletedRepo"},
			expectedAction:  actionArchive,
			expectedDirList: []string{"syncRepo", "deletedRepo"},
		},
		{
			tName: "repo to clone",
			repo: repo{
				Name:     "cloneRepo",
				Archived: false,
				SSHURL:   "git@giturl/cloneRepo",
			},
			dirList:         []string{"archivedRepo", "syncRepo", "deletedRepo"},
			expectedAction:  actionClone,
			expectedDirList: []string{"archivedRepo", "syncRepo", "deletedRepo"},
		},
		{
			tName: "repo to sync",
			repo: repo{
				Name:     "syncRepo",
				Archived: false,
				SSHURL:   "git@giturl",
			},
			dirList:         []string{"archivedRepo", "syncRepo", "deletedRepo"},
			expectedAction:  actionSync,
			expectedDirList: []string{"archivedRepo", "deletedRepo"},
		},
		{
			tName: "repo to clone and archive",
			repo: repo{
				Name:     "cloneArchiveRepo",
				Archived: true,
				SSHURL:   "git@giturl/cloneArchiveRepo",
			},
			dirList:         []string{"archivedRepo", "syncRepo", "deletedRepo"},
			expectedAction:  actionCloneArchive,
			expectedDirList: []string{"archivedRepo", "syncRepo", "deletedRepo"},
		},
	}
	var repos []repo
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.tName, func(t *testing.T) {
			action, dirList := repoAction(tc.repo, tc.dirList)
			assert.Equal(t, tc.expectedAction, action)
			assert.Equal(t, tc.expectedDirList, dirList)
		})
		repos = append(repos, tc.repo)
	}

	inR, _ := regexp.Compile(".*")
	exR, _ := regexp.Compile("^$")
	reposToSync, reposToClone, reposToArchive := repoActions(repos, []string{"archivedRepo", "syncRepo", "deletedRepo"}, "foobar", inR, exR)
	assert.Equal(t, []string{"syncRepo"}, reposToSync)
	assert.Equal(t, []string{"git@giturl/cloneRepo", "git@giturl/cloneArchiveRepo"}, reposToClone)
	assert.Equal(t, []string{"archivedRepo", "cloneArchiveRepo", "deletedRepo"}, reposToArchive)
}

type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	// just in case you want default correct return value
	return &http.Response{}, nil
}

func TestGetRepoList(t *testing.T) {
	body := ioutil.NopCloser(bytes.NewReader([]byte(`
	[
		{
			"Name": "foobar",
			"ssh_url": "git@github.com/foobar.git",
			"Archive": false
		}
	]
	`)))

	client := &MockClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// do whatever you want
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       body,
			}, nil
		},
	}
	repoList := getRepoList("foobar", client)
	expectRepos := []repo{
		{
			SSHURL: "git@github.com/foobar.git", Name: "foobar", Archived: false,
		},
	}
	assert.Equal(t, expectRepos, repoList)

}

func TestProcessFlags(t *testing.T) {
	dir, archiveDir, org, inR, exR := processFlags([]string{"foobar", "/tmp/foobar"})
	assert.Equal(t, "/tmp/foobar", dir)
	assert.Equal(t, "/tmp/foobar/.archive", archiveDir)
	assert.Equal(t, "foobar", org)
	expectedInR, _ := regexp.Compile(".*")
	assert.Equal(t, expectedInR, inR)
	expectedExR, _ := regexp.Compile("^$")
	assert.Equal(t, expectedExR, exR)
}
