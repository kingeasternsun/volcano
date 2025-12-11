#!/bin/bash
# 构建并推送 Volcano 多平台镜像（不使用 buildx）
# 支持组件: scheduler, controller-manager, webhook-manager
# 使用方法: 
#   ./build-multi-platform-images.sh                    # 构建所有组件
#   ./build-multi-platform-images.sh scheduler          # 只构建 scheduler
#   ./build-multi-platform-images.sh scheduler controller-manager  # 构建指定组件

set -e

REGISTRY="registry.xxx.dev/volcano"
VERSION="v1.11.0-v7.1.RC1"

# 所有支持的组件
ALL_COMPONENTS=("scheduler" "controller-manager" "webhook-manager")

# 平台列表
PLATFORMS=("linux/amd64" "linux/arm64")
PLATFORM_TAGS=("amd64" "arm64")

# 解析命令行参数，确定要构建的组件
if [ $# -eq 0 ]; then
    # 没有参数，构建所有组件
    COMPONENTS=("${ALL_COMPONENTS[@]}")
else
    # 有参数，只构建指定的组件
    COMPONENTS=("$@")
    # 验证组件名称
    for comp in "${COMPONENTS[@]}"; do
        if [[ ! " ${ALL_COMPONENTS[@]} " =~ " ${comp} " ]]; then
            echo "错误: 不支持的组件 '${comp}'"
            echo "支持的组件: ${ALL_COMPONENTS[@]}"
            exit 1
        fi
    done
fi

echo "=========================================="
echo "Volcano 多平台镜像构建工具"
echo "=========================================="
echo "Registry: ${REGISTRY}"
echo "Version: ${VERSION}"
echo "平台: ${PLATFORMS[@]}"
echo "组件: ${COMPONENTS[@]}"
echo "注意: 此方法不使用 buildx，需要分别为每个平台构建"
echo "=========================================="

# 检查是否支持 manifest 命令
if ! docker manifest --help > /dev/null 2>&1; then
    echo "错误: docker manifest 命令不可用"
    echo "请确保 Docker 版本 >= 19.03 并且启用了实验性功能"
    echo "在 ~/.docker/config.json 中添加: { \"experimental\": \"enabled\" }"
    exit 1
fi

# 函数：构建单个组件的多平台镜像
build_component() {
    local COMPONENT=$1
    local IMAGE_NAME="vc-${COMPONENT}"
    local DOCKERFILE="./installer/dockerfile/${COMPONENT}/Dockerfile"
    local FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${VERSION}"
    
    echo ""
    echo "=========================================="
    echo "开始构建组件: ${COMPONENT}"
    echo "镜像: ${FULL_IMAGE}"
    echo "=========================================="
    
    # 检查 Dockerfile 是否存在
    if [ ! -f "${DOCKERFILE}" ]; then
        echo "错误: Dockerfile 不存在: ${DOCKERFILE}"
        return 1
    fi
    
    # 临时镜像标签数组
    local TEMP_IMAGES=()
    
    # 1. 为每个平台分别构建并推送镜像
    for i in "${!PLATFORMS[@]}"; do
        local PLATFORM="${PLATFORMS[$i]}"
        local PLATFORM_TAG="${PLATFORM_TAGS[$i]}"
        local TEMP_IMAGE="${FULL_IMAGE}-${PLATFORM_TAG}"
        TEMP_IMAGES+=("${TEMP_IMAGE}")
        
        echo ""
        echo "  [${COMPONENT}] 构建平台: ${PLATFORM}"
        echo "  临时镜像标签: ${TEMP_IMAGE}"
        
        # 构建镜像（使用 --platform 参数）
        docker build \
            --platform ${PLATFORM} \
            -t ${TEMP_IMAGE} \
            -f ${DOCKERFILE} \
            .
        
        # 推送镜像
        echo "  [${COMPONENT}] 推送 ${PLATFORM} 平台镜像..."
        docker push ${TEMP_IMAGE}
        
        echo "  ✓ [${COMPONENT}] ${PLATFORM} 平台构建完成"
    done
    
    # 2. 创建并推送多平台 manifest
    echo ""
    echo "  [${COMPONENT}] 创建多平台 manifest..."
    
    # 删除已存在的 manifest（如果存在）
    docker manifest rm ${FULL_IMAGE} 2>/dev/null || true
    
    # 创建新的 manifest
    local MANIFEST_CMD="docker manifest create ${FULL_IMAGE}"
    for TEMP_IMAGE in "${TEMP_IMAGES[@]}"; do
        MANIFEST_CMD="${MANIFEST_CMD} ${TEMP_IMAGE}"
    done
    
    eval ${MANIFEST_CMD}
    
    # 为每个平台设置架构信息
    for i in "${!PLATFORMS[@]}"; do
        local PLATFORM="${PLATFORMS[$i]}"
        local TEMP_IMAGE="${TEMP_IMAGES[$i]}"
        
        docker manifest annotate \
            --arch $(echo ${PLATFORM} | cut -d'/' -f2) \
            --os $(echo ${PLATFORM} | cut -d'/' -f1) \
            ${FULL_IMAGE} \
            ${TEMP_IMAGE}
    done
    
    # 推送 manifest
    echo "  [${COMPONENT}] 推送 manifest..."
    docker manifest push ${FULL_IMAGE}
    
    # 验证镜像清单
    echo "  [${COMPONENT}] 验证镜像清单..."
    docker manifest inspect ${FULL_IMAGE} | grep -E '"architecture"|"os"' | head -4 || true
    
    echo ""
    echo "  ✓ [${COMPONENT}] 构建完成！"
    echo "  最终镜像: ${FULL_IMAGE}"
    
    # 保存临时镜像标签，用于后续清理
    echo "${TEMP_IMAGES[@]}" > /tmp/volcano-temp-images-${COMPONENT}.txt
}

# 记录开始时间
START_TIME=$(date +%s)

# 构建所有指定的组件
SUCCESSFUL_COMPONENTS=()
FAILED_COMPONENTS=()

for COMPONENT in "${COMPONENTS[@]}"; do
    if build_component "${COMPONENT}"; then
        SUCCESSFUL_COMPONENTS+=("${COMPONENT}")
    else
        FAILED_COMPONENTS+=("${COMPONENT}")
        echo "  ✗ [${COMPONENT}] 构建失败"
    fi
done

# 计算总耗时
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
MINUTES=$((DURATION / 60))
SECONDS=$((DURATION % 60))

# 输出总结
echo ""
echo "=========================================="
echo "构建总结"
echo "=========================================="
echo "总耗时: ${MINUTES} 分 ${SECONDS} 秒"

if [ ${#SUCCESSFUL_COMPONENTS[@]} -gt 0 ]; then
    echo ""
    echo "成功构建的组件 (${#SUCCESSFUL_COMPONENTS[@]}):"
    for comp in "${SUCCESSFUL_COMPONENTS[@]}"; do
        echo "  ✓ ${comp}: ${REGISTRY}/vc-${comp}:${VERSION}"
    done
fi

if [ ${#FAILED_COMPONENTS[@]} -gt 0 ]; then
    echo ""
    echo "构建失败的组件 (${#FAILED_COMPONENTS[@]}):"
    for comp in "${FAILED_COMPONENTS[@]}"; do
        echo "  ✗ ${comp}"
    done
    exit 1
fi

# 清理临时镜像标签（可选）
echo ""
read -p "是否删除所有临时平台特定标签的镜像? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "清理临时镜像标签..."
    for COMPONENT in "${SUCCESSFUL_COMPONENTS[@]}"; do
        if [ -f "/tmp/volcano-temp-images-${COMPONENT}.txt" ]; then
            while read -r TEMP_IMAGE; do
                echo "删除 ${TEMP_IMAGE}..."
                docker rmi ${TEMP_IMAGE} 2>/dev/null || true
            done < /tmp/volcano-temp-images-${COMPONENT}.txt
            rm -f /tmp/volcano-temp-images-${COMPONENT}.txt
        fi
    done
    echo "清理完成"
else
    echo "保留临时镜像标签"
fi

echo ""
echo "=========================================="
echo "所有操作完成！"
echo "=========================================="
echo "构建的镜像:"
for comp in "${SUCCESSFUL_COMPONENTS[@]}"; do
    echo "  ${REGISTRY}/vc-${comp}:${VERSION}"
done
echo ""
echo "支持的平台: ${PLATFORMS[@]}"
echo "=========================================="

