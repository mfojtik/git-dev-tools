Developers tools and helpers
============================

#### git-sync

This utility will take a path as an argument and look for the `.gitrepos` file
in given path. This file contains a list of directories with GIT repositories
you want to synchronize with `upstream/master`. This utility assume you have
following branches and remotes:

* `origin/master` - This is a master branch of your Github fork.
* `upstream/master` - This is the original repo (repo you forked from) master.

The tool will do following operations for each repository defined:

* `git fetch upstream`
* `git checkout master`
* `git merge upstream/master`
* `git branch` (get list of branches)
* `git checkout $branch`
* `git cherry upstream/master`

If all commits from $branch are available in `upstream/master`, then:

* `git checkout master`
* `git branch -D $branch`
* `git push origin :$branch`
