# ADR 014: queryId time frame structure
## Authors

@themandalore

## Changelog

- 2024-03-06: initial version

## Context

For some queryId's, it will take longer than one block to submit data after a tip.  Examples include manual questions (e.g. "who is the president") or data without API's (e.g. prediction market answers, some US govt data).  To solve this, we set a report time frame for each query type when registering.  For example, a string question could set the time frame to take up to 5 hours for reports, giving all reporting parties a chance to see the question and formulate a response.  This ADR specifies how we are to handle the timing of tips, reporting commits, reveals, and restarts in case of unreported data.  


The proposed solution is that the report time frame starts on the block after the tip is recieved.  If there is another tip that happens during the report time frame, it is added to the original tip.  If no data is reported throughout the time frame, a new report time frame must be opened by another tip (it can be as small as one loya), and the previous unclaimed tip is added to the new tip.  

The commit reveal cycle is still intact and reveals can be anytime before reveal window is over, even in the commit phase.  Note that this doesn't prevent mirroring if revealed during the commit phase, but it allows you to wait to reveal if you feel that mirroring is occuring. 


## Alternative Approaches

### time frame starts when first commit happens

One approach would be to have shorter report time frames, but the window doesn't start right after the tip, but rather on the first report.  This would allow for queries that are not seeking consensus, to just have a tip with a short time frame, and then once one party reports, it is basically over and handled by disputes.  The issue we had with this is that it unties the time frame from the tip.  Usually if you tip, you want some certainty around when you get the data.  It also gives the false impression to the tipper that they should expect the data in some time frame.  If you tip for a manual query and it has a time frame of one hour, you would expect it to be reported in one hour.  The time frame in our mind is tied to how long it should be expected to take reporters to get the data.  

### time frame restarts if no one reports

One option would be to just restart the time frame if no one reports.  This would work, but no one might be reporting because the tip is too low.  It's also unclear if the report is still even needed if the time frame ends.  Additionally if you have lots of restarting time frames, it could clutter or spam the chain very easily. 

### refund tip if no report

I think the argument against is that the reason no one reported is that the tip was too low.  It would also be easy to spam the chain with small tips or for unsupported queryId's, if you know that you'll get your tokens back.  Additionally, voting power is given on tips, and more tracking would need to be put in place in regards to refunds.  

### all reveals done in reveal window

The problem here is that for longer time frames, reporters will need to wait until the right block and then have a limited time (right now 2 blocks) to reveal.  If they go down or need to maintain a long list of reveals to submit in the future, this could be computationally expensive.  In order to just allow the reporter to decide their comfort level in being mirrored (they have to split rewards with parties mirroring them), we allow reveals at any stage.  Not to mention, that large parties will not mirror smaller ones.  If a large party mirrors a smaller one, the smaller party can submit a bad value and dispute the larger party right away to make back their funds.  

### new tips start a new window

New tips could start a new window, in order to have faster reports (e.g. if the time frame is an hour, you could stack the tips 30 minutes apart to get reports every 30 minutes), however the technical difficulty comes into play when figuring out which time frame a report is for.  When setting a report time frame, parties should be aware that the time frame is the fastest that the data will be available, so if they need faster data, they should set a shorter time frame.  Longer time frame queries are expected to be slower moving data, such as prediction market answers or data queries that have a time frame attached (e.g. what is the ETH block header at 10:35:00 UTC).  

### no commits, only revealed data

We had debated just throwing out the commits and you could just punish it via governance later.  This would be beneficial and make our reporting time twice as fast (one vs two blocks), but the mirroring attack is very real and should be guarded against more closely.  Not to mention that monitoring for mirroring attacks is very difficult without risking slashing (e.g. submitting a bad price). 

## Issues / Notes on Implementation

Note that parties can change the queryId time frame and it should be monitored to make sure that commit times are enough notice on a given tip
