# Ultimate-Proxy.com profile switcher

An example profit-switching daemon for [ultimate-proxy.com](https://ultimate-proxy.com).

In its current state it monitors the profitability of several coins on the [Kryptex](https://pool.kryptex.com) mining pool and automatically switches all your workers to the most profitable coin by reassigning them to the corresponding profile on Ultimate Proxy. **By leveraging Ultimate Proxy features, your miners never lose a single second of mining time.**

## How it works

1. Every `interval` seconds the daemon calls the **Kryptex Pool API** to fetch live daily revenue and exchange rates for every configured coin.
2. It calls the **Ultimate Proxy API** to get your current aggregate hashrate (1 h average) so the revenue calculation reflects your real miners.
3. It sorts coins by daily revenue in your chosen fiat currency and picks the best one.
4. If the best coin changed, it bulk-assigns all your workers to the matching Ultimate Proxy profile and sets that profile as the default for new connections.
5. It prints a profitability table, historical averages, and a live ASCII chart in the terminal.
6. History is persisted to disk (JSON) so averages survive restarts.

```

  Profitability Report — 2026-02-23 18:34:45  ⚡ 8.07 KH/s
────────────────────────────────────────────────────────────────────────────────────
  Rank  Coin            Daily (coin)       Daily (EUR)      BTC/MH/Day   Price (USD)
────────────────────────────────────────────────────────────────────────────────────
  1     ★ XTM           227.10352962        0.30435865    0.0004931422      0.001136
  2       SAL             5.51041249        0.26074753    0.0004224805      0.040110
  3       XMR             0.00064817        0.23873052    0.0003868071    312.200000
  4       ZEPH            0.31730020        0.20397201    0.0003304891      0.544900
────────────────────────────────────────────────────────────────────────────────────
  ★ = currently mining

  Averages (10 samples)
────────────────────────────────────────────────────────
  Rank  Coin               Avg (EUR)    Avg BTC/MH/D
────────────────────────────────────────────────────────
  1     ★ XTM             0.26726135    0.0004383656
  2       SAL             0.26637118    0.0004378426
  3       XMR             0.23671362    0.0003893992
  4       ZEPH            0.20299451    0.0003336372
────────────────────────────────────────────────────────
  ⛏  MINED AVG        0.27729369    0.0004552985
────────────────────────────────────────────────────────


  Profitability Chart (EUR/day)
    0.310465 │     ┊●●●●●
             │    ●●───╮ 
             │ ●╭●╭╮  ╭╰─
             │───╰│╰──╯╰─
             │   ╰╭──────
             │────╯┊   
             │     ┊   
    0.155232 │     ┊   
             │     ┊   
             │     ┊   
             │     ┊   
             │     ┊   
             │     ┊   
             │     ┊   
    0.000000 │     ┊   
             └───────────
              17:48 18:34
   ● SAL ● XMR ● XTM ● ZEPH   ┊ = switch
```

## Requirements

- Go 1.24+ (to build from source)
- An [ultimate-proxy.com](https://ultimate-proxy.com) account with at least one profile per coin
- Miners already connected and routing through Ultimate Proxy

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

**Where to find profile IDs:** go to [ultimate-proxy.com/profiles](https://ultimate-proxy.com/profiles), click the action icon next to a profile, and choose `Copy ID`.

**Where to find your API key:** go to [https://ultimate-proxy.com/settings/api-keys](https://ultimate-proxy.com/settings/api-keys), the key need the following scopes: `workers:read, workers:write, profiles:write`

### 3. Run

```bash
./ultimate-proxy-profile-switcher -config config.yaml
```

## CLI flags


| Flag            | Default       | Description                                                 |
| --------------- | ------------- | ----------------------------------------------------------- |
| `-config`, `-c` | `config.yaml` | Path to the YAML config file                                |
| `-dry-run`      | `false`       | Print the profitability table without switching any workers |
| `-once`         | `false`       | Run a single cycle and exit immediately                     |

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

## Extending to other algorithms / pools

- **Different pool:** replace `fetchRates` and `fetchDailyRevenue` with calls to your pool's API.
- **Different algorithm:** set `proxy_algorithm` to whatever your miners use (`kawpow`, `scrypt`, etc.) — Ultimate Proxy will filter workers accordingly.
- **Multiple algorithms:** run a separate instance with a separate config file for each algorithm.

## Notes

- The default profile is always updated so that miners connecting for the first time are sent to the current best coin.
- History is capped at 24 hours of snapshots. The ASCII chart displays the last 60 data points.
