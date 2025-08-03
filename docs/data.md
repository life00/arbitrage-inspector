# Data Retrieval and Transformation

## Fees

Conversion and withdrawal/deposit fees are retrieved and applied to the existing exchange rate data. For example, if $E_{A\to B}$ is an exchange rate then a fee is applied as $E_{A\to B}\times (1-F)$ ($F$ being a fee in the percentage form). This way all exchange rates already account for all possible fees.

There are two main types of fees:

1. **Conversion fee:** applied to all intra-exchange rates (weights) of currencies
2. **Withdrawal/deposit fee:** applied to transfers of the same currency across exchanges

Note that conversion across exchanges is only possible with the same currency (e.g. `BTC_Binance -> BTC_Kraken`).
