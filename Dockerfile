FROM gcr.io/distroless/static:nonroot

COPY bin/cbhelmpkg /

ENTRYPOINT ["/cbhelmpkg"]
