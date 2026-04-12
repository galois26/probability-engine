---
name: conflict_energy
marketScope: [oil, gas, energy]
directionDefault: negative
threshold: 0.60
positiveFeatures:
  - conflict
  - energy infrastructure
  - attacked
  - pipeline
  - refinery
  - pipeline outage
  - refinery fire
  - refinery shutdown
  - fuel depot attack
  - terminal attack
negativeFeatures:
  - ceasefire
  - operations restored
  - output restored
examples:
  - energy infrastructure attacked
  - pipeline outage after conflict
  - refinery fire disrupts output
labels:
  category: energy
---