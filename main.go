// A CLI for managing Github Classroom workflows
// Brighton Balfrey, balfrey@usc.edu

package main

import (
    "fmt"
    "runtime"
    "os"
    "os/exec"
    "io"
    "errors"
    "context"
    "strings"
    "encoding/csv"
    "log"
    "bufio"
    "flag"
    "github.com/google/go-github/v32/github"
    "golang.org/x/oauth2"
    "github.com/joho/godotenv"
)

/**
 * Open url in web browser
 * https://gist.github.com/hyg/9c4afcd91fe24316cbf0
 */
func StartBrowser(url string) error {
    var err error
    switch runtime.GOOS {
    case "linux":
        err = exec.Command("xdg-open", url).Start()
    case "windows":
        err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
    case "darwin":
        err = exec.Command("open", url).Start()
    }
    return err
}

/**
 * Fetches GITHUB_AUTH_TOKEN string from env
 */
func GithubTokenFromEnv() (string, error) {
    return fromEnv("GITHUB_AUTH_TOKEN")
}

/**
 * Fetches GITHUB_CLASSROOM_ORG string from env
 */
func GithubOrgFromEnv() (string, error) {
    return fromEnv("GITHUB_CLASSROOM_ORG")
}

/**
 * Gets GRADING_LOGGING_DEST string from env
 */
func LoggingDestFromEnv() (string, error) {
    return fromEnv("GRADING_LOGGING_DEST")
}

/**
 * Fetches GITHUB_USERNAME_MAP string from env
 */
func UsernameMapPathFromEnv() (string, error) {
    return fromEnv("GITHUB_USERNAME_MAP")
}

/**
 * Accessor function to get env var from system.
 */
func fromEnv(name string) (string, error) {
    res, prs := os.LookupEnv(name)
    if !prs {
        err := fmt.Sprintf("Problem getting %s from environment. Make sure it's set.", name)
        return "", errors.New(err)
    }
    return res, nil
}

/**
 * Gets Authorized Github HTTP Client
 */
func GithubClient(ctx context.Context, token string) *github.Client {
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    tc := oauth2.NewClient(ctx, ts)
    client := github.NewClient(tc)
    return client
}

/**
 * Fetches all repos in a given org.
 */
func OrgRepos(ctx context.Context, client *github.Client, org string) ([]*github.Repository, error) {

    // slice to contain all repos, and err object
    var allRepos []*github.Repository

    // Set options to load 100 repos at a time
    // This is bc the # of repos in a given classroom org
    // tends to be in the scale of 100s (at least), as there
    // is usually one repo per assignment
    opt := &github.RepositoryListByOrgOptions{
        ListOptions: github.ListOptions{PerPage: 100},
    }
    for {
        repos, resp, err := client.Repositories.ListByOrg(ctx, org, opt)
        if err != nil {
            return allRepos, err
        }
        allRepos = append(allRepos, repos...)
        if resp.NextPage == 0 {
            break
        }
        opt.Page = resp.NextPage
    }
    return allRepos, nil
}

func RepoNameByPrefixAndUser(pref, user string) string {
    return fmt.Sprintf("%s%s", pref, user)
}

/**
 * Github classroom repos are created with the following naming standard:
 * `{some assignment prefix}{username}`, with a good example being
 * `assignment-3-brighton1101`
 *
 * This method operates under that assumption: there is some assignment
 * prefix that needs to be combined with the users' username. The repo
 * then lives under the org
 */
func RepoByPrefixAndUser(ctx context.Context, client *github.Client, org, pref, user string) (*github.Repository, error) {
    rname := RepoNameByPrefixAndUser(pref, user)
    repo, _, err := client.Repositories.Get(ctx, org, rname)
    return repo, err
}

// Wrapper struct for posting issues
type PostIssueOptions struct {
    OrgName string
    RepoName string
    Header string
    Body string
}

/**
 * Posts an issue to a Github repo.
 */
func PostIssue(ctx context.Context, client *github.Client, opts *PostIssueOptions) error {
    issueopt := &github.IssueRequest{
        Title: &opts.Header,
        Body: &opts.Body,
    }
    _, _, err := client.Issues.Create(ctx, opts.OrgName, opts.RepoName, issueopt)
    return err
}

/**
 * Get url to repo from repo object
 */
