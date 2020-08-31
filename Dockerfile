FROM gcr.io/distroless/static:nonroot

COPY _output/manager /manager

CMD ["/manager"]
