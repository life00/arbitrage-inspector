# Data Retrieval and Transformation

This code module is responsible for retrieving the price and fee data from various exchanges using `exchange.go` and transforming it into an appropriate format.

## Exchange and currency selection

The program allows to pass a list of exchanges and a list of currencies as input for retrieving the data. The input combination of exchanges and currencies is further validated.

The validation is done by ensuring that (1) all specified exchanges support necessary API functionality (see [CCXT library](#ccxt-library) and ./trade.md); (2) all markets (currency pairs) are available in all exchanges; and (3) all markets have necessary characteristics (e.g. being a spot market, using percentage fees)

However, if no input is provided, the data management module will randomly generate the exchange and currency combination in accordance with previously mentioned technical requirements. The maximum number of exchanges and currency pairs may also be specified in the input. Additionally, it is possible to specify a list of exchanges/currencies that must be included or excluded from the exchange network. If the number of input exchanges/currencies is less than the required number of exchanges/currencies, then additional exchanges/currencies will be randomly generated. It is also possible to export the randomly generated exchanges/currencies combination for further use.

Additionally, it will be possible to continuously run the arbitrage algorithm on randomly sampled exchange and currency pairs. This would allow to collect data on most common exchanges/currencies that are present in arbitrage paths to create a ranking system for exchanges/currencies to further enhance the exchange/currency selection system.

In general, it seems that most of the markets (currency pairs) are connected with stablecoins, i.e. conversion between various alt coins is only possible through stablecoins. They include: USDC, USDT, FDUSD, BUSD, TUSD, etc. In addition to that, regular currencies are also commonly connecting markets, including: USD, TRY, EUR, JPY, etc. Otherwise, the remaining connecting currencies are the most popular cryptocurrencies, such as BTC, ETH, and BNB. To achieve the best results in triangular arbitrage it might be better to avoid using any of the stablecoins or regular currencies in the arbitrage network.

## Price data

The latest price data is retrieved from all the exchanges with bid and ask price for each currency. Both bid and ask price will be used as weights for corresponding side of the transaction.

For example, if there is data for ticker "ETH/BTC", then for the conversion from ETH $\to$ BTC the bid price is used ($E_{\text{bid}}$), but if it's from BTC $\to$ ETH then the inverse ask price is used ($(E_{\text{ask}})^{-1}$).

## Fee data

There are two main kinds of fees involved:

1. **deposit/withdraw fee**: usually a flat fee dependent on a selected cryptocurrency network
   - the program selects the cheapest network which is commonly available across all exchanges
2. **conversion fee**: usually a percentage fee for the entire exchange, but there may be difference across _markets_
   - two kinds of fees:
     - market-taker, which are usually higher
     - market-maker, which are usually lower

## CCXT library

1. `fetchCurrencies()`
   - fetch info about all currencies for an exchange
1. `fetchMarkets()`
   - fetch info about all markets for an exchange
1. `fetchTickers()`
   - fetch most up to date bid and ask prices for a symbol
1. `fetchDepositWithdrawFees()`
   - fetch deposit/withdrawal fees for an exchange
1. `fetchTradingFees()`
   - fetch trading fees for an exchange
