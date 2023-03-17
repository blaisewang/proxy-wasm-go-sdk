## dispatch_request_body

this example authorize requests depending on the result from the detector cluster

### Run

```shell
docker run -d -p 18000:18000 -p 8001:8001 -v $(pwd)/examples/dispatch_request_body/envoy.yaml:/etc/envoy/envoy.yaml -v $(pwd)/examples:/examples envoyproxy/envoy:v1.25-latest -c /etc/envoy/envoy.yaml -l debug
```

### Test

I used the following Python script to generate the `example.txt` file. The generated text file is ~160KB.

```python
s = ""
for i in range (0, 10):
    s += str(i) * 16384 + "\n\n"

with open("example.txt", "w") as f:
    f.write(s)
```

Request with the `example.txt` file.

```shell
curl -X POST -d "@example.txt" http://127.0.0.1:18000
```
