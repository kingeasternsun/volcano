# Copyright 2019 The Volcano Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# 显式指定 builder 在构建机原生架构运行，提高速度
FROM --platform=$BUILDPLATFORM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.21.10 AS builder

# 声明 Buildx 自动提供的变量
ARG TARGETARCH

WORKDIR /go/src/volcano.sh/
COPY go.mod go.sum ./
RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct && go mod download
ADD . volcano

# 关键修改：将 TARGETARCH 传给 Makefile 的 GOARCH
RUN cd volcano && GOOS=linux GOARCH=${TARGETARCH} make vc-scheduler

FROM --platform=$TARGETPLATFORM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/alpine:latest 
# 注意：这里的路径要对应上
COPY --from=builder /go/src/volcano.sh/volcano/_output/bin/vc-scheduler /vc-scheduler
ENTRYPOINT ["/vc-scheduler"]