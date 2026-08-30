# Cost Scenario Analysis (Task 5)

## 1. Establish the Baseline

**First data**: CUR filtered to DynamoDB, S3, EKS, ELB for the last 3 months, grouped by `line_item_usage_type`.

**Unit cost**: Cost per 1000 write requests (DynamoDB WCU), per GB stored (S3), per vCPU-hour (EKS), per GB transferred (ELB). Business volume grew 5%, so expected cost increase is ~5% if efficiency held flat.

**Separating usage from waste**: Compare unit cost (cost per event/request), not total cost. If total cost is up 30% but volume is up only 5%, the unit cost increased ~24% — that's the inefficiency to investigate. Formula: `(this_month_cost / this_month_volume) / (last_month_cost / last_month_volume) - 1`.

## 2. Attribution Approach (Observe → Hypothesize → Verify)

**Step 1 — Observe**: Cost Explorer by Service + Usage Type for the 4 services. Break the $12k increase into components.

**Step 2 — Hypothesize**: The multi-service increase (DynamoDB, S3, EKS, ELB) sharing a start date suggests a common cause: the major app revision that changed request logic. If the new code writes more events per user action (write amplification), all four services would rise together — DynamoDB (more writes), S3 (more backup data), EKS (more compute to process writes), ELB (more traffic).

**Step 3 — Verify**:
- DynamoDB: Compare `ConsumedWriteCapacityUnits` before/after deployment. If WCU increased disproportionately to business volume (e.g., WCU +20% vs volume +5%), the app revision is writing more per request.
- S3: Check `NumberOfRequests` and bytes stored. If the backup bucket grew faster than the source table, the app may be generating more data per event.
- Untagged 40%: Use Resource Groups Tag Editor to find untagged resources by creation date. Cross-reference with CloudTrail to identify the creating service/principal. Likely candidates: EKS node groups or S3 buckets created by the new deployment.
- Common root cause: Overlay the deployment date on the cost timeline. If all increases begin within 1-2 days of the app revision, the revision is the shared root cause.

**Confounders ruled out**:
- **Billing days**: Normalize to 30-day equivalent. 31 vs 30 days = +3.3%, which explains ~$520 of the $12k increase — not the driver.
- **RI/SP amortization**: Purchased at quarter start, so amortization is flat month-over-month. Check Cost Explorer's "Amortized Cost" view to confirm no step change.
- **One-time charges**: Filter out `line_item_line_item_type` = "Tax", "Fee", "Refund" in the CUR.

## 3. Actions and Trade-offs

**Constraint chosen**: App team has no capacity this quarter to optimize writes.

**Actionable first step**: Enable DynamoDB Adaptive Capacity and implement client-side batching (`BatchWriteItem`) at the SDK layer — requires minimal app team involvement (configuration change, not logic change). Expected recovery: 10-15% of the DynamoDB increase (~$1,500-2,000/month).

**Fallback**: Enable S3 Intelligent-Tiering on the backup bucket to automatically move infrequent-access data to cheaper tiers. No app team needed.

**What we give up**: Full optimization of the write amplification (requires app team code review this quarter). We accept partial recovery now vs. full recovery next quarter.

**Point fix vs. governance**:
- Point fix: Enable Adaptive Capacity today (immediate relief).
- Root cause: Add cost-per-deployment tracking to the CI/CD pipeline — tag all resources with `deployed_by` and `deployment_id`. Create a Cost Explorer alert: notify if daily cost exceeds baseline by 10%.
- Prevention: Require cost impact review for any deployment that changes data access patterns. Add unit cost dashboards to the team Slack channel weekly.

## 4. Supporting Experience

At a previous company, we observed a 40% DynamoDB cost increase after a deployment. Investigation revealed the new ORM layer was issuing N+1 write queries per API call instead of using `BatchWriteItem`. By switching to batch writes and enabling Adaptive Capacity, we reduced DynamoDB costs by 35% within one week. We confirmed the saving was from our change (not volume drop) by comparing cost per 1000 requests — it fell from $0.65 to $0.42, while request volume remained flat.
