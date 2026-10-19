# Demo notes

These are the screens worth capturing for the portfolio. Use the running application, not mockups.

## Short walkthrough (60–90 seconds)

1. Start the app locally using the instructions in the root README.
2. Sign in as a homeowner and show a job you have posted.
3. Sign in as an approved tradesperson and submit a bid for that job.
4. Return to the homeowner account, review bids, and award the job.
5. Show the accepted/rejected bid statuses and the awarded job state.

Keep the browser window focused on the app. Cut loading time, not actual application steps.

## Three useful screenshots

- **Homeowner:** job details showing what was posted.
- **Tradesperson:** an open job and its bid form or submitted bid.
- **Outcome:** the homeowner's bid list after a job is awarded.

If one screenshot contains private details, remove or replace the test data inside the app *before* taking it. Avoid publishing real customer names, phone numbers, addresses, login details, or database credentials.

Save genuine captures in `docs/images/` and link to them from the top of the README after reviewing them. If there are no real screenshots yet, it's better to leave that section absent.

## What to explain out loud

The interesting backend detail is not the number of forms. Awarding one bid also updates the competing bids and the job status. The [award transaction](../internal/repo/bids.go) locks rows and performs the changes together.

Be specific about what's implemented and tested. The Go checks and database tests are useful, but they aren't evidence of production security or an end-to-end browser test.

## Before sharing

- [ ] Use only demonstration accounts and invented job details.
- [ ] Show the app behaving as described; don't substitute a static design.
- [ ] Remove secrets and local-only credentials from recordings.
- [ ] Verify every screenshot matches the current `main` branch.
- [ ] Only describe scenarios you actually tried.
