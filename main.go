package main

import (
	"bytes"
	"context"
	"fmt"
	//"io/ioutil"
	"sort"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/nlopes/slack"
  "github.com/dsciamma/ghreport"
)

// NbHighlights defines the number of items displayed is the summary
const NbHighlights = 5

// ReportInput defines the JSON structure expected as Input of the Lambda
type ReportInput struct {
	GitHubOrganization string
	GitHubToken        string
	SlackToken         string
	SlackChannel       string
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
func BuildMessageParameters(report *ghreport.ActivityReport) (slack.PostMessageParameters, error) {

	now := time.Now()
	params := slack.PostMessageParameters{}

	/*
	 * Create Merged PR section
	 */
	mergedPRAttachment := slack.Attachment{}

	mergedPRAttachment.Title = fmt.Sprintf(
		"%d Merged PRs!",
		len(report.Result.MergedPRs))
	var buffer bytes.Buffer
	for _, pullrequest := range report.Result.MergedPRs {
		t, _ := time.Parse(ghreport.ISO_FORM, pullrequest.MergedAt)
		diff := now.Sub(t)
		buffer.WriteString(
			fmt.Sprintf(
				"- <https://github.com/%s/%s/pull/%d|%s>\n\t(%s) merged %d days ago\n",
				report.Organization,
				pullrequest.Repository,
				pullrequest.Number,
				pullrequest.Title,
				pullrequest.Repository,
				int(diff/(24*time.Hour))))
	}
	mergedPRAttachment.Color = "#36a64f"
	mergedPRAttachment.Text = buffer.String()


	/*
	 * Create Active Open PR section
	 */
	buffer.Reset()
	top := NbHighlights
	activePRAttachment := slack.Attachment{}
	if len(report.Result.OpenPRsWithActivity) < top {
		top = len(report.Result.OpenPRsWithActivity)
		activePRAttachment.Title = fmt.Sprintf(
			"%d open PRs with an activity:",
			len(report.Result.OpenPRsWithActivity))
	} else {
		activePRAttachment.Title = fmt.Sprintf(
			"%d open PRs with an activity. Here is the %d most active ones:",
			len(report.Result.OpenPRsWithActivity),
			top)
	}
	sort.Sort(ghreport.ByActivity(report.Result.OpenPRsWithActivity))
	for i := 0; i < top; i++ {
		pr := report.Result.OpenPRsWithActivity[i]
		buffer.WriteString(
			fmt.Sprintf(
				"- <https://github.com/%s/%s/pull/%d|%s>\n\t(%s) %d events, %d participants\n",
				report.Organization,
				pr.Repository,
				pr.Number,
				pr.Title,
				pr.Repository,
				pr.Timeline.TotalCount,
				pr.Participants.TotalCount))
	}
	activePRAttachment.Color = "#356ecc"
	activePRAttachment.Text = buffer.String()

	/*
	 * Create Inactive Open PR section
	 */
	buffer.Reset()
	top = NbHighlights
	inactivePRAttachment := slack.Attachment{}
	if len(report.Result.OpenPRsWithoutActivity) < top {
		top = len(report.Result.OpenPRsWithoutActivity)
		inactivePRAttachment.Title = fmt.Sprintf(
			"%d open PRs without any activity last week:",
			len(report.Result.OpenPRsWithoutActivity))
	} else {
		inactivePRAttachment.Title = fmt.Sprintf(
			"%d open PRs without any activity last week. Here is the %d oldest:",
			len(report.Result.OpenPRsWithoutActivity),
			top)
	}
	sort.Sort(ghreport.ByAge(report.Result.OpenPRsWithoutActivity))
	for i := 0; i < top; i++ {
		pr := report.Result.OpenPRsWithoutActivity[i]
		t, _ := time.Parse(ghreport.ISO_FORM, pr.CreatedAt)
		diff := now.Sub(t)
		buffer.WriteString(
			fmt.Sprintf(
				"- <https://github.com/%s/%s/pull/%d|%s>\n\t(%s) open %d days ago\n",
				report.Organization,
				pr.Repository,
				pr.Number,
				pr.Title,
				pr.Repository,
				int(diff/(24*time.Hour))))
	}
	inactivePRAttachment.Color = "#e89237"
	inactivePRAttachment.Text = buffer.String()

	params.Attachments = []slack.Attachment{mergedPRAttachment, activePRAttachment, inactivePRAttachment}

	return params, nil
}

// HandleRequest is the general lambda handler
func HandleRequest(ctx context.Context, input ReportInput) (string, error) {

	report := ghreport.NewActivityReport(input.GitHubOrganization, input.GitHubToken, 7)
	report.Log = func(s string) { fmt.Printf(s) }
	err := report.Run()

	if err != nil {
		fmt.Printf("An error occured %v\n", err)
	} else {

		api := slack.New(input.SlackToken)
		params, errMessage := BuildMessageParameters(report)
		if errMessage != nil {
			return "Error when building message", errMessage
		}

		_, _, errPost := api.PostMessage(input.SlackChannel, "Here is your GitHub weekly report", params)
		if errPost != nil {
			return "Error when posting message", errPost
		}
	}
	return fmt.Sprintf("Report generated for %s!", input.GitHubOrganization), nil
}

func main() {
	lambda.Start(HandleRequest)
}