func RepoUrl(repo *github.Repository) string {
    return *repo.HTMLURL
}

/**
 * Assignment repos for specific assignments can be identified by a prefix in the
 * repository name. Given the desired prefix, return all repos with names
 * that contain that prefix
 */
func FilterReposByPref(repos []*github.Repository, pref string) []*github.Repository {
    var prefRepos []*github.Repository
    for _, repo := range(repos) {
        if strings.Contains(*repo.Name, pref) {
            prefRepos = append(prefRepos, repo)
        }
    }
    return prefRepos
}

/**
 * Get username from repo
 */
func UsernameFromRepo(repo *github.Repository, pref string) string {
    reponame := *repo.Name
    return strings.Replace(reponame, pref, "", 1)
}

/**
 * Reads a username map from a given csv file. Operates under the assumption that
 * the 0 index will contain the name, and the 1 index will contain the username. Returns
 * two maps, for either direction.
 */
func ReadUsernameMap(path string) (map[string]string, map[string]string, error) {
    f, err := os.Open(path)
    if err != nil {
        emsg := fmt.Sprintf("Could not load file from given path: %s", path)
        return nil, nil, errors.New(emsg)
    }
    defer f.Close()
    reader := csv.NewReader(f)
    name_username := map[string]string{}
    username_name := map[string]string{}
    for {
        row, err := reader.Read()
        if err == io.EOF {
            return name_username, username_name, nil
        } else if err != nil {
            return nil, nil, err
        }
        name := row[0]
        username := row[1]
        name_username[name] = username
        username_name[username] = name
    }
}

/**
 * Gather input from user, terminated by the endin char
 */
func GatherInput() (string, error) {
    reader := bufio.NewReader(os.Stdin)
    in, err := reader.ReadString('\n')
    if err != nil {
        return "", err
    }
    // This is ugly. There's probably a better way to do this,
    // but eh it's the way I knew how to do it haha
    in = strings.Replace(in, "\n", "", -1)
    return in, nil
}

/**
 * Handles gathering and posting feedback for given repo
 */
func HandleIssueFeedback(ctx context.Context, client *github.Client, repo *github.Repository, org string) (string, error) {
    fmt.Println("Enter any feedback for student:")
    fback, err := GatherInput()
    if err != nil {
        return "", err
    }
    // Gracefully exit if no feedback provided - don't want to post empty feedback
    if fback == "" {
        return "", nil
    }
    issueopts := &PostIssueOptions{
        // This might cause this to fail for personal repos
        // outside of organizations. Since this is designed with
        // Github Classroom Organizations in mind, we're betting that
        // this property is defined
        OrgName: org,
        RepoName: *repo.Name,
        Header: "[FEEDBACK]",
        Body: fback,
    }
    return fback, PostIssue(ctx, client, issueopts)
}

/**
 * Handles individual repo
 */
func HandleRepo(ctx context.Context,
    client *github.Client,
    repo *github.Repository,
    username, name, orgname string,
    postfeedback bool) error {
    if name == "" {
        msg := ""
        if postfeedback {
            msg = "You can still post feedback below."
        }
        fmt.Printf("Note that username %s not found in CSV. %s\n", username, msg)
        name = "[NAME NOT FOUND]"
    }

    // Get repo url from repo
    url := RepoUrl(repo)

    // Open URL in browser
    err := StartBrowser(url)
    if err != nil {
        return err
    }

    // Handle Feedback and log
    if postfeedback {
        fback, err := HandleIssueFeedback(ctx, client, repo, orgname)
        if err != nil {
            return err
        } else if fback != "" {
            log.Println(fmt.Sprintf("Name: %s, Username: %s, Feedback: %s", name, username, fback))
        }
    }
    return nil
}

/**
 * Handles single student.
 */
func SingleStudent(ctx context.Context,
    client *github.Client,
    postfeedback bool,
    prefix, orgname, username, name string,
    username_name, name_username map[string]string) error {

    // If name is not defined, get the name from username
    if name == "" {
        var prs bool
        name, prs = username_name[username]
        if !prs {
            fmt.Printf("Note that username %s not found in csv. Trying to proceed w/o... \n", username)
        }
    }

    // If username is not defined, get the username from name
    if username == "" {
        var prs bool
        username, prs = name_username[name]
        if !prs {
            return errors.New(
                fmt.Sprintf("Username for name %s not found in mapping", name))
        }
    }

    // Get repo for user based on org, prefix, and username
    repo, err := RepoByPrefixAndUser(ctx, client, orgname, prefix, username)
    if err != nil {
        return err
    }

    return HandleRepo(ctx, client, repo, username, name, orgname, postfeedback)
}

