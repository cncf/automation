jq '[to_entries | map(.value) | map(select(.Timestamp != "")) | group_by(.Country)[] | .[0] | {Timestamp: .Timestamp, Country: .Country}]' ../Kubestronaut.json > countries_timeline.json
