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

FROM --platform=$BUILDPLATFORM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.21.10 AS builder
WORKDIR /go/src/volcano.sh/

# 声明 Buildx 自动提供的变量
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct && go mod download
ADD . volcano
RUN cd volcano && make vc-controller-manager

FROM swr.cn-south-1.myhuaweicloud.com/ascendhub/vc-controller-manager:v1.9.0-v7.3.0 
COPY --from=builder /go/src/volcano.sh/volcano/_output/bin/vc-controller-manager /vc-controller-manager
ENTRYPOINT ["/vc-controller-manager"]
