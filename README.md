# Goto

A small hyperlink bounce server.

## Usage

Define some configuration:

```json
{
  "foo": "https://example.com/foo",
  "bar": "https://example.com/bar?q=1"
}
```

Launch the server:

```
% go build .
% ./goto -listen='[::]:8080'
```

Fetch a link:

```
% curl -v http://localhost:8080/bar?n=3
< HTTP/1.1 303 See Other
< Location: https://example.com/bar?q=1&n=3
```
