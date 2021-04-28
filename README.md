# monoprice
REST API for Monoprice multi-zone amplifier system

## Usage

Start the server:
```sh
go run ./cmd/ampserver/ -noauth server 
2021/04/28 15:46:48 Initializing amplifier zones
2021/04/28 15:46:48 Found Zone 11
2021/04/28 15:46:48 Found Zone 12
2021/04/28 15:46:48 Found Zone 13
2021/04/28 15:46:49 Found Zone 14
2021/04/28 15:46:49 Found Zone 15
2021/04/28 15:46:49 Found Zone 16
2021/04/28 15:46:50 Zone 21 is not attached
2021/04/28 15:46:51 Zone 22 is not attached
2021/04/28 15:46:52 Zone 23 is not attached
2021/04/28 15:46:53 Zone 24 is not attached
2021/04/28 15:46:54 Zone 25 is not attached
2021/04/28 15:46:55 Zone 26 is not attached
2021/04/28 15:46:56 Zone 31 is not attached
2021/04/28 15:46:57 Zone 32 is not attached
2021/04/28 15:46:58 Zone 33 is not attached
2021/04/28 15:46:59 Zone 34 is not attached
2021/04/28 15:47:00 Zone 35 is not attached
2021/04/28 15:47:01 Zone 36 is not attached
2021/04/28 15:47:01 Connected to amplifier, found zones 11,12,13,14,15,16
2021/04/28 15:47:01 API Server started, listening on port 8000
```

Query discovered zones:
```sh
curl localhost:8000/zones
[11,12,13,14,15,16]
```


Query for status:
```sh
curl localhost:8000/11/status
{"pa":false,"power":false,"mute":false,"do_not_disturb":false,"volume":13,"treble":7,"bass":5,"balance":10,"source":3,"keypad":true}
```

Send command:
`sh
curl -X PUT localhost:8000/11/power/false
{}
`
