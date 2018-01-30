package main

import (
  "fmt"
  "context"
	//"io/ioutil"
  "sort"
  "time"

  "github.com/dsciamma/GitHubWeeklyReport/report"
	"github.com/aws/aws-lambda-go/lambda"
  "github.com/nlopes/slack"
)

// ReportInput defines the JSON structure expected as Input of the Lambda
type ReportInput struct {
  GitHubOrganization string
  GitHubToken string
  SlackToken string
  SlackChannel string
}

// BuildMessageParameters generates the message parameters from a GitHubReport.
// The JSON equivalent looks like:
// {
//    "attachments": [
//        {
//            "title": "15 PullRequests merged!",
//      "text": "- <http://foo.com/|PR1>: 6 days ago\n- <http://foo.com/|PR2>: 3 days ago"
//        },
//        {
//            "title": "20 open PullRequests with an activity. Here is the 5 most active:",
//      "text": "- <http://foo.com/|PR1>: 16 events, 4 participants\n- <http://foo.com/|PR2>: 12 events, 3 participants"
//        },
//        {
//            "title": "4 open PullRequests without any activity last week. Here is the 5 oldest:",
//      "text": "- <http://foo.com/|PR1>: open 233 days ago\n- <http://foo.com/|PR2>: open 154 days ago"
//        }
//    ]
//}
func BuildMessageParameters(ghreport report.GitHubReport) (slack.PostMessageParameters, error) {

  now := time.Now()
  params := slack.PostMessageParameters{}

  /*
   * Create Merged PR section
   */
  mergedPRAttachment := slack.Attachment{}
  mergedPRAttachment.Title = fmt.Sprintf(
    "%d Merged PRs!",
    len(ghreport.Result.MergedPRs))
  var buffer bytes.Buffer
  for _, pullrequest := range ghreport.Result.MergedPRs {
    t, _ := time.Parse(report.ISO_FORM, pullrequest.MergedAt)
    diff := now.Sub(t)
    buffer.WriteString(
      fmt.Sprintf(
        "- <https://github.com/%s/%s/pull/%d|%s>: merged %d days ago\n",
        ghreport.Organization
        pullrequest.Repository,
        pullrequest.Number,
        pullrequest.Title,
        int(diff / (24 * time.Hour))))
  }
  mergedPRAttachment.Text = buffer.String()

  /*
   * Create Active Open PR section
   */
  buffer.Reset()
  top := NB_HIGHLIGHTS
  activePRAttachment := slack.Attachment{}
  if len(ghreport.Result.OpenPRsWithActivity) < top {
    top = len(ghreport.Result.OpenPRsWithActivity)
    activePRAttachment.Title = fmt.Sprintf(
      "%d open PRs with an activity:",
      len(ghreport.Result.OpenPRsWithActivity))
  } else {
    activePRAttachment.Title = fmt.Sprintf(
      "%d open PRs with an activity. Here is the %d most active ones:",
      len(ghreport.Result.OpenPRsWithActivity),
      top)
  }
  sort.Sort(report.ByActivity(ghreport.Result.OpenPRsWithActivity))
  for i := 0; i < top; i++ {
    pr := ghreport.Result.OpenPRsWithActivity[i]
    buffer.WriteString(
      fmt.Sprintf(
        "- <https://github.com/%s/%s/pull/%d|%s>: %d events, %d participants\n",
        ghreport.Organization
        pr.Repository,
        pr.Number,
        pr.Title,
        pr.Timeline.TotalCount,
        pr.Participants.TotalCount))
  }
  activePRAttachment.Text = buffer.String()

  /*
   * Create Inactive Open PR section
   */
  buffer.Reset()
  top := NB_HIGHLIGHTS
  inactivePRAttachment := slack.Attachment{}
  if len(ghreport.Result.OpenPRsWithoutActivity) < top {
    top = len(ghreport.Result.OpenPRsWithoutActivity)
    inactivePRAttachment.Title = fmt.Sprintf(
      "%d open PRs without any activity last week:",
      len(ghreport.Result.OpenPRsWithoutActivity))
  } else {
    inactivePRAttachment.Title = fmt.Sprintf(
      "%d open PRs without any activity last week. Here is the %d oldest:",
      len(ghreport.Result.OpenPRsWithoutActivity),
      top)
  }
  sort.Sort(report.ByAge(ghreport.Result.OpenPRsWithoutActivity))
  for i := 0; i < top; i++ {
    pr := ghreport.Result.OpenPRsWithoutActivity[i]
    t, _ := time.Parse(report.ISO_FORM, pr.CreatedAt)
    diff := now.Sub(t)
    buffer.WriteString(
      fmt.Sprintf(
        "- <https://github.com/%s/%s/pull/%d|%s>: open %d days ago\n",
        ghreport.Organization
        pullrequest.Repository,
        pullrequest.Number,
        pullrequest.Title,
        int(diff / (24 * time.Hour))))
  }
  inactivePRAttachment.Text = buffer.String()

  params.Attachments = []slack.Attachment{mergedPRAttachment, activePRAttachment, inactivePRAttachment}

  return params, nil
}

// HandleRequest is the general lambda handler
func HandleRequest(ctx context.Context, input ReportInput) (string, error) {

  const NB_HIGHLIGHTS = 5

  ghreport := report.NewGitHubReport(input.GitHubOrganization, input.GitHubToken, 7)
  ghreport.Log = func(s string) { fmt.Printf(s) }
  err := ghreport.Run()

  if err != nil {
    fmt.Printf("An error occured %v\n", err)
  } else {

    api := slack.New(input.SlackToken)
    params, errMessage := BuildMessageParameters(ghreport)
    if errMessage != nil {
      return "Error when building message", errMessage
    }

    channelID, timestamp, errPost := api.PostMessage(input.SlackChannel, "Here is your GitHub weekly report", params)
    if errPost != nil {
      return "Error when posting message", errPost
    }
  }
  return fmt.Sprintf("Report generated for %s!", input.GitHubOrganization ), nil
}


func main() {
  lambda.Start(HandleRequest)
}
