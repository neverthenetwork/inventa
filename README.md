# Inventa (latin for Discovery)

Inventa is a visualiser/path computer for network topology data. Currently it's fed by BGP-LS using the GoBGP libraries but there are plans for more inputs in future.

It has the ability to show the entire topology, and also calculate/display visually shortest/best path(s) between source and destination nodes.

## To use
Copy src/inventa/config.yaml.example to config.yaml somewhere and edit

```
cd src/inventa/
go run . -c /path/to/config.yaml
```

More/better instructions to come!

