# Hoverfly captures

This directory holds hoverfly simulations.

To load these into hoverfly

```shell
hoverctl import <simulation.json> -v
```

## Simulations 

### `happy.json`

Call to AWS that returns a VPN with two healthy gateways

### `unhappy_one_down.json`

Call to AWS that returns a VPN with two gateways, one down and one up.

## Adding new simulation manually

You can decode an existing capture body and modify to create a new scenario.

To decode an existing capture

```shell
cat happy.json | jq -r '.data.pairs[1].response.body' | base64 -D | gzip -d > api.xml
```

Edit the resulting XML as you want, and then

```shell
cat api.xml | gzip | base64 > api_encoded.xml
```

Now copy the existing scenario and replace the response body with the contents of `api_encoded.xml`
