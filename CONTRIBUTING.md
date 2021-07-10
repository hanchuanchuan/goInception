# How to contribute

# Contribution Guide

This document outlines some conventions about development workflow, commit message formatting, contact points and other resources to make it easier to get your contribution accepted.

<!-- TOC -->

- [How to contribute](#how-to-contribute)
- [Contribution Guide](#contribution-guide)
  * [Before you get started](#before-you-get-started)
    + [Setting up your development environment](#setting-up-your-development-environment)
  * [Contribution Workflow](#contribution-workflow)
    + [Step 1: Fork in the cloud](#step-1--fork-in-the-cloud)
    + [Step 2: Clone fork to local storage](#step-2--clone-fork-to-local-storage)
    + [Step 3: Branch](#step-3--branch)
    + [Step 4: Develop](#step-4--develop)
      - [Edit the code](#edit-the-code)
      - [Prepare the test environment](#prepare-the-test-environment)
      - [Test](#test)
    + [Step 5: Keep your branch in sync](#step-5--keep-your-branch-in-sync)
    + [Step 6: Commit](#step-6--commit)
    + [Step 7: Push](#step-7--push)
    + [Step 8: Create a pull request](#step-8--create-a-pull-request)
    + [Step 9: Get a code review](#step-9--get-a-code-review)
  * [Code style](#code-style)

<!-- /TOC -->

## Before you get started

<!-- ### Sign the CLA

Click the **Sign in with GitHub to agree** button to sign the CLA. See an example [here](https://cla-assistant.io/pingcap/tidb).

What is [CLA](https://en.wikipedia.org/wiki/Contributor_License_Agreement)? -->

### Setting up your development environment

Before you start, you need
Set up your GO development environment.

1. Install `Go` version **1.14** or above. Refer to [How to Write Go Code](http://golang.org/doc/code.html) for more information.
2. Define `GOPATH` environment variable and modify `PATH` to access your Go binaries. A common setup is as follows. You could always specify it based on your own flavor.

    ```sh
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin
    ```

>**Note:** TiDB uses [`Go Modules`](https://github.com/golang/go/wiki/Modules)
to manage dependencies.

Now you should be able to use the `make build` command to build TiDB.

## Contribution Workflow

To contribute to the goInception code base, please follow the workflow as defined in this section.

1. Create a topic branch from where you want to base your work. This is usually master.
2. Make commits of logical units and add test case if the change fixes a bug or adds new functionality.
3. Run tests and make sure all the tests are passed.
4. Make sure your commit messages are in the proper format (see below).
5. Push your changes to a topic branch in your fork of the repository.
6. Submit a pull request.

Thanks for your contributions!


### Step 1: Fork in the cloud

1. Visit https://github.com/hanchuanchuan/goInception
2. On the top right of the page, click the `Fork` button (top right) to create
   a cloud-based fork of the repository.

### Step 2: Clone fork to local storage

Per Go's [workspace instructions](https://golang.org/doc/code.html#Workspaces),
place TiDB's code on your `GOPATH` using the following cloning procedure.

Define a local working directory:

```sh
# Set `user` to match your github profile name:
export user={your github profile name}
export working_dir=$GOPATH/src/github.com/${user}
```

Both `$working_dir` and `$user` are mentioned in the figure above.

Create your clone:

```sh
mkdir -p $working_dir
cd $working_dir

git clone https://github.com/${user}/goInception.git
# or: git clone git@github.com/${user}/goInception.git

cd $working_dir/goInception
git remote add upstream https://github.com/hanchuanchuan/goInception.git
# or: git remote add upstream git@github.com/hanchuanchuan/goInception.git

# Never push to the upstream master.
git remote set-url --push upstream no_push

# Confirm that your remotes make sense:
# It should look like:
# origin    git@github.com:$(user)/goInception.git (fetch)
# origin    git@github.com:$(user)/goInception.git (push)
# upstream  https://github.com/hanchuanchuan/goInception (fetch)
# upstream  no_push (push)
git remote -v
```

Set the `pre-commit` hook. This hook checks your commits for formatting,
building, doc generation, etc:

```sh
cd $working_dir/goInception/
ln -s `pwd`/hooks/pre-commit .git/hooks/
chmod +x $working_dir/goInception/.git/hooks/pre-commit
```

### Step 3: Branch

Get your local master up to date:

```sh
cd $working_dir/goInception
git fetch upstream
git checkout master
git rebase upstream/master
```

Branch from master:

```sh
git checkout -b feature-test
```

### Step 4: Develop

#### Edit the code

You can now edit the code on the `feature-test` branch.

#### Prepare the test environment

A mysql instance is required to run local tests to test audit, execution, and backup functions.

* mysql recommended version: `5.7`
* mysql startup parameters(or set my.cnf) `mysqld --log-bin=on --server_id=111 --character-set-server=utf8mb4`
* Create a test database `mysql -e "create database if not exists test DEFAULT CHARACTER SET utf8;create database if not exists test_inc DEFAULT CHARACTER SET utf8;"`
* Create a test user  `mysql -e "grant all on *.* to test@'127.0.0.1' identified by 'test';FLUSH PRIVILEGES;"`

#### Test

Build and run all tests:

```sh
# build and run the unit test to make sure all tests are passed.
make dev

# Check the checklist before you move on.
make checklist
```

You can also run a single unit test in a file. For example, to run test
`TestToInt64` in file `types/datum.go`:

```sh
GO111MODULE=on go test session/session_inception_*.go
# or:
GO111MODULE=on go test session/session_inception_test.go
```

### Step 5: Keep your branch in sync

*The branch needs to be merged with the latest version of goInception before submitting, so as not to be unable to merge PR*

```sh
# While on your feature-test branch.
git fetch upstream
git rebase upstream/master
```

Please don't use `git pull` instead of the above `fetch`/`rebase`. `git pull`
does a merge, which leaves merge commits. These make the commit history messy
and violate the principle that commits ought to be individually understandable
and useful (see below). You can also consider changing your `.git/config` file
via `git config branch.autoSetupRebase` always to change the behavior of `git pull`.

### Step 6: Commit

Commit your changes.

```sh
git commit
```

Likely you'll go back and edit/build/test further, and then `commit --amend` in a
few cycles.

### Step 7: Push

When the changes are ready to review (or you just to create an offsite backup
or your work), push your branch to your fork on `github.com`:

```sh
git push --set-upstream ${your_remote_name} feature-test
```

### Step 8: Create a pull request

1. Visit your fork at `https://github.com/$user/goInception`.
2. Click the `Compare & Pull Request` button next to your `feature-test` branch.
3. Fill in the required information in the PR template.

### Step 9: Get a code review

After PR submission, travisci test and circleci test will be automatically performed,
During the review, any changes can be directly modified and submitted on the branch, the PR will use the latest commit, and there is no need to submit the PR again.
If a PR involves multiple functions or fixes, it is recommended to split into multiple branches to facilitate review and merging.


## Code style

You can refer to the coding style suggested by the Golang community [style doc](https://github.com/golang/go/wiki/CodeReviewComments)
