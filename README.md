# Grading tool for Github Classroom, written in Go

### Automating the clicking around when grading student assignments
- Opens link to students' repo within Github in the browser
- Optionally allows you (with the flag `-f`) to post feedback in the form of Github issue directly from command line
- Gets rid of all the useless clicking around and searching it takes to find students' repos within Github classroom, especially when names don't correspond with usernames, etc.
- For more background on why I did this, [check out this mini doc](https://docs.google.com/document/d/1h9DTCI3Gie3w6UbmmnIv3HzBrxqU0uV_6eZnTGBtaRU/edit?usp=sharing)
- Note that this is designed for a class that does <strong>not</strong> have any potential for automated testing

### What does this do?
- Automatically finds repos with assignment prefixes, and opens up the repo in the browser
- Prompts user for feedback optionally that will post a new issue automatically
- Cuts down on clicking around significantly, and makes it so users don't have to search around to find the right repo for the right student

### Flows:
- Single student flow allows you to get repo for single student, given username or name and assignment prefix
- Multi student flow gets all repos with given assignment prefix

### Configuration
- Create copy of `.env-example` to a file called `.env`
- Provide a `GITHUB_AUTH_TOKEN` which can be retrieved from user settings
- Provide a `GITHUB_CLASSROOM_ORG` which is the name of the github classroom org that assignments are in
- Provide a `GRADING_LOGGING_DEST` where feedback that is left is logged
- Provide a `GITHUB_USERNAME_MAP` which is the path to a headerless csv file containing two rows: first row should contain full student name, and second row should be github username

### Single Dependency
- go 1.12

### Usage:
```
go build -o main.o main.go
./main.o -u "brighton1101" -p "test-assignment-" -f
```
The above creates compiles the cli to `main.o`, which then can be executed
The flags indicate the following:
- `-help` will display the following information
- `-u` is for github username
- `-n` is for student name
- NOTE: only include one of the following: (`-u`, `-n`) (required for single student flow)
- NOTE: (`-u`, `-n`) flags only affect single student flow
- `-p` is for assignment prefix (ie `-p "assignment-2-"` ) (required)
- `-f` allows you to leave feedback for each student in form of an issue (optional)
- `-a` allows you to iterate through all repos with given prefix (optional)
