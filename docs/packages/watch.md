# Watch Package

The watch package implements the complex process of establish many concurrent WebSocket connections to multiple exchanges and updating the exchanges data structure with prices which account for liquidity.

It is currently not used in the main version of Arbitrage Inspector because it cannot easily handle monitoring thousands of different markets due to API limitations. The current implementation uses `fetchOrderBook()` method by checking liquidity of the identified arbitrage as a temporary solution.

## Architecture

The architecture of the watch package is very straightforward. The client can access watch package functionality through an instance of `Watcher` struct. That _watcher_ struct accepts the exchanges and clients data structures, based on which it determines the markets to watch. It then creates multiple _exchange watchers_ which in turn create individual workers that are responsible for establishing the WebSocket connections and saving the received data to cache. Each exchange watcher has a specific WebSocket creation configuration to match the required rate limits of the exchange.

When requested, the watcher will ask all exchange watchers to calculate volume-weighted average prices (VWAP) based on saved cache, update exchanges data structure, and clear out the cache. This cycle repeats based on clients configuration.

## CCXT library

1. `watchOrderBookForSymbols()`
   - establish a WebSocket connection which provides continuous orderbook data stream for multiple markets
1. `close()`
   - closes all exchange connections
