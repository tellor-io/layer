# ADR 002: QueryId time frame structure

## Authors

@themandalore
@brendaloya

## Changelog

- 2024-03-06: initial version
- 2024-04-02: clarity
- 2024-04-05: clarity
- 2024-08-03: clean up

## Context

For some queryId's, it will take longer than one block to submit data after a tip.  Examples include manual questions (e.g. "who is the president") or data without API's (e.g. prediction market answers, some US govt data).  To solve this, we set a report *time frame* for each query type when registering.  For example, a string question could set the *time frame* to take up to 5 hours for reports, giving all reporting parties a chance to see the question and formulate a response.  This ADR specifies how we are to handle the timing of tips, reporting commits, reveals, and restarts in case of unreported data.  

The proposed solution is that the report *time frame* starts on the block after the tip is received.  If there is another tip that happens during the report *time frame*, it is added to the original tip.  If no data is reported throughout the *time frame*, a new report *time frame* must be opened by another tip (it can be as small as one loya), and the previous unclaimed tip is added to the new tip.  

The commit reveal cycle remains intact and reveals can be any time before reveal window is over, even in the commit phase. Note that this doesn't prevent mirroring if revealed during the commit phase, but it allows reporters to wait to reveal if they feel that mirroring is occurring.

Aggregation of reports is done in `BeginBlock`; This is because in the current structure, if a report comes in before aggregation has happened but the window has expired, the new report will open a new query window before previous reports are aggregated causing the older reports not to be aggregated at all and always pushing the window expiration forward.  However, in `BeginBlock` queries with expired windows would be aggregated first before newer reports that are eligible could extend reporting window.  

## Alternative Approaches

### Time frame starts when first commit happens

One approach would be to have shorter report time frames where the window doesn't start right after the tip, but rather on the first report. This would allow for queries that are not seeking consensus to just have a tip with a short *time frame*, and then once a party reports, it is basically over and handled by disputes. The issue we had with this is that it unties the *time frame* from the tip.  Usually if you tip, you want some certainty around when you get the data. It also gives the false impression to the tipper that they should expect the data in some *time frame*.  If you tip for a manual query and it has a *time frame* of one hour, you would expect it to be reported in one hour.  The *time frame* concept is tied to how long it should be expected to take reporters to get the data.  If longer is needed, then the data spec itself is just likely not well defined or supported and it should be updated/changed.

### Time frame restarts if no one reports

Another option considered was to restart the *time frame* if no reports are received.  This could work, but no one might be reporting because the tip is too low or the question is vague.  It's also unclear if the report would still be needed by the time the *time frame* ends since it would be automatically extending. Additionally, if you have lots of restarting time frames, it could clutter or spam the chain very easily and cheaply (without additional tips).

### New tips start a new time frame window

We also discussed to possibility of allowing new tips to start a new *time frame* window (versus being added to currently open tip), in order to have faster reports (e.g. if the *time frame* is an hour, you could stack the tips 30 minutes apart to get reports every 30 minutes), however the technical difficulty comes into play when figuring out which *time frame* a report is for.  When setting a report *time frame*, parties should be aware that the *time frame* is the fastest that the data will be available, so if they need faster data, they should set a shorter *time frame*.  Longer *time frame* queries are expected to be slower moving data, such as prediction market answers or data queries that have a *time frame* attached (e.g. what is the ETH block header at 10:35:00 UTC).  

### Refund tip if no report

Refunds were discussed if no data was received. However, the argument against this is that the reason for no one reporting is that the tip was too low or again the data spec is not clear.  It would also be easy to spam the chain with small tips or for unsupported queryId's if a malicious actor knows that they'll get the tokens back.  Additionally, user voting power is based on tips, and more tracking would need to be put in place to avoid voting power mining. 

### All reveals done in reveal window

Data mirroring is an attack that we want to guard against, and we considered only allowing reporters to reveal during the reveal window. Under this option, all reporters wait to reveal right after the *time frame*.  The problem here is that for longer time frames, reporters will need to wait until the right block and then have a limited time (right now 2 blocks) to reveal.  If the reporter software goes down or they need to maintain a long list of reveals to submit in the future, this could be computationally expensive. To allow the reporter to decide their comfort level in being mirrored (they have to split rewards with parties mirroring them), we allow reveals at any stage. In general, large parties would not mirror smaller ones because if a large party mirrors a smaller one, the smaller party can submit a bad value and dispute the larger party right away to get back their funds.  

### No commits, only revealed data

To make it easier for data reporters to not have to be wait for the reveal window, we debated removing commits and punishing for mirroring via governance later. This would be beneficial and make our reporting time twice as fast (one vs two blocks), but the mirroring attack is very real and should be guarded against closely. If mirroring becomes an issue the whole chain could come to consensus on bad data because of mirroring. Monitoring for mirroring attacks is very difficult without risking slashing (e.g. submitting a bad price on purpose to monitor for mirrorers).

## Issues / Notes on Implementation

Note that parties can change the queryId *time frame* and it should be monitored to make sure that commit times are enough notice on a given tip.


