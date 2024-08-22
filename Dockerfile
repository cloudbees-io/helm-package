FROM gcr.io/distroless/static:nonroot
COPY bin/cbhelmpkg /
USER root:root
ENTRYPOINT ["/cbhelmpkg"]
