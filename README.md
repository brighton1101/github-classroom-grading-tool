# Grading tool for Github Classroom, written in Go


If you get the error:
```
Problem getting GITHUB_AUTH_TOKEN from environment. Make sure it's set.
```
and you are sure you have set the `GITHUB_AUTH_TOKEN` env var, try doing the following
- Find the path to your go installation (on macosx, should be `$HOME/go`)
- Set the `GOPATH` env var (I put this in my dotfile): `export GOPATH=$HOME/go` (or wherever your go installation is)
- Add to your path: `export PATH=$GOPATH/bin:$PATH`