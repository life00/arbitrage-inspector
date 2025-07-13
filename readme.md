# Arbitrage Inspector

## 1. Data retrieval

The program receives an array of exchanges and cryptocurrencies. Then it retrieves conversion price data and up to date fee information from various API's.

## 2. Arbitrage identification

The program receives all cryptocurrency conversion information and an input cryptocurrency with a given balance. Using a clever algorithm based on graph theory (e.g. Bellman-Ford algorithm) the program identifies possible arbitrage opportunities while accounting for conversion, withdrawal, and other fees. It returns the top 5 arbitrage opportunities if such exist.

### Alternative algorithms

Alternatively, it is possible to use other algorithms. The naive approach would be to iterate through all possible combinations of conversions to identify the best conversion paths, however this is extremely inefficient. Other comprehensive option involves using linear programming (aka function optimization) methods to construct a function with given constrains and identify the objective. Additionally, other advanced machine learning algorithms can be utilized.

The graph theory method presents the most classic and straightforward solution to the problem, furthermore, new research papers may be used to further improve the algorithm. Therefore, it is selected as the ideal arbitrage identification algorithm.

## 3. Trade execution

The program receives the most profitable arbitrage path. Using the necessary API's it executes the trade as soon as possible to avoid any price slippage risk or other delay. It repeats the conversion cycle until the arbitrage opportunity is no longer present.

## 4. Application interface

The program may be accessible through CLI application, GUI application, website, messenger bot, or other medium. It provides a simple control interface and reports top 5 identified arbitrage opportunities and results from the trades.

## Technical note

Whenever possible, all processes must be executed in parallel to maximize the speed of processing and reduce the price slippage risk.
