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
negativeFeatures:
  - ceasefire
  - operations restored
examples:
  - energy infrastructure attacked
  - pipeline outage after conflict
labels:
  category: energy
---

# conflict_energy

Signals related to conflict impacting energy systems.