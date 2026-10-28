# This dockerfile should be built with `make docker-build` from the stash root.

# Build Frontend
FROM node:24-alpine AS frontend
RUN apk add --no-cache make git
## cache node_modules separately
COPY ./ui/v2.5/package.json ./ui/v2.5/pnpm-lock.yaml /stash/ui/v2.5/
WORKDIR /stash
COPY Makefile /stash/
COPY ./graphql /stash/graphql/
COPY ./ui /stash/ui/
# pnpm install with npm
RUN npm install -g pnpm
RUN make pre-ui
RUN make generate-ui
ARG GITHASH
ARG STASH_VERSION
RUN BUILD_DATE=$(date +"%Y-%m-%d %H:%M:%S") make ui-only

# Build Backend
FROM golang:1.25.9-alpine AS backend
RUN apk add --no-cache make alpine-sdk
WORKDIR /stash
COPY ./go* ./*.go Makefile gqlgen.yml .gqlgenc.yml /stash/
COPY ./graphql /stash/graphql/
COPY ./scripts /stash/scripts/
COPY ./pkg /stash/pkg/
COPY ./cmd /stash/cmd/
COPY ./internal /stash/internal/
# needed for generate-login-locale
COPY ./ui /stash/ui/
RUN make generate-backend generate-login-locale
COPY --from=frontend /stash /stash/
ARG GITHASH
ARG STASH_VERSION
RUN make flags-release flags-pie stash

# Final Runnable Image
FROM alpine:latest
# vips-tools and ffmpeg are required.
# intel-media-driver (Broadwell and newer, incl. Arc), libva-intel-driver (older
# iGPUs) and libva-utils (`vainfo`) give ffmpeg VAAPI access to Intel GPUs when
# /dev/dri is passed into the container. They are inert without a GPU.
RUN apk add --no-cache ca-certificates vips-tools ffmpeg \
    intel-media-driver libva-intel-driver libva-utils
# Runtime for Intel Quick Sync (QSV). The package names differ between Alpine
# releases, so a miss here must not fail the build: VAAPI works without it.
RUN apk add --no-cache libvpl vpl-gpu-rt \
    || apk add --no-cache libvpl onevpl-intel-gpu \
    || echo "QSV runtime not available on this Alpine release; VAAPI will be used"
COPY --from=backend /stash/stash /usr/bin/
ENV STASH_CONFIG_FILE=/root/.stash/config.yml
EXPOSE 2704
ENTRYPOINT ["stash"]
