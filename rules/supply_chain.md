---
name: supply_chain
marketScope: [metals, commodities, industrials]
directionDefault: negative
threshold: 0.60
positiveFeatures:
 # - minerals
  - supply chain
  - supply shortages
 # - disruption
 # - shortages
 # - bottleneck
  - industrial metals
  - shipment delay
  - shipment disruption
  - freight disruption
  - jet fuel shortage
  - marine fuel oil
  - aviation fuel shortage
  - shipping disruption
  - cargo backlog
  - port congestion
  - port closure
  - port strike
  - customs delay
  - refinery outage
  - refinery shutdown
  - terminal shutdown
  - depot disruption
  - pipeline outage
  - pipeline disruption
  - fuel crisis
  - fuel shortage
  - fuel shortages
  - suez canal
  - suez canal blockage
  - panama canal
  - panama canal restrictions
  - strait of hormuz
  - hormuz shipping
  - red sea shipping
  - bab el mandeb
negativeFeatures:
  - supply restored
  - bottleneck eased
  - operations restored
  - congestion eased
weightedFeatures:
  - token: shipping
    weight: 0.20
  - token: shutdown
    weight: 0.30
  - token: shortage
    weight: 0.25
  - token: shortages
    weight: 0.25
  - token: refinery
    weight: 0.25
  - token: pipeline
    weight: 0.25
examples:
  - minerals export restrictions announced
  - supply chain disruption increases
  - port congestion delays metal shipments
  - refinery outage disrupts fuel supply
labels:
  category: trade
---
