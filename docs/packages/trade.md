# Trade Package

The trade package is responsible for evaluating arbitrage feasibility and safely executing the trades. It is not implemented fully.

## Cross-exchange transfer of funds

In order to transfer funds across exchanges it is possible to use the `withdraw()` method in a source exchange with an address of `fetchDepositAddress()` of the destination exchange.

## CCXT library

- `createOrder()`
  - create an order to make a trade
- `fetchBalance()`
  - fetch current balance in an account
- `withdraw()`
  - withdraw funds to a specified address
- `fetchDepositAddress()`
  - fetch deposit address of the exchange account
