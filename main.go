package main

import (
  "fmt"
  "context"
	//"io/ioutil"
  "sort"
  "time"

  "github.com/dsciamma/GitHubWeeklyReport/report"
	"github.com/aws/aws-lambda-go/lambda"
)

// ReportInput defines the JSON structure expected as Input of the Lambda
type ReportInput struct {
  Organization string
  GitHubToken string
}

// HandleRequest is the general lambda handler
func HandleRequest(ctx context.Context, input ReportInput) (string, error) {

  ghreport := report.NewGitHubReport(input.Organization, input.GitHubToken, 7)
  ghreport.Log = func(s string) { fmt.Printf(s) }
  err := ghreport.Run()

  if err != nil {
    fmt.Printf("An error occured %v\n", err)
  } else {
    now := time.Now()

    // Print report
    fmt.Printf(`
============================
Report
============================
`)

    // Display Merged PR
    fmt.Printf("\nMerged PRs:\n")
    for _, pullrequest := range ghreport.Result.MergedPRs {
      t, _ := time.Parse(report.ISO_FORM, pullrequest.MergedAt)
      diff := now.Sub(t)
      fmt.Printf("\t'%s' on %s merged %d days ago\n", pullrequest.Title, pullrequest.Repository, int(diff / (24 * time.Hour)))
    }

    // Display activity for Open PR
    top := 10
    if len(ghreport.Result.OpenPRsWithActivity) < top {
      top = len(ghreport.Result.OpenPRsWithActivity)
    }
    fmt.Printf("\nTop %d Open PRs with activity:\n", top)

    sort.Sort(report.ByActivity(ghreport.Result.OpenPRsWithActivity))
    for i := 0; i < top; i++ {
      pr := ghreport.Result.OpenPRsWithActivity[i]
      fmt.Printf(
        "\t'%s' on %s with %d events\n",
        pr.Title, pr.Repository, pr.Timeline.TotalCount)
    }

    fmt.Printf("\n\t%d other PRs with activity\n", (len(ghreport.Result.OpenPRsWithActivity) - top))

    fmt.Printf("\nOpen PRs without activity: %d\n", len(ghreport.Result.OpenPRsWithoutActivity))

    sort.Sort(report.ByAge(ghreport.Result.OpenPRsWithoutActivity))
    top = 10
    if len(ghreport.Result.OpenPRsWithoutActivity) < top {
      top = len(ghreport.Result.OpenPRsWithoutActivity)
    }
    fmt.Printf("\n%d oldest open PRs without activity:\n", top)
    for i := 0; i < top; i++ {
      pr := ghreport.Result.OpenPRsWithoutActivity[i]
      t, _ := time.Parse(report.ISO_FORM, pr.CreatedAt)
      diff := now.Sub(t)
      fmt.Printf(
        "\t'%s' on %s created %d days ago\n",
        pr.Title, pr.Repository, int(diff / (24 * time.Hour)))
    }
  }
  return fmt.Sprintf("Hello %s!", input.Organization ), nil
}


func main() {
  lambda.Start(HandleRequest)
}
