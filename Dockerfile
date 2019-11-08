FROM pmdcosta/golang:1.13 AS builder
WORKDIR /code

# Add code and compile it
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /app ./cmd/crawler

# Final image
FROM gcr.io/distroless/base
COPY --from=builder /app ./
ENTRYPOINT ["./app"]
