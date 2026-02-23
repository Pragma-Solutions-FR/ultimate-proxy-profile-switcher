# Ultimate-Proxy.com profile switcher

An example profit-switching daemon for [ultimate-proxy.com](https://ultimate-proxy.com).

In its current state it monitors the profitability of several coins on the [Kryptex](https://pool.kryptex.com) mining pool and automatically switches all your workers to the most profitable coin by reassigning them to the corresponding profile on Ultimate Proxy. **By leveraging [ultimate-proxy.com](https://ultimate-proxy.com) features your miners never loose a single second of mining time.**

---

## How it works

1. Every `interval` seconds the daemon calls the **Kryptex Pool API** to fetch live daily revenue and exchange rates for every configured coin.
2. It calls the **Ultimate Proxy API** to get your current aggregate hashrate (1 h average) so the revenue calculation reflects your real miners.
3. It sorts coins by daily revenue in your chosen fiat currency and picks the best one.
4. If the best coin changed, it bulk-assigns all your workers to the matching Ultimate Proxy profile and sets that profile as the default for new connections.
5. It prints a profitability table, historical averages, and a live ASCII chart in the terminal.
6. History is persisted to disk (JSON) so averages survive restarts.

```
  Profitability Report — 2025-01-15 14:32:01  ⚡ 450.00 KH/s
────────────────────────────────────────────────────────────────────────────────────
  Rank  Coin        Daily (coin)       Daily (EUR)       BTC/MH/Day      Price (USD)
────────────────────────────────────────────────────────────────────────────────────
  1     ★ XMR       0.00341200        0.54820000    0.0000052341       161.820000
  2       XTM       0.00000120        0.48100000    0.0000047210         0.000400
  3       ZEPH      0.12300000        0.41230000    0.0000040100         0.033500
  4       SAL       ...
```

---

## Requirements

- Go 1.24+ (to build from source)
- An [ultimate-proxy.com](https://ultimate-proxy.com) account with at least one profile per coin
- Miners already connected and routing through Ultimate Proxy

---

## Quick start

### 1. Build

```bash
git clone https://github.com/Pragma-Solutions-FR/ultimate-proxy-profile-switcher
cd ultimate-proxy-profile-switcher
go build .
```

Or download a pre-built binary from the releases page.

### 2. Configure

Copy the example config and fill in your values:

```bash
cp config.example.yaml config.yaml
```

```yaml
# Kryptex Pool API (default, no need to change)
kryptex_base_url: "https://pool.kryptex.com/api/v1"

# Ultimate Proxy
proxy_api_key: "up_k_xxxxxxxxxxxxxxxxxx"   # https://ultimate-proxy.com/settings/api-keys
proxy_algorithm: "randomx"                 # algorithm your miners use

# Display currency
fiat_currency: "EUR"

# How often to check profitability (seconds)
interval: 300

# Fallback hashrate if the API returns no data (H/s)
default_hashrate: 150000

coins:
  - ticker: "XMR"
    profile_id: "REPLACE_WITH_PROFILE_ID"

  - ticker: "XTM"
    revenue_ticker: "XTM_rx"   # some coins use a different ticker on the revenue endpoint
    profile_id: "REPLACE_WITH_PROFILE_ID"

  - ticker: "ZEPH"
    profile_id: "REPLACE_WITH_PROFILE_ID"

  - ticker: "SAL"
    profile_id: "REPLACE_WITH_PROFILE_ID"
```

**Where to find profile IDs:** go to [ultimate-proxy.com/profiles](https://ultimate-proxy.com/profiles), click the action icon next to a profile, and choose *Copy ID*.

**Where to find your API key:** go to [ultimate-proxy.com/profiles](https://ultimate-proxy.com/settings/api-keys), the key need the following scopes: workers read, workers write, profiles write

### 3. Run

```bash
./ultimate-proxy-profile-switcher -config config.yaml
```

---

## CLI flags


| Flag            | Default       | Description                                                 |
| --------------- | ------------- | ----------------------------------------------------------- |
| `-config`, `-c` | `config.yaml` | Path to the YAML config file                                |
| `-dry-run`      | `false`       | Print the profitability table without switching any workers |
| `-once`         | `false`       | Run a single cycle and exit immediately                     |

**Dry-run example** (useful to verify your setup without touching workers):

```bash
./ultimate-profile-switch -c config.yaml -dry-run
```

**Single shot** (useful in a cron job):

```bash
./ultimate-profile-switch -c config.yaml -once
```

---

## Configuration reference


| Key                      | Required | Default                           | Description                                                                   |
| ------------------------ | -------- | --------------------------------- | ----------------------------------------------------------------------------- |
| `proxy_api_key`          | yes      | —                                | Ultimate Proxy API key                                                        |
| `proxy_algorithm`        | yes      | —                                | Algorithm your miners use (`randomx`, `kawpow`, …)                           |
| `kryptex_base_url`       | no       | `https://pool.kryptex.com/api/v1` | Kryptex Pool API base URL                                                     |
| `fiat_currency`          | no       | `USD`                             | Currency for revenue display (`USD`, `EUR`, `GBP`, …)                        |
| `interval`               | no       | `300`                             | Seconds between profitability checks                                          |
| `default_hashrate`       | no       | `1000`                            | Fallback hashrate in H/s used when the API returns no live data               |
| `history_file`           | no       | `profswitch_history.json`         | Path where history snapshots are persisted                                    |
| `coins[].ticker`         | yes      | —                                | Coin ticker as used by Kryptex for rate lookup (e.g.`XMR`)                    |
| `coins[].profile_id`     | yes      | —                                | Ultimate Proxy profile ID to activate when this coin is best                  |
| `coins[].revenue_ticker` | no       | same as`ticker`                   | Override ticker used on the Kryptex`/daily-revenue/` endpoint (e.g. `XTM_rx`) |

---

## Extending to other algorithms / pools

The project is intentionally kept as a readable single-file example. To adapt it:

- **Different pool:** replace `fetchRates` and `fetchDailyRevenue` with calls to your pool's API.
- **Different algorithm:** set `proxy_algorithm` to whatever your miners use (`kawpow`, `verushash`, etc.) — Ultimate Proxy will filter workers accordingly.
- **Multiple algorithms:** run a separate instance with a separate config file for each algorithm.

---

## Notes

- The daemon switches workers only when a different coin becomes more profitable. If the best coin is already active, no API call to switch is made.
- The default profile is always updated so that miners connecting for the first time are sent to the current best coin.
- History is capped at 24 hours of snapshots. The ASCII chart displays the last 60 data points.
- Graceful shutdown on `SIGINT` / `SIGTERM`.
