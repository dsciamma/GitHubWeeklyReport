package report

import (
  "fmt"
  "errors"
  //"sort"
  "time"
  "context"

  "golang.org/x/oauth2"

  "github.com/dsciamma/graphql"
)


const ISO_FORM = "2006-01-02T15:04:05Z"

type pageInfoStruct struct {
  HasNextPage bool
  StartCursor string
  EndCursor string
  HasPreviousPage bool
}

type rateLimitStruct struct {
  Limit int
  Cost int
  Remaining int
  ResetAt string
}

type userStruct struct {
  Login string
}

type PRStruct struct {
  Number int
  Title string
  Repository string
  CreatedAt string
  MergedAt string
  State string
  Participants struct {
    Nodes []userStruct
    PageInfo pageInfoStruct
    TotalCount int
  }
  Timeline struct {
    TotalCount int
  }
}

type ByActivity []PRStruct
func (a ByActivity) Len() int           { return len(a) }
func (a ByActivity) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByActivity) Less(i, j int) bool { return a[i].Timeline.TotalCount > a[j].Timeline.TotalCount }

type ByAge []PRStruct
func (a ByAge) Len() int           { return len(a) }
func (a ByAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAge) Less(i, j int) bool {
  ti, _ := time.Parse(ISO_FORM, a[i].CreatedAt)
  tj, _ := time.Parse(ISO_FORM, a[j].CreatedAt)
  return tj.After(ti)
}

type repositoriesResponseStruct struct {
  Organization struct {
    Repositories struct {
      Nodes []struct {
        Name string
        Owner userStruct
      }
      PageInfo pageInfoStruct
      TotalCount int
    }
  }
  RateLimit rateLimitStruct
}

type reportResponseStruct struct {
  Repository struct {
    Name string
    MergedPR struct {
      Nodes []PRStruct
      PageInfo pageInfoStruct
      TotalCount int
    }
    OpenPR struct {
      Nodes []PRStruct
      PageInfo pageInfoStruct
      TotalCount int
    }
    Refs struct {
      Nodes []struct {
        Name string
        Target struct {
          History struct {
            Nodes []struct {
              Oid string
              CommittedDate string
              Author userStruct
              Message string
            }
            PageInfo pageInfoStruct
            TotalCount int
          }
        }
      }
      PageInfo pageInfoStruct
      TotalCount int
    }
  }
  RateLimit rateLimitStruct
}


// Report object
type GitHubReport struct {
  Organization  string
  GitHubToken   string
  Duration      int
  ReportDate    time.Time
  Result        struct {
    MergedPRs               []PRStruct
    OpenPRsWithActivity     []PRStruct
    OpenPRsWithoutActivity  []PRStruct
  }

  // Log is called with various debug information.
  // To log to standard out, use:
  //  report.Log = func(s string) { log.Println(s) }
  Log func(s string)
}

// NewGitHubReport makes a new Report to extract data from GitHub.
func NewGitHubReport(org string, token string, duration int) *GitHubReport {
  report := &GitHubReport{
    Organization: org,
    GitHubToken: token,
    Duration: duration,
  }
  return report
}

func (gr *GitHubReport) ListRepositories(
  ctx context.Context,
  client *graphql.Client,
  organization string,
  cursor string) ([]string, error){

  var req *graphql.Request
  if cursor == "" {
    req = graphql.NewRequest(`
  query ($organization: String!, $size: Int!) {
    organization(login:$organization) {
      repositories(first:$size, affiliations:OWNER) {
        nodes {
          name
          owner {
            login
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
        totalCount
      }
    }
    rateLimit {
      limit
      cost
      remaining
      resetAt
    }
  }
    `)
    } else {
      req = graphql.NewRequest(`
    query ($organization: String!, $size: Int!, $cursor: String!) {
      organization(login:$organization) {
        repositories(first:$size, after:$cursor) {
          nodes {
            name
          }
          pageInfo {
            hasNextPage
            endCursor
          }
          totalCount
        }
      }
      rateLimit {
        limit
        cost
        remaining
        resetAt
      }
    }
      `)
      req.Var("cursor", cursor)
    }
  req.Var("organization", organization)
  req.Var("size", 50)


  repositories := []string{}
  var respData repositoriesResponseStruct
  if err := client.Run(ctx, req, &respData); err != nil {
    return nil, err
  } else {
    for _, repo := range respData.Organization.Repositories.Nodes {
      repositories = append(repositories, repo.Name)
    }
    if respData.Organization.Repositories.PageInfo.HasNextPage {
      additionalRepos, err := gr.ListRepositories(ctx, client, organization, respData.Organization.Repositories.PageInfo.EndCursor)
      if err != nil {
        return nil, err
      } else {
        repositories = append(repositories, additionalRepos...)
      }
    }
    gr.logf("Credits remaining %v\n", respData.RateLimit.Remaining)
    return repositories, nil
  }
}

