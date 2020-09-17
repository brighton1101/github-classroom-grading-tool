# Grading tool for Github Classroom, written in Go

### Automating the clicking around when grading student assignments
- Opens link to students' repo within Github in the browser
- Optionally allows you (with the flag `-f`) to post feedback in the form of Github issue directly from command line
- Gets rid of all the useless clicking around and searching it takes to find students' repos within Github classroom, especially when names don't correspond with usernames, etc.
- For more background on why I did this, [check out this mini doc](https://docs.google.com/document/d/1h9DTCI3Gie3w6UbmmnIv3HzBrxqU0uV_6eZnTGBtaRU/edit?usp=sharing)

### Single Dependency
- go 1.12

### Usage:
```
go build -o main.o main.go
./main.o -u "brighton1101" -p "test-assignment-" -f
```
The above creates compiles the cli to `main.o`, which then can be executed
The flags indicate the following:
- `-u` is for github username
- `-n` is for student name
- NOTE: only include one of the following: (`-u`, `-n`)
- `-p` is for assignment prefix
- `-f` is for leaving feedback in form of github issue
