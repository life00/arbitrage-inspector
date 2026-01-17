# Engine Package: Algorithms

The engine package provides various functions to execute key algorithms for data analysis.

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

## Super-source node

In order to maximize the chance of finding an arbitrage, the algorithm allows to have the capital stored in several different currencies. It will work as long as the balance on all specified currencies is equal or above the initial capital used for the arbitrage execution.

It works by creating a _super-source_ node in a graph which is connected (with weight `1`) to all currencies with sufficient balance. The Bellman-Ford algorithm starts the search from the super-source node, and then it allows to trace back the arbitrage cycle to the closest initial capital currency.

## Fees

Conversion and withdrawal/deposit fees are retrieved and applied to the existing exchange rate data. There are two main types of fees:

1. **Conversion fee:** applied to all intra-exchange rates (weights) of currencies
2. **Withdrawal/deposit fee:** applied to transfers of the same currency across exchanges

In case of conversion fees the following formula is used: $E_{A\to B}\times (1-F)$ ($E_{A\to B}$ being the nominal bid/ask price, and $F$ being a fee in the percentage form).

To properly account for withdrawal/deposit fees it was necessary to find the denomination of the initial capital in all possible currencies. To do that I used breadth-first search (BFS) algorithm to find conversion rates from initial capital currency to all other assets. This then allows to compute the conversion weights for all cross-exchange transactions using the following formula: $E_{A_1\to A_2}=\frac{I_{A_1}-F_{A_1}}{I_{A_1}}$ ($I_A$ being the initial capital denominated in currency $A$). Note that conversion across exchanges is only possible with the same currency (e.g. `BTC_binance -> BTC_kraken`).

This way all exchange rates already account for all possible fees.

## Other resources

- [Arbitrage using Bellman-Ford algorithm](https://www.thealgorists.com/Algo/ShortestPaths/Arbitrage)
