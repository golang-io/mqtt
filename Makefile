# MQTT客户端测试程序 Makefile

.PHONY: build clean run test help

# 默认目标
all: build

# 构建程序
build:
	@echo "构建MQTT客户端测试程序..."
	go build -o mqtt-test-client .
	@echo "构建完成: mqtt-test-client"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	rm -f mqtt-test-client
	@echo "清理完成"

# 运行测试程序
run: build
	@echo "运行MQTT客户端测试程序..."
	./mqtt-test-client

# 运行测试（需要MQTT服务端运行）
test: build
	@echo "运行MQTT客户端测试程序..."
	@echo "注意: 确保MQTT服务端正在运行"
	./mqtt-test-client

# 安装依赖
deps:
	@echo "安装依赖..."
	go mod tidy
	go mod vendor

# 检查代码
check:
	@echo "检查代码..."
	go vet .
	gofmt -d .

# 显示帮助信息
help:
	@echo "MQTT客户端测试程序 Makefile"
	@echo ""
	@echo "可用目标:"
	@echo "  make build    # 构建程序"
:q	@echo "  make run      # 构建并运行"
	@echo "  make clean    # 清理文件"
