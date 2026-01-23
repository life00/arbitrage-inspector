# Problems and Solutions

This file presents a list of problems which were encountered during the development, as well as their solutions.

## Current (solved) problems

### 1. Performance

Performance is a mostly solved problem. As mentioned in the previously presented control flow chart, the architecture is designed to minimize the amount of time required to process the data by caching constant (e.g. currency and market lists) or semi-constant values (e.g. fees).

The Bellman-Ford algorithm itself is also very efficient, with time complexity of $O(V \times E)$ where $V$ and $E$ corresponds to the number of vertices (currencies) and edges (currency pairs). An alternative algorithm, for instance based on linear programming methods would've been significantly slower due to higher time complexity. Therefore, Bellman-Ford algorithm is preferred and scales very well as the network grows.

In practice, the code first transforms the exchanges data into inter and intra pairs and then executes the algorithm. With 2500 different assets and 6000 pairs it takes under 400 ms from fetching new data to placing market orders. Of course this is far from _perfect_ HFT speed, but such a result is already pretty impressive. It can be potentially improved by reducing the number of currencies and pairs in the network, as well as using more advanced techniques of caching part of the graph which hasn't changed and/or executing the algorithm in parallel.

### 2. Fees

Fees were another major issue which was fixed. There are two different fees: (1) exchange fees, as a share of transaction; and (2) network fees, constant amount of transacted currency when moving funds from one exchange to another. Both of these issues were fixed by making the transformation algorithm create _effective_ exchange rate (which accounts for fees) across currency pairs.

Thankfully, CCXT library provides data on exchange and network fees by default, so data accessibility wasn't an issue, however the implementation was more complicated. For exchange fees it was rather straightforward, simply applying the fee coefficient to the exchange rate (creating intra pairs). However, applying network fees was much more problematic. The issue was that network fees are constants, but the arbitrage finding algorithm can only accept exchange rate weights.

The only way to convert network fees into coefficients is to subtract them from the invested capital and divide by initial invested capital. This works, except the fees are denominated in the transaction currency itself, so it is first necessary to convert the initial capital into that transaction currency.

Despite the complexity of accounting for network fees, I have implemented a system which converts the initial capital into all possible currencies (using BFS on a nominal network), and then the transformation algorithm uses it to calculate effective inter pairs.

### 3. Liquidity (slippage)

Liquidity (aka slippage risk) is one of the most problematic issues which still was not fully addressed. In essence, the issue is caused by the fact that the bid/ask prices which exchanges provide do not have infinite liquidity. In other words, if I make a market order with a value of a million USD, I would pay significantly more than the specified bid/ask price. Therefore, the exchange rates used in arbitrage detection are not accurate. This is especially prominent in low-liquidity markets with high spreads, where a small limit order can significantly under/overvalue an asset for long periods of time, triggering arbitrage detection.

The only real way of solving this issue is by looking at the orderbook. It is possible to calculate a volume-weighted average price (VWAP) for all markets based on the amount of invested capital, and then use it instead of the bid/ask prices. However, it turns out, that the main problem is data accessibility.

CCXT library does provide a suitable method to request orderbook data for all markets (fetchOrderBooks), however it is supported by only few exchanges, and those supported exchanges do not support other functionality that arbitrage inspector needs (e.g. network and exchange fee data), therefore this method cannot be used. There is also fetchOrderBook which fetching orderbook, however it is also not useful because it is necessary to fetch orderbook for over 1000 markets per exchange at once.

Fortunately, there is another method which is supported by more exchanges and can somewhat provide data of 1000+ markets per exchange at once, and it is watchOrderBookForSymbols. It allows to create WebSocket connections and watch the orderbook of many markets at once.

Unfortunately, it isn't perfect. WebSocket connections are much more complicated to manage than simple REST API requests. In addition to that, each exchange has it's own API limits which restrict the number of markets per WS connection, number of created WS connections per minute, number of concurrent WS connections, etc. This made it incredibly difficult to maximize data retrieval, while staying within rate limits.

Despite this, I designed an architecture which perfectly integrates such a watcher into the system (see control flow chart), and I fully implemented the watch package to support concurrent watcher, which spawns exchange watchers, which spawn individual WebSocket connections. Each exchange watcher has it's own WS connection spawning configuration to stay within rate limits of the exchange.