/**
 * Handles all students in the username_name map that have assignments
 * with the given prefix within the org
 */
func AllStudents(
    ctx context.Context,
    client *github.Client,
    postfeedback bool,
    prefix, orgname string,
    username_name map[string]string) error {
    
    // Get all repos for the org with prefix
    unfilteredrepos, err := OrgRepos(ctx, client, orgname)
    if err != nil {
        return err
    }

    // Filter by prefix for assignment
    repos := FilterReposByPref(unfilteredrepos, prefix)

    // Iterate over all repos with given prefix
    for _, repo := range repos {

        // Look up name by username
        // Note: It's non blocking for name mapping to not be present here,
        // but there will be a warning message.
        username := UsernameFromRepo(repo, prefix)
        name, prs := username_name[username]
        if !prs {
            name = ""
        }

        err := HandleRepo(ctx, client, repo, username, name, orgname, postfeedback)
        if err != nil {
            return err
        }

        if !postfeedback {
            fmt.Print("Press enter to continue")
            _, inerr := GatherInput()
            if inerr != nil {
                return inerr
            }
        }
    }
    return nil
}

func main() {

    // First we gather two pieces of information from the user
    // 1. The assignment prefix (ie, 'assignment-2-') AND
    // 2. EITHER the students' full name, as saved in the csv
    //    that maps names to usernames, or the students' github
    //    username itself
    in_prefix := flag.String("p", "",
        "The assignment prefix to usernames for github classroom ie 'assignment-2-'")
    in_name := flag.String("n", "",
        "The students' full name to use")
    in_username := flag.String("u", "",
        "The students' username to use")
    in_postfeedback := flag.Bool("f", false,
        "Whether or not to post feedback if entered to Github Classroom")
    in_allstudents := flag.Bool("a", false,
        "Handle all students, as opposed to a single student")
    flag.Parse()
    prefix := *in_prefix
    name := *in_name
    username := *in_username
    postfeedback := *in_postfeedback

    // Only allow users one option or the other
    if name != "" && username != "" {
        fmt.Println("Using -n and -u together is not allowed. Please only specify one.")
        return
    }

    // Require prefix
    if prefix == "" {
        fmt.Println("Did not provide prefix. Cannot proceed without -p flag.")
    }

    // Load environment vars from .env file
    derr := godotenv.Load()
    if derr != nil {
        fmt.Println(derr)
        return
    }

    // Set up logging based on GRADING_LOGGING_DEST var
    logdir, perr := LoggingDestFromEnv()
    if perr != nil {
        fmt.Println(perr)
        return
    }
    f, lerr := os.OpenFile(fmt.Sprintf("%s/%s.log", logdir, prefix), os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if lerr != nil {
        fmt.Println(lerr)
        return
    }
    defer f.Close()
    log.SetOutput(f)

    // Read mappings between usernames and passwords
    mapdir, perr := UsernameMapPathFromEnv()
    if perr != nil {
        fmt.Println(perr)
        return
    }
    name_username, username_name, nerr := ReadUsernameMap(mapdir)
    if nerr != nil {
        fmt.Println(nerr)
        return
    }

    // Get orgname from env var
    orgname, oerr  := GithubOrgFromEnv()
    if oerr != nil {
        fmt.Println(oerr)
        return
    }

    // Get Github token from env var
    token, terr := GithubTokenFromEnv()
    if terr != nil {
        fmt.Println(terr)
        return
    }

    // Get context and github client
    // I think? these should be treated as singletons or
    // at least singleton-like
    ctx := context.Background()
    client := GithubClient(ctx, token)

    var err error
    if *in_allstudents {
        err = AllStudents(ctx, client, postfeedback, prefix, orgname, username_name)
    } else {
        err = SingleStudent(
            ctx, client, postfeedback, prefix, orgname, username, name, username_name, name_username)
    }
    if err != nil {
        fmt.Println(err)
    }
}
