# Example json file content

Create-validator

```json
{
    "pubkey": {"@type":"/cosmos.crypto.ed25519.PubKey","key":"c+EuycPpudgiyVl6guYG9oyPSImHHJz1z0Pg4ODKveo="},
    "amount": "99000000loya",
    "moniker": "calebmoniker",
    "identity": "optional identity signature (ex. UPort or Keybase)",
    "website": "validator's (optional) website",
    "security": "validator's (optional) security contact email",
    "details": "validator's (optional) details",
    "commission-rate": "0.1",
    "commission-max-rate": "0.2",
    "commission-max-change-rate": "0.01",
    "min-self-delegation": "1"
}
```

register-dataspec

```json
{
    "document_hash": "evm call data spec",
    "response_value_type": "bytes",
    "abi_components": [{
        "name": "chainId",
        "field_type": "uint256"
    }, {
        "name": "contractAddress",
        "field_type": "address"
    }, {
        "name": "calldata",
        "field_type": "bytes"
    }],
    "aggregation_method": "weighted-mode",
    "registrar": "tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l",
    "report_buffer_window": "300s"
}
```

gov-proposal

```json
