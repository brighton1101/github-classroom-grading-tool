# Grading tool for Github Classroom, written in Go

### Automating the clicking around when grading student assignments

### Single Dependency
- go 1.12

### Usage:
```
go build -o main.o main.go
./main.o -u "skaterdav85" -p "test-assignment-" -f
```
The above creates compiles the cli to `main.o`, which then can be executed
The flags indicate the following:
- `-u` is for github username
- `-n` is for student name
- `-p` is for assignment prefix
- `-f` is for leaving feedback in form of github issue
