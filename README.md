# lyf-kitty-prometheus-exporter

This is just a simple and stupid exporter for Lyf's kitties.

# Disclaimer

I'm not tied to Lyf, this project has no testing and simply exports the two metrics I needed:

- `lyf_contributions_counter`
- `lyf_total_collected_amount`

Owner firstname, lastname, ID and kitty ID are exported as metrics labels.

# Usage

Just replace the kittie's uuid with the right one. You can find it at the end of the URL: `https://app.lyf.eu/pot/fr/pot/**(uuid)**`

## With Docker

```
docker run -p 8080:8080 -e LYF_KITTY_UUID=uuid ghcr.io/cleming/lyf-kitty-prometheus-exporter
```