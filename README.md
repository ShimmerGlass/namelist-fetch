# namelist-fetch

Fetch, refresh and combine namelist (pihole lists, adblocking etc) into a single file.

Use pihole lists with other dns proxies such as [dnscrypt-proxy](https://github.com/DNSCrypt/dnscrypt-proxy).

## Config

| Key               | Default | Description                                                                    |
| ----------------- | ------- | ------------------------------------------------------------------------------ |
| `NLF_TARGET_FILE` | unset   | **Required**, Where to write the file. example: `blocked-names.txt`            |
| `NLF_TEMP_DIR`    | `/tmp`  | Where to write temporary files                                                 |
| `NLF_LISTEN_ADDR` | unset   | HTTP server listen address. Set to enable prometheus metrics. example: `:8080` |
| `NLF_INTERVAL`    | `4h`    | How often to refresh lists                                                     |
| `NLF_LIST_[NAME]` |         | Name and URL of list to fetch, can be set multiple times for multiple lists    |

## Docker compose

```
services:
  namelist-fetch:
   image: shimmerglass/namelist-fetch:latest
   container_name: namelist-fetch
   restart: unless-stopped
   volumes:
     - ./dnscrypt:/config

   environment:
     NLF_LISTEN_ADDR: :8080
     NLF_INTERVAL: 3h

     NLF_TARGET_FILE: /config/blocked-names.txt
     NLF_TEMP_DIR: /config/blocklists-parts

     NLF_LIST_ADS: https://media.githubusercontent.com/media/zachlagden/Pi-hole-Optimized-Blocklists/refs/heads/main/lists/advertising.txt
     NLF_LIST_TRACK: https://media.githubusercontent.com/media/zachlagden/Pi-hole-Optimized-Blocklists/refs/heads/main/lists/tracking.txt
     NLF_LIST_MALICIOUS: https://media.githubusercontent.com/media/zachlagden/Pi-hole-Optimized-Blocklists/refs/heads/main/lists/malicious.txt
     NLF_LIST_SUSPCIOUS: https://media.githubusercontent.com/media/zachlagden/Pi-hole-Optimized-Blocklists/refs/heads/main/lists/suspicious.txt
```

## Metrics

(if enabled with `NLF_LISTEN_ADDR`)

- `namelistfetch_list_status`: list last refresh status
- `namelistfetch_list_reload_time_seconds`: time taken to refresh the list
- `namelistfetch_list_last_fetch_unix`: last unix timestamp (seconds) when the list was last refreshed
