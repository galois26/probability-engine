---
name: crypto_market
marketScope: [crypto]
directionDefault: negative
threshold: 0.60
positiveFeatures:
  - exchange outage
  - exchange disruption
  - trading suspended
  - withdrawals suspended
  - deposits suspended
  - network congestion
  - blockchain congestion
  - transaction delays
  - settlement delays
  - validator outage
  - rpc outage
  - bridge exploit
  - protocol exploit
  - smart contract exploit
  - stablecoin depeg
  - liquidity crisis
  - liquidation cascade
  - market maker withdrawal
negativeFeatures:
  - trading resumed
  - withdrawals resumed
  - deposits resumed
  - network restored
  - congestion eased
  - peg restored
examples:
  - exchange outage halts withdrawals
  - stablecoin depeg triggers liquidation cascade
  - network congestion delays transactions
  - protocol exploit disrupts market activity
labels:
  category: crypto
---