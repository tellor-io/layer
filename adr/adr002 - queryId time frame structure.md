# ADR 002: QueryId time frame structure

## Authors

@themandalore
@brendaloya

## Changelog

- 2024-03-06: initial version
- 2024-04-02: clarity
- 2024-04-05: clarity
- 2024-08-03: clean up
- 2024-10-07: change to blocks
- 2024-10-07: commit/reveal considered

## Context

For some queryId's, it will take longer than one block to submit data after a tip.  Examples include manual questions (e.g. "who is the president") or data without API's (e.g. prediction market answers, some US govt data).  To solve this, we set a report *time frame* for each query type when registering.  For example, a string question could set the *time frame* to take up to 5 hours for reports, giving all reporting parties a chance to see the question and formulate a response.  This ADR specifies how we are to handle the timing of tips, reporting time frames, and restarts in case of unreported data.  

The proposed solution is that the report *time frame* starts on the block after the tip is received.  If there is another tip that happens during the report *time frame*, it is added to the original tip.  If no data is reported throughout the *time frame*, a new report *time frame* must be opened by another tip (it can be as small as one loya), and the previous unclaimed tip is added to the new tip.  

Note that the *time frame* is specified in number of blocks within the registry.  This is so there is no wasted time when reporting; aggregation of reports is done in `EndBlock` on the block that the *time frame* expires. 

## Alternative Approaches

### Time frame starts when first report happens

One approach would be to have shorter report time frames where the window doesn't start right after the tip, but rather on the first report. This would allow for queries that are not seeking consensus to just have a tip with a short *time frame*, and then once a party reports, it is basically over and handled by disputes. The issue we had with this is that it unties the *time frame* from the tip.  Usually if you tip, you want some certainty around when you get the data. It also gives the false impression to the tipper that they should expect the data in some *time frame*.  If you tip for a manual query and it has a *time frame* of one hour, you would expect it to be reported in one hour.  The *time frame* concept is tied to how long it should be expected to take reporters to get the data.  If longer is needed, then the data spec itself is just likely not well defined or supported and it should be updated/changed.

### Time frame restarts if no one reports

Another option considered was to restart the *time frame* if no reports are received.  This could work, but no one might be reporting because the tip is too low or the question is vague.  It's also unclear if the report would still be needed by the time the *time frame* ends since it would be automatically extending. Additionally, if you have lots of restarting time frames, it could clutter or spam the chain very easily and cheaply (without additional tips).

### New tips start a new time frame window

We also discussed to possibility of allowing new tips to start a new *time frame* window (versus being added to currently open tip), in order to have faster reports (e.g. if the *time frame* is an hour, you could stack the tips 30 minutes apart to get reports every 30 minutes), however the technical difficulty comes into play when figuring out which *time frame* a report is for.  When setting a report *time frame*, parties should be aware that the *time frame* is the fastest that the data will be available, so if they need faster data, they should set a shorter *time frame*.  Longer *time frame* queries are expected to be slower moving data, such as prediction market answers or data queries that have a *time frame* attached (e.g. what is the ETH block header at 10:35:00 UTC).  

### Refund tip if no report

Refunds were discussed if no data was received. However, the argument against this is that the reason for no one reporting is that the tip was too low or again the data spec is not clear.  It would also be easy to spam the chain with small tips or for unsupported queryId's if a malicious actor knows that they'll get the tokens back.  Additionally, user voting power is based on tips, and more tracking would need to be put in place to avoid voting power mining. 

## Commit / reveal process

A data commit/reveal process was considered to avoid data mirroring (when reporters copy and report other reporters’ data instead of pulling the data and doing the aggregation themselves). The official value is the aggregate of all reports submitted on-chain and mirroring can influence the official value. Implementing this process significantly increased the time needed to finalize the official value.  For now, mirroring will be monitored for, and disputes and the social layer can be used if suspected.

## Issues / Notes on Implementation

Note that parties can change the queryId *time frame* in the registry and it should be monitored to make sure that reporting times are enough notice on a given tip.


