# Inventa (latin for Discovery)

Inventa is a visualiser/path computer for network topology data. Currently it's fed by BGP-LS using the GoBGP libraries but there are plans for more inputs in future. It's currently in pretty early stages but is (mostly) functional - and needs some better webui skills to pretty it up some.

It has the ability to show the entire topology, and also calculate/display visually shortest/best path(s) between source and destination nodes.

It uses the following projects:
https://github.com/osrg/gobgp - For the BGP-LS libraries
https://github.com/cytoscape/cytoscape.js/ - For the WebUI Visualisation
https://github.com/RyanCarrier/dijkstra - For the Dijkstra SPF calculations

## To use

Copy config.yaml.example to config.yaml somewhere and edit

```
cd src/inventa/
go run . -c /path/to/config.yaml
```

## Docker
You can build/run a docker container for this, put config.yaml in an empty
directory and mount it into the container.

```
docker build -t inventa .
docker run --rm -v/path/to/local/conf/directory:/etc/inventa:ro --expose 8123 inventa
```

More/better instructions to come!