func (gr *GitHubReport) ListSingleRepositories(
  ctx context.Context,
  client *graphql.Client,
  organization string,
  cursor string) ([]string, error){

  var req *graphql.Request
  if cursor == "" {
    req = graphql.NewRequest(`
  query ($organization: String!, $size: Int!) {
    organization(login:$organization) {
      repositories(last:$size, affiliations:OWNER) {
        nodes {
          name
          owner {
            login
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
        totalCount
      }
    }
    rateLimit {
      limit
      cost
      remaining
      resetAt
    }
  }
    `)
    } else {
      req = graphql.NewRequest(`
    query ($organization: String!, $size: Int!, $cursor: String!) {
      organization(login:$organization) {
        repositories(first:$size, after:$cursor) {
          nodes {
            name
          }
          pageInfo {
            hasNextPage
            endCursor
          }
          totalCount
        }
      }
      rateLimit {
        limit
        cost
        remaining
        resetAt
      }
    }
      `)
      req.Var("cursor", cursor)
    }
  req.Var("organization", organization)
  req.Var("size", 10)


  repositories := []string{}
  var respData repositoriesResponseStruct
  if err := client.Run(ctx, req, &respData); err != nil {
    return nil, err
  } else {
    for _, repo := range respData.Organization.Repositories.Nodes {
      repositories = append(repositories, repo.Name)
    }
    gr.logf("Credits remaining %v\n", respData.RateLimit.Remaining)
    return repositories, nil
  }
}

func (gr *GitHubReport) ReportRepository (
  ctx context.Context,
  client *graphql.Client,
  organization string,
  repository string,
  since time.Time) (reportResponseStruct, error){

  // make a request
  req := graphql.NewRequest(`
query ($organization: String!, $repo: String!, $date: GitTimestamp!, $date2: DateTime!, $size: Int!) {
  repository(owner: $organization, name: $repo) {
    name
    mergedPR: pullRequests(last: $size, states: [MERGED], orderBy: {field: UPDATED_AT, direction: ASC}) {
      nodes {
        number
        title
        createdAt
        participants(last: $size) {
          nodes {
            login
          }
          totalCount
        }
        mergedAt
      }
      totalCount
    }
    openPR: pullRequests(last: $size, states: [OPEN]) {
      nodes {
        number
        title
        createdAt
        mergedAt
        state
        participants(last: $size) {
          nodes {
            login
          }
          totalCount
        }
        timeline(since: $date2) {
          totalCount
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
      totalCount
    }
    refs(refPrefix: "refs/heads/", first: $size) {
      nodes {
        ... on Ref {
          name
          target {
            ... on Commit {
              history(first: $size, since: $date) {
                nodes {
                  ... on Commit {
                    oid
                    committedDate
                    author {
                      name
                    }
                    message
                  }
                }
                pageInfo {
                  hasNextPage
                  endCursor
                }
                totalCount
              }
            }
          }
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
      totalCount
    }
  }
  rateLimit {
    limit
    cost
    remaining
    resetAt
  }
}
  `)


  // set any variables
  req.Var("organization", organization)
  req.Var("repo", repository)
  req.Var("date", since.Format(ISO_FORM))
  req.Var("date2", since.Format(ISO_FORM))
  req.Var("size", 50)

  // run it and capture the response
  var respData reportResponseStruct
  if err := client.Run(ctx, req, &respData); err != nil {
    return respData, err
  } else {
    gr.logf("Credits remaining %v\n", respData.RateLimit.Remaining)
    return respData, nil
  }
}


func (gr *GitHubReport) logf(format string, args ...interface{}) {
  gr.Log(fmt.Sprintf(format, args...))
}

func (gr *GitHubReport) Run() error {

  // create a client (safe to share across requests)
  ctx := context.Background()
  tokenSource := oauth2.StaticTokenSource(
    &oauth2.Token{AccessToken: gr.GitHubToken},
  )
  httpClient := oauth2.NewClient(ctx, tokenSource)
  client := graphql.NewClient("https://api.github.com/graphql", graphql.WithHTTPClient(httpClient), graphql.UseInlineJSON())
  //client.Log = func(s string) { fmt.Println(s) }

  now := time.Now()
  since := now.AddDate(0, 0, -gr.Duration)

  gr.ReportDate = now

  repositories, err := gr.ListSingleRepositories(ctx, client, gr.Organization, "")
  if err != nil {
    return errors.New(fmt.Sprintf("An error occured during repositories listing %v\n", err))
  } else {
    for _, repoName := range repositories {
      report, err2 := gr.ReportRepository(ctx, client, gr.Organization, repoName, since)
      if err2 != nil {
        return errors.New(fmt.Sprintf("An error occured during report for %s: %v\n", repoName, err2))
      } else {
        // Build report

        // Extract Merged PR (keep the ones merged during last 7 days)
        for _, pullrequest := range report.Repository.MergedPR.Nodes {
          t, _ := time.Parse(ISO_FORM, pullrequest.MergedAt)
          if t.After(since) {
            pullrequest.Repository = repoName
            gr.Result.MergedPRs = append(gr.Result.MergedPRs, pullrequest)
          }
        }

        // Extract Open PR with and without activity
        for _, pullrequest := range report.Repository.OpenPR.Nodes {
          pullrequest.Repository = repoName
          if pullrequest.Timeline.TotalCount > 0 {
            gr.Result.OpenPRsWithActivity = append(gr.Result.OpenPRsWithActivity, pullrequest)
          } else {
            gr.Result.OpenPRsWithoutActivity = append(gr.Result.OpenPRsWithoutActivity, pullrequest)
          }
        }
      }
    }
    gr.logf("Nb merged pr:%d\n", len(gr.Result.MergedPRs))
    gr.logf("Nb open pr with activity:%d\n", len(gr.Result.OpenPRsWithActivity))
    gr.logf("Nb open pr without activity:%d\n", len(gr.Result.OpenPRsWithoutActivity))
    return nil
  }
}
