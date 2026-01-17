# Models Package

The models package provides data structure definitions for the entire program.

## Exchanges structure

Exchanges structure represents a semantic-based hierarchy of exchange related data. It consists of different exchanges, which contain markets and currencies, which represent assets with various attributes.

## Pairs structure

Pairs structure is derived from the exchanges structure to create a network-like structure of currency connection which can be easily fed into an advanced graph/network algorithm.

## Arbitrage path structure

Arbitrage path structure represents a sequence of conversion steps across currencies in order to execute a successful arbitrage.

## Others

There are other supplementary data types which are used for configuration and other type classification.
