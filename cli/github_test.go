package cli

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/lhopki01/git-mass-sync/actions"
	"github.com/stretchr/testify/assert"
)

func TestRepoActions(t *testing.T) {
	type testCase struct {
		tName           string
		repo            *actions.Repo
		dirList         []string
		expectedAction  action
		expectedDirList []string
	}
	testCases := []testCase{
		{
			tName: "repo to archive",
			repo: &actions.Repo{
				Name:     "archivedRepo",
				Archived: true,
				SSHURL:   "git@giturl/archivedRepo",
			},
			dirList:         []string{"archivedRepo", "syncRepo", "deletedRepo"},
			expectedAction:  actionArchive,
			expectedDirList: []string{"syncRepo", "deletedRepo"},
		},
		{
			tName: "repo to clone",
			repo: &actions.Repo{
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
			repo: &actions.Repo{
				Name:     "syncRepo",
				Archived: false,
				SSHURL:   "git@giturl/syncRepo",
			},
			dirList:         []string{"archivedRepo", "syncRepo", "deletedRepo"},
			expectedAction:  actionSync,
			expectedDirList: []string{"archivedRepo", "deletedRepo"},
		},
		{
			tName: "repo to clone and archive",
			repo: &actions.Repo{
				Name:     "cloneArchiveRepo",
				Archived: true,
				SSHURL:   "git@giturl/cloneArchiveRepo",
			},
			dirList:         []string{"archivedRepo", "syncRepo", "deletedRepo"},
			expectedAction:  actionCloneArchive,
			expectedDirList: []string{"archivedRepo", "syncRepo", "deletedRepo"},
		},
	}
	var repos actions.Repos
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

	assert.Equal(t, actions.Repos{
		&actions.Repo{
			Name:   "syncRepo",
			SSHURL: "git@giturl/syncRepo",
		},
	}, reposToSync)

	assert.Equal(t, actions.Repos{
		&actions.Repo{
			Name:   "cloneRepo",
			SSHURL: "git@giturl/cloneRepo",
		},
		&actions.Repo{
			Name:     "cloneArchiveRepo",
			Archived: true,
			SSHURL:   "git@giturl/cloneArchiveRepo",
		},
	}, reposToClone)

	assert.Equal(t, actions.Repos{
		&actions.Repo{
			Name:     "archivedRepo",
			Archived: true,
			SSHURL:   "git@giturl/archivedRepo",
		},
		&actions.Repo{
			Name:     "cloneArchiveRepo",
			Archived: true,
			SSHURL:   "git@giturl/cloneArchiveRepo",
		},
		&actions.Repo{
			Name: "deletedRepo",
		},
	}, reposToArchive)
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