However this workaround system has it's own limitations. Some exchanges start returning errors with too many concurrent connections (e.g. kucoin), while others start throttling or completely stopping the connection (e.g. binance). Even if it works, the provided detail of orderbook (total volume) may be insufficient to calculate actual VWAP. The current implementation cannot watch all markets on all specified exchanges. This might potentially be improved by fine tuning the configuration for each exchange, but this is not a guaranteed solution. Another issue is the efficiency. Large amount of traffic and no way of configuring it results in very high CPU usage due to JSON parsing. This may only be limited with per-exchange API configuration.

As mentioned above there is also a function fetchOrderBook which fetches market orderbooks individually. It is not ideal to fetch all orderbooks at once, however it still can be used to verify whether the detected arbitrage actually has enough liquidity. Therefore, it was implemented as a temporary solution to account for liquidity before executing the trade. The implementation concurrently executes fetchOrderBook function for all markets within the identified ArbitragePath and checks whether the expected return is still positive after computing VWAP. Such an approach does add additional latency of about 1-2 seconds, however there is no better solution.

### Others

There are a couple problems which I still did not get to fixing. Most of them will likely become obvious when testing actual trade execution.

#### Network transaction speed

Even though market orders can be performed rather instantly, transferring funds across exchanges can take a lot of time. The speed of transaction execution on different networks is not accounted in the current implementation. Therefore, even if a real multi-exchange arbitrage is detected, it is likely to disappear or significantly change by the time the full cycle of transactions is performed.

#### Transaction risks

It is essential to ensure that the trade mechanism is very well written and tested. Any bugs in the transaction making code may potentially result in complete loss of funds. In addition to that, it is necessary to implement proper fail safe mechanisms. For instance, if a transaction fails in the middle of an arbitrage cycle it is necessary to revert all transactions and/or convert back to a safe asset.

#### Full arbitrage path

While the current implementation of the algorithm can find arbitrage cycles, it cannot create a full trade execution path from the source assets to arbitrage cycle and back. Fortunately, when designing the algorithm implementation I took this into account, however it is still not fully implemented (in FindArbitrage()). After finding a viable arbitrage cycle, the engine would also have to find the shortest path from any of the source assets to an arbitrage cycle, and back.

#### Arbitrage cycle length

Due to technical reasons, Bellman-Ford algorithm doesn't detect the _best_ arbitrage cycle, but rather it detects the first cycle it finds. Therefore, it is not the best in terms of return, and it is also not the shortest one possible. Sometimes it detects cycles with over six transactions. Longer cycles may involve higher risk and longer transaction time.

To avoid that, it might be possible to somehow modify the Bellman-Ford algorithm to skip negative weight cycles with over six transactions. Alternatively, it might be possible to do post-processing on the identified arbitrage cycle, and attempt to find any shortcuts.

## Future Extensions

The current implementation can be further improved, primarily by solving the above mentioned issues, but there are also some other ideas.

### Limiting currency/market/exchange selection

The current implementation is limited in terms of performance and API rate limits because it attempts to capture all possible arbitrage cycles in all currencies/markets in specified exchanges. If it can be achieved then that would be great, however an alternative approach would be to somehow limit the selection of currencies, markets, and exchanges to maximize the chance of finding an arbitrage, while not exceeding rate limits and performance requirements.

It is quite a difficult task, however I had an idea of using some basic concepts of probability theory. Instead of defining complex selection rules which may or may not work, it is possible to monitor the market and record the currencies which tend to appear more often in the arbitrage cycle. This would require monitoring a very large set of markets for a long time to get a reliable list of currencies. Afterwards, this list of currencies can be used for actual arbitrage execution. Perhaps, there are some other methods, but this seems to be the most straightforward.

Additionally, it might possible to limit the number of watched markets by using a hybrid version of watcher. The regular fetcher method would be called once in a while, while watcher will only monitor markets with low liquidity. The CCXT ticker data structure provides information on bid/ask price volume, so watcher may only monitor markets where the bid/ask volume is below the capital amount.

### Trade execution

Trade execution has not been implemented yet in the project. It will be implemented based on stages of ArbitragePath execution (ToCycle, Cycle, FromCycle). In each stage it will perform individual transactions From and To currency. In case an error occurs, it will move the capital to the closest safe asset.

### Automation and UI

Similarly to trade execution, client automation and UI are aspects of the program which will be implemented in the future if there are good analytical results.
