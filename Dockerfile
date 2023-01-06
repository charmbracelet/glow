FROM gcr.io/distroless/static
COPY glow /usr/local/bin/glow
ENTRYPOINT [ "/usr/local/bin/glow" ]
