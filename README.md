Tool to sync all repos for a github org or in a local directory

### Install
```
brew tap lhopki01/brew git@github.com:lhopki01/brew
brew install git-mass-sync
```
If you have installed `git-mass-sync` before you need to remove the old tap
```
brew untap lhopki01/git-mass-sync
```

### Usage

#### Sync all repos in a github org

`git-mass-sync github foobar ~/github/foobar`

#### Find all git repos in a local directory and run hub sync on them

`git-mass-sync local ~/github/local_repos`
