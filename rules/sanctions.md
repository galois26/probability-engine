---
name: sanctions
marketScope: [fx, commodities, equities]
directionDefault: negative
threshold: 0.65
positiveFeatures:
  - sanctions
  - export ban
  - embargo
  - restrictions
negativeFeatures:
  - sanctions lifted
  - exemptions granted
examples:
  - sanctions imposed on country A
  - export restrictions announced
labels:
  category: geopolitics
---

# sanctions

Signals related to sanctions, export controls, and embargoes.