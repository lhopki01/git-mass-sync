package actions

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncRepos(t *testing.T) {
	testDir := CreateTestDirs()
	repos := Repos{
		&Repo{
			Name: "gitDir",
		},
	}
	repos.SyncRepos(testDir)
	assert.Equal(t, "no git remotes found\n", repos[0].Message)
	assert.Equal(t, Error, repos[0].Severity)
	os.RemoveAll(testDir)
}

func TestCloneRepos(t *testing.T) {
	testDir := CreateTestDirs()
	repos := Repos{
		&Repo{
			Name:   "bar",
			SSHURL: "git@github.com/foo/bar.git",
		},
	}
	repos.CloneRepos(testDir)
	//expectedFailures := []string{"[cyan]Cloning git@gitub.com/foo/bar.git: [red]exit status 128\nfatal: repository 'git@gitub.com/foo/bar.git' does not exist\n"}
	assert.Equal(t, "fatal: repository 'git@github.com/foo/bar.git' does not exist\n", repos[0].Message)
	assert.Equal(t, Error, repos[0].Severity)
	os.RemoveAll(testDir)
}

func TestArchiveRepos(t *testing.T) {
	testDir := CreateTestDirs()
	archiveDir := testDir + "/.archive"
	repos := Repos{
		&Repo{
			Name:     "gitDir",
			Archived: true,
		},
		&Repo{
			Name:     "nonExistantDir",
			Archived: true,
		},
	}

	repos.ArchiveRepos(testDir, archiveDir)
	expectedFailures := fmt.Sprintf("rename %s/nonExistantDir %s/.archive/nonExistantDir: no such file or directory", testDir, testDir)
	assert.Equal(t, expectedFailures, repos[1].Message)
	assert.DirExists(t, archiveDir+"/gitDir")
	os.RemoveAll(testDir)
}

func CreateTestDirs() string {
	dir, err := ioutil.TempDir("", "git-mass-sync")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Mkdir(dir+"/gitDir", 0755)
	if err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = dir + "/gitDir"
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	err = os.Mkdir(dir+"/notGitDir", 0755)
	if err != nil {
		log.Fatal(err)
	}

	_, err = os.Create(dir + "/file")
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func TestGetDirList(t *testing.T) {
	testDir := CreateTestDirs()
	assert.Equal(t, []string{"gitDir"}, GetGitDirList(testDir))
	os.RemoveAll(testDir)
}

func TestRemoveElementFromSlice(t *testing.T) {
	type testCase struct {
		tName         string
		slice         []string
		indexToRemove int
		expectedSlice []string
	}
	testCases := []testCase{
		{
			tName:         "remove from the middle",
			slice:         []string{"a", "b", "c"},
			indexToRemove: 1,
			expectedSlice: []string{"a", "c"},
		},
		{
			tName:         "remove from end",
			slice:         []string{"a", "b", "c"},
			indexToRemove: 2,
			expectedSlice: []string{"a", "b"},
		},
		{
			tName:         "remove from beginning",
			slice:         []string{"a", "b", "c"},
			indexToRemove: 0,
			expectedSlice: []string{"b", "c"},
		},
		{
			tName:         "remove from len1 slice",
			slice:         []string{"a"},
			indexToRemove: 0,
			expectedSlice: []string{},
		},
		{
			tName:         "remove out of bounds index",
			slice:         []string{"a", "b", "c"},
			indexToRemove: 3,
			expectedSlice: []string{"a", "b", "c"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.tName, func(t *testing.T) {
			t.Parallel()
			result := RemoveElementFromSlice(tc.slice, tc.indexToRemove)
			sort.Strings(result)
			assert.Equal(t, tc.expectedSlice, result)
		})
	}
}
