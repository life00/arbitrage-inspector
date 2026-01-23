# Fetch Package

The fetch package is responsible for retrieving the price, fee, and orderbook data from various exchanges.

## Exchange and currency selection

The program allows to pass a list of exchanges and a list of currencies as input for retrieving the data. The input combination of exchanges and currencies is further validated.

The validation is done by ensuring that (1) all specified exchanges support necessary API functionality (see [CCXT library](#ccxt-library) and ./trade.md); (2) all markets (currency pairs) are available in all exchanges; and (3) all markets have necessary characteristics (e.g. being a spot market, using percentage fees)

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

## Orderbook data

Contains the orderbook with individual prices and corresponding volume amounts for bid and ask sides of the market. It is used to calculate the volume-weighted average price (VWAP) based on invested capital to properly account for liquidity.

Orderbook data is used to verify that the ArbitragePath has sufficient liquidity to perform a profitable arbitrage.

## CCXT library

1. `fetchCurrencies()`
   - fetch info about all currencies for an exchange
2. `fetchMarkets()`
   - fetch info about all markets for an exchange
3. `fetchTickers()`
   - fetch most up to date bid and ask prices for a symbol
4. `fetchOrderBook()`
   - fetch most up to date orderbook of a specified market
