# Arbitrage Inspector

Arbitrage inspector is a trading/analysis bot which identifies triangular and multi-exchange arbitrage in cryptocurrency exchanges. It automatically accounts for network and exchange fees, as well as liquidity constraints (aka slippage risk). It is using the CCXT library to interact with exchange API's.

## Objective

The primary objective of this project is to identify and exploit real arbitrage opportunities within live financial environments. By combining market theory with sophisticated algorithmic analysis, the project pushes the boundaries of automated trading to implement a viable solution. The cryptocurrency ecosystem was selected for its transparency and accessibility compared to traditional finance, however the same algorithms can be implemented in traditional financial markets.

## Achievements

Arbitrage inspector is able to fetch the most up-to-date price and fee information and find triangular arbitrage opportunities across multiple exchanges while accounting for all exchange fees, network fees, and liquidity. The fetch version is able to find real arbitrage opportunities with capital below $500, especially during periods of volatility. The watch version is able to continuously monitor markets, however it is limited in scale.

## Limitations

The main limitation is accessibility of data, specifically orderbook data. CCXT does provide functions to retrieve orderbook data, however due to various reasons they are not suitable to fetch orderbook for thousands of different markets. The current workaround is to use `fetchOrderBook()` method and verify liquidity only after the arbitrage is identified. There are other limitations, but this is the main one. See `./docs/other/problems.md` for more details.

## Usage

The current implementation only analyzes the market and does not execute any transactions. There are two different versions: fetcher and watcher. Fetcher periodically gets ticket data, while watcher continuously gets orderbook data.

Watcher implementation is limited by the number of markets it can watch simultaneously due to API restrictions, therefore it is not able to find arbitrage. Fetcher is able to find arbitrage, however the effectiveness and speed of the detection algorithm is limited (see `./docs/other/problems.md` for details).

The fetcher version is in the `master` branch, while the watcher version is in the `feature/watcher` branch. To run Arbitrage Inspector, simply checkout to the corresponding branch and run the following command:

```sh
go run ./cmd/arbi/*
```

Additionally, you may configure the application inside of `./cmd/arbi/main.go` in the `initialization()` function.

## Control flow

The project was implemented based on a conceptual control flow shown in the following diagram:

<details>
  <summary>Process control flow</summary>
  <img src="./docs/process/process.png" alt="Process control flow">
</details>
  
The control flow aims to minimize the required time from data retrieval to trade execution. I believe it scales well with the project structure and performance.

## Project structure

The project is following separation of concerns based on functionality. The following is the project layout:

```c
arbitrage-inspector
├── cmd // clients
│   ├── arbi // main CLI client
│   └── tester // small testing client
├── docs // extra documentation
├── go.mod
├── go.sum
├── internal // packages
│   ├── engine // main algorithms
│   ├── fetch // RESP API data retrieval
│   ├── models // data structures
│   ├── trade // trade execution
│   ├── transform // data transformation
│   └── watch // WS API data watching
├── makefile
├── readme.md
├── todo.md
└── *.json // data cache for testing
```

## Documentation

Further documentation can be found in the `./docs/` directory. It includes information about internal packages, process execution, and other technical details.
