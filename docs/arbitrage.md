# Arbitrage Algorithm

## Bellman-Ford algorithm

- finds the shortest path from a node (i.e. vertex) to any other node on a graph
  - also, crucially, allows to **identify negative cycles in a graph**
- this can be used to identify arbitrage opportunity:
  - a directional graph is created
  - nodes represent different currencies
  - edges represent conversion between currencies
    - conversion across exchanges can only be done with the same currency
  - edges have weights based on a negated natural logarithm of the exchange rate
    - $-\ln(E_{A\to B})$ where $E$ is the exchange rate between any two currencies
- on the first iteration Bellman-Ford algorithm _relaxes_ the nodes
- if on the second iteration it gets stuck on relaxing other nodes then a negative cycle (arbitrage opportunity) is found
- based on the predecessors array the arbitrage path is determined from the negative cycle

## Fees

Conversion and withdrawal/deposit fees are retrieved and applied to the existing exchange rate data. For example, if $E_{A\to B}$ is an exchange rate then a fee is applied as $E_{A\to B}\times (1-F)$ ($F$ being a fee in the percentage form). This way all exchange rates already account for all possible fees.

There are two main types of fees:

1. **Conversion fee:** applied to all intra-exchange rates (weights) of currencies
2. **Withdrawal/deposit fee:** applied to transfers of the same currency across exchanges

Note that conversion across exchanges is only possible with the same currency (e.g. `BTC_Binance -> BTC_Kraken`).

## Other resources

- [Arbitrage using Bellman-Ford algorithm](https://www.thealgorists.com/Algo/ShortestPaths/Arbitrage)
