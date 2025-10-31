# ==================== CONFIGURAÇÃO ====================
# EDITE AQUI OS IPs DAS MÁQUINAS PARA PRODUÇÃO
MACHINE1_IP ?= 172.16.201.16
MACHINE2_IP ?= 172.16.201.15
MACHINE3_IP ?= 172.16.201.1

# Portas NATS para cada máquina
NATS_PORT_1 := 4223
NATS_PORT_2 := 4224
NATS_PORT_3 := 4225
# ========================================================

.PHONY: run-pair1 run-pair2 run-pair3 run-client stop-all-nats server1 server2 server3 stop-prod-nats client build clean help

# ==================== DESENVOLVIMENTO LOCAL (Localhost) ====================
# (Esta seção permanece como a sua, está perfeita)

run-pair1:
	@echo "Iniciando NATS 1 (Local) na porta 4223..."
	@docker run -d --rm --name nats1 -p 4223:4222 -p 8223:8222 nats:latest
	@echo "Iniciando Servidor 1 (Local)..."
	@ID=1 \
	PORT=8001 \
	PEERS="2=http://localhost:8002,3=http://localhost:8003" \
	NATS_URL="nats://localhost:4223" \
	RAFT_ADVERTISE_ADDR="localhost:8001" \
	go run ./server/

run-pair2:
	@echo "Iniciando NATS 2 (Local) na porta 4224..."
	@docker run -d --rm --name nats2 -p 4224:4222 -p 8224:8222 nats:latest
	@echo "Iniciando Servidor 2 (Local)..."
	@ID=2 \
	PORT=8002 \
	PEERS="1=http://localhost:8001,3=http://localhost:8003" \
	NATS_URL="nats://localhost:4224" \
	RAFT_ADVERTISE_ADDR="localhost:8002" \
	go run ./server/

run-pair3:
	@echo "Iniciando NATS 3 (Local) na porta 4225..."
	@docker run -d --rm --name nats3 -p 4225:4222 -p 8225:8222 nats:latest
	@echo "Iniciando Servidor 3 (Local)..."
	@ID=3 \
	PORT=8003 \
	PEERS="1=http://localhost:8001,2=http://localhost:8002" \
	NATS_URL="nats://localhost:4225" \
	RAFT_ADVERTISE_ADDR="localhost:8003" \
	go run ./server/

# ==================== PRODUÇÃO (Máquinas Diferentes) ====================
#
# COMO USAR:
# 1. Edite os IPs no topo do arquivo.
# 2. Copie o projeto para as 3 máquinas.
# 3. Em CADA máquina, rode: make build
# 4. Na Máquina 1, rode: make server1
# 5. Na Máquina 2, rode: make server2
# 6. Na Máquina 3, rode: make server3
#

# Servidor 1 (rodar na Máquina 1)
server1: build
	@echo "==> Iniciando NATS 1 (Produção) na porta $(NATS_PORT_1)..."
	@docker run -d --rm --name prod-nats1 \
		-p $(NATS_PORT_1):4222 \
		-p 8223:8222 \
		nats:latest
	@echo "==> Iniciando Servidor 1 (Produção)..."
	@ID=1 \
	PORT=8001 \
	PEERS="2=http://$(MACHINE2_IP):8002,3=http://$(MACHINE3_IP):8003" \
	NATS_URL="nats://localhost:$(NATS_PORT_1)" \
	RAFT_ADVERTISE_ADDR="$(MACHINE1_IP):8001" \
	./bin/server

# Servidor 2 (rodar na Máquina 2)
server2: build
	@echo "==> Iniciando NATS 2 (Produção) na porta $(NATS_PORT_2)..."
	@docker run -d --rm --name prod-nats2 \
		-p $(NATS_PORT_2):4222 \
		-p 8224:8222 \
		nats:latest
	@echo "==> Iniciando Servidor 2 (Produção)..."
	@ID=2 \
	PORT=8002 \
	PEERS="1=http://$(MACHINE1_IP):8001,3=http://$(MACHINE3_IP):8003" \
	NATS_URL="nats://localhost:$(NATS_PORT_2)" \
	RAFT_ADVERTISE_ADDR="$(MACHINE2_IP):8002" \
	./bin/server

# Servidor 3 (rodar na Máquina 3)
server3: build
	@echo "==> Iniciando NATS 3 (Produção) na porta $(NATS_PORT_3)..."
	@docker run -d --rm --name prod-nats3 \
		-p $(NATS_PORT_3):4222 \
		-p 8225:8222 \
		nats:latest
	@echo "==> Iniciando Servidor 3 (Produção)..."
	@ID=3 \
	PORT=8003 \
	PEERS="1=http://$(MACHINE1_IP):8001,2=http://$(MACHINE2_IP):8002" \
	NATS_URL="nats://localhost:$(NATS_PORT_3)" \
	RAFT_ADVERTISE_ADDR="$(MACHINE3_IP):8003" \
	./bin/server

# Para os NATS de produção
stop-prod-nats:
	@echo "Parando NATS de produção..."
	@docker rm -f prod-nats1 prod-nats2 prod-nats3 || true

# ==================== CLIENTE ====================

run-client:
	@echo "Iniciando cliente (dev)..."
	@go run ./client/

client:
	@echo "Executando cliente de produção (./bin/client)..."
	@echo "Lembre-se de editar 'client/main.go' com os IPs e compilar (make build)."
	@./bin/client

# ==================== BUILD E LIMPEZA ====================

build:
	@echo "==> Compilando servidor e cliente..."
	@mkdir -p bin
	@go build -o bin/server ./server/
	@go build -o bin/client ./client/
	@echo "Binários compilados em ./bin/"

stop-all-nats:
	@echo "Parando todos os NATS (dev e prod)..."
	@docker rm -f nats1 nats2 nats3 prod-nats1 prod-nats2 prod-nats3 || true

clean:
	@echo "Limpando binários..."
	@rm -rf bin/

# ==================== HELP ====================
help:
	@echo "==========================================================="
	@echo " Comandos Disponíveis"
	@echo "==========================================================="
	@echo ""
	@echo " DESENVOLVIMENTO LOCAL (1 máquina, 3 terminais):"
	@echo "  make run-pair1     - NATS1 + Servidor 1"
	@echo "  make run-pair2     - NATS2 + Servidor 2"
	@echo "  make run-pair3     - NATS3 + Servidor 3"
	@echo "  make run-client    - Cliente local"
	@echo ""
	@echo " PRODUÇÃO (3 máquinas separadas):"
	@echo "  make build         - Compile primeiro em CADA máquina"
	@echo "  make server1       - NATS1 + Servidor 1 (Rodar na Máquina 1)"
	@echo "  make server2       - NATS2 + Servidor 2 (Rodar na Máquina 2)"
	@echo "  make server3       - NATS3 + Servidor 3 (Rodar na Máquina 3)"
	@echo "  make client        - Cliente de produção (bin/client)"
	@echo ""
	@echo "  Configure IPs no topo do Makefile!"
	@echo ""
	@echo " LIMPEZA:"
	@echo "  make stop-all-nats - Para todos os NATS (dev e prod)"
	@echo "  make stop-prod-nats- Para NATS de produção"
	@echo "  make clean         - Remove binários"
	@echo ""