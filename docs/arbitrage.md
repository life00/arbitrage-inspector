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

## Other resources

- [Arbitrage using Bellman-Ford algorithm](https://www.thealgorists.com/Algo/ShortestPaths/Arbitrage)
