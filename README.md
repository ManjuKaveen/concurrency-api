# concurrency-api

This api supports concurrent request processing with configurable limits and track the performance metrics.

---

## features

- POST `\process` to simulate task process
- GET `\metrics` to retrieve the following:
        - total number of request handled
        - average request processing time in ms
        - active request
- configure max concurrency via .env file
- Graceful shutdown

---

## Test

Clone the project: 

 ```bash
   git clone https://github.com/manjukaveen/concurrency-api.git
   cd concurrency-api
   go run main.go


