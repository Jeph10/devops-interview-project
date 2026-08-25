# Cost Scenario Analysis (Task 5)

> This is the deliverable for Task 5. **If you can access the data, provide specific numbers and queries. If you cannot, state your key assumptions, the data you would use to validate them, and how your conclusions would change if they proved false.** Tie each conclusion to the data and validation method instead of listing generic cost-cutting ideas. Aim for no more than 600 words.

## 1. Establish the Baseline

Which data would you inspect first to decide whether the increase is expected? How would you separate usage growth from lower efficiency or waste? Which unit would you use, such as cost per event, request, or daily active user?

## 2. Attribution Approach (Observe → Hypothesize → Verify)

Describe one investigation chain that breaks the 30% increase down to specific sources. Explain which data you would use at each step:

- which dimensions you would start with, such as service, usage type, account, Region, or tag, and how you would attribute the untagged 40%;
- how you would determine whether increases across several services share one root cause;
- which confounding factors you would rule out at each step, such as billing-period length, RI or Savings Plan amortization, and one-time charges.

## 3. Actions and Trade-offs

Choose one constraint. Give an actionable first step, a fallback option, and what you would give up:

- you want to use committed-use discounts such as RIs or Savings Plans, but the event table is scheduled to migrate next quarter;
- reducing volume at the source depends on the app team, but that team has no capacity this quarter;
- last month you expected costs to fall, but this month they reached a new high, and you must set honest expectations with your leader.

Separate one-time point optimizations from root-cause governance or long-term controls, and explain how you would prevent the cost from returning.

## 4. Supporting Experience (Optional)

Describe one real cost optimization, including the measured before-and-after result and how you confirmed that your change caused the improvement rather than a coincidental change in business volume.
