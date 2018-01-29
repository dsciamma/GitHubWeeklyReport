GitHub Weekly Report generator
==================================================

This lambda function uses data input to query GitHub GraphQL API and extract activity for the past week.
Input:
{
  "organization": "GitHub organization name"
  "gitHubToken": "GitHub token used to query API"
}
