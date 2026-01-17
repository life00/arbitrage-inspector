# Process Flow List

This file presents a conceptual process flow of Arbitrage Inspector.

## Process

- **initialization**
  - **client**: define initial config
  - **fetch**: initialize exchanges data structure
  - **transform**: creates asset index
- **periodic update**
  - **fetch**: update exchange data structure (prices, exchange fees, network fees)
  - _balances_
    - **transform**: create nominal intra-exchange pairs (no fees)
    - **transform**: create nominal inter-exchange pairs (no fees)
    - **engine**: find balances of all assets based on initial capital
      - create a nominal graph
      - run bellman-ford
      - use resulting weights to convert nominal value of reference asset to all source assets
    - **transform**: update balances of SourceAssets assets in config
    - **trade**: ensure there is sufficient balance in all SourceAssets
  - _inter pairs_
    - **transform**: create effective intra-exchange pairs (with regular bid/ask prices)
    - **transform**: create effective inter-exchange pairs\n(with 1 USD network fees, denominated in ReferenceAsset)
    - **engine**: find balances of all assets
    - **transform**: create actual inter-exchange pairs (based on found balances)
- **update loop**
  - _client_
    - **client**: wait some time
    - **watch**: call watcher to update data
    - **transform**: create actual effective inter-exchange pairs
    - **engine**: search for (reasonable) arbitrage and find full ArbitragePath (from/to SourceAssets)
    - **trade**: verify that the arbitrage is profitable
  - _watcher_
    - **watch**: initialize orderbook watcher (establish WS connections)
    - **watch**: cache all received orderbook data
    - **transform** calculate effective prices (from orderbook data)
      - based on capital requirements (using nominal balances of all assets)
    - **watch**: update `exchange` data structure
- **trade**
  - _verifier (WIP)_
    - **fetch**: fetch orderbook data of markets in ArbitragePath
    - **transform**: calculate effective prices
    - **trade**: check if arbitrage is still profitable
  - **trade**: todo

## Data Updates

- orderbook watcher
  - continuously watching for changes in orderbook for all markets in all exchanges
  - requires specialized websocket spawning process for each exchange to avoid API rate limits
  - `watchOrderBooks()`
- orderbook fetcher
  - concurrently fetching orderbook for markets within arbitrage path
  - checks if the arbitrage is still possible, accounting for liquidity
  - multiple `fetchOrderBook()` concurrently
- price data
  - bid/ask prices
  - `fetchTickers()`
- currency data
  - network fees
  - `fetchCurrencies()`
- market data
  - exchange fees
  - `fetchMarkets()`
