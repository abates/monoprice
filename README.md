# monoprice
REST API for Monoprice multi-zone amplifier system

## Usage

Start the server:
`sh
go run ./cmd/ampserver/ -noauth server &
`

Query discovered zones:
`sh
curl localhost:8000/11/zones
`


Query for status:
`sh
curl localhost:8000/11/status
`

Send command:
`sh
curl -X PUT localhost:8000/11/power/false
`
