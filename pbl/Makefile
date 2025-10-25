# Inicia o NATS 1 em background e o Servidor 1 em foreground
run-pair1:
	@echo "Iniciando NATS 1 na porta 4223..."
	@docker run -d --rm --name nats1 -p 4223:4222 -p 8223:8222 nats:latest
	@echo "Iniciando Servidor 1 (conectando ao NATS 1)..."
	@ID=1 \
	PORT=8001 \
	PEERS="2=http://localhost:8002,3=http://localhost:8003" \
	NATS_URL="nats://localhost:4223" \

	go run ./server/

# Inicia o NATS 2 em background e o Servidor 2 em foreground
run-pair2:
	@echo "Iniciando NATS 2 na porta 4224..."
	@docker run -d --rm --name nats2 -p 4224:4222 -p 8224:8222 nats:latest
	@echo "Iniciando Servidor 2 (conectando ao NATS 2)..."
	@ID=2 \
	PORT=8002 \
	PEERS="1=http://localhost:8001,3=http://localhost:8003" \
	NATS_URL="nats://localhost:4224" \

	go run ./server/

# Inicia o NATS 3 em background e o Servidor 3 em foreground
run-pair3:
	@echo "Iniciando NATS 3 na porta 4225..."
	@docker run -d --rm --name nats3 -p 4225:4222 -p 8225:8222 nats:latest
	@echo "Iniciando Servidor 3 (conectando ao NATS 3)..."
	@ID=3 \
	PORT=8003 \
	PEERS="1=http://localhost:8001,2=http://localhost:8002" \
	NATS_URL="nats://localhost:4225" \

	go run ./server/


# Inicia o cliente (requer seu próprio terminal)
run-client:
	@echo "Iniciando cliente..."
	@go run ./client/


# Para e remove todos os contêineres NATS
stop-all-nats:
	@echo "Parando e removendo todos os conteineres NATS..."
	@docker rm -f nats1 nats2 nats3 || true
	@echo "Conteineres NATS removidos."

.PHONY: run-pair1 run-pair2 run-pair3 run-client stop-all-nats


# ==================== CONFIGURAÇÃO ====================
# EDITE AQUI OS IPs DAS MÁQUINAS PARA PRODUÇÃO
MACHINE1_IP ?= 192.168.1.10
MACHINE2_IP ?= 192.168.1.11
MACHINE3_IP ?= 192.168.1.12

# ==================== DESENVOLVIMENTO LOCAL (Localhost) ====================

# Inicia o NATS 1 em background e o Servidor 1 em foreground
run-pair1:
	@echo "Iniciando NATS 1 na porta 4223..."
	@docker run -d --rm --name nats1 -p 4223:4222 -p 8223:8222 nats:latest
	@echo "Iniciando Servidor 1 (conectando ao NATS 1)..."
	@ID=1 \
	PORT=8001 \
	PEERS="2=http://localhost:8002,3=http://localhost:8003" \
	NATS_URL="nats://localhost:4223" \
	RAFT_PEER_1="localhost:8001" \
	RAFT_PEER_2="localhost:8002" \
	RAFT_PEER_3="localhost:8003" \
	go run ./server/

# Inicia o NATS 2 em background e o Servidor 2 em foreground
run-pair2:
	@echo "Iniciando NATS 2 na porta 4224..."
	@docker run -d --rm --name nats2 -p 4224:4222 -p 8224:8222 nats:latest
	@echo "Iniciando Servidor 2 (conectando ao NATS 2)..."
	@ID=2 \
	PORT=8002 \
	PEERS="1=http://localhost:8001,3=http://localhost:8003" \
	NATS_URL="nats://localhost:4224" \
	RAFT_PEER_1="localhost:8001" \
	RAFT_PEER_2="localhost:8002" \
	RAFT_PEER_3="localhost:8003" \
	go run ./server/

# Inicia o NATS 3 em background e o Servidor 3 em foreground
run-pair3:
	@echo "Iniciando NATS 3 na porta 4225..."
	@docker run -d --rm --name nats3 -p 4225:4222 -p 8225:8222 nats:latest
	@echo "Iniciando Servidor 3 (conectando ao NATS 3)..."
	@ID=3 \
	PORT=8003 \
	PEERS="1=http://localhost:8001,2=http://localhost:8002" \
	NATS_URL="nats://localhost:4225" \
	RAFT_PEER_1="localhost:8001" \
	RAFT_PEER_2="localhost:8002" \
	RAFT_PEER_3="localhost:8003" \
	go run ./server/

# # ==================== PRODUÇÃO (Máquinas Diferentes) ====================

# # Servidor 1 (rodar na Máquina 1)
# server1:
# 	@echo " Iniciando Servidor 1..."
# 	@echo "   Cluster: $(MACHINE1_IP), $(MACHINE2_IP), $(MACHINE3_IP)"
# 	@ID=1 \
# 	PORT=8001 \
# 	PEERS="2=http://$(MACHINE2_IP):8002,3=http://$(MACHINE3_IP):8003" \
# 	NATS_URL="nats://$(MACHINE1_IP):4222" \
# 	RAFT_PEER_1="$(MACHINE1_IP):8001" \
# 	RAFT_PEER_2="$(MACHINE2_IP):8002" \
# 	RAFT_PEER_3="$(MACHINE3_IP):8003" \
# 	go run ./server/

# # Servidor 2 (rodar na Máquina 2)
# server2:
# 	@echo " Iniciando Servidor 2..."
# 	@echo "   Cluster: $(MACHINE1_IP), $(MACHINE2_IP), $(MACHINE3_IP)"
# 	@ID=2 \
# 	PORT=8002 \
# 	PEERS="1=http://$(MACHINE1_IP):8001,3=http://$(MACHINE3_IP):8003" \
# 	NATS_URL="nats://$(MACHINE1_IP):4222" \
# 	RAFT_PEER_1="$(MACHINE1_IP):8001" \
# 	RAFT_PEER_2="$(MACHINE2_IP):8002" \
# 	RAFT_PEER_3="$(MACHINE3_IP):8003" \
# 	go run ./server/

# # Servidor 3 (rodar na Máquina 3)
# server3:
# 	@echo " Iniciando Servidor 3..."
# 	@echo "   Cluster: $(MACHINE1_IP), $(MACHINE2_IP), $(MACHINE3_IP)"
# 	@ID=3 \
# 	PORT=8003 \
# 	PEERS="1=http://$(MACHINE1_IP):8001,2=http://$(MACHINE2_IP):8002" \
# 	NATS_URL="nats://$(MACHINE1_IP):4222" \
# 	RAFT_PEER_1="$(MACHINE1_IP):8001" \
# 	RAFT_PEER_2="$(MACHINE2_IP):8002" \
# 	RAFT_PEER_3="$(MACHINE3_IP):8003" \
# 	go run ./server/

# # NATS standalone (rodar na Máquina 1)
# nats:
# 	@echo " Iniciando NATS..."
# 	@docker run -d --rm --name nats-server \
# 		-p 4222:4222 \
# 		-p 8222:8222 \
# 		nats:latest

# # Para NATS standalone
# stop-nats:
# 	@echo "Parando NATS..."
# 	@docker rm -f nats-server || true

# # ==================== CLIENTE ====================

# # Cliente desenvolvimento
# run-client:
# 	@echo "Iniciando cliente..."
# 	@go run ./client/

# # Cliente produção (conecta ao cluster remoto)
# client:
# 	@echo " Iniciando cliente (conectando ao cluster)..."
# 	@SERVER1="$(MACHINE1_IP):8001" \
# 	SERVER2="$(MACHINE2_IP):8002" \
# 	SERVER3="$(MACHINE3_IP):8003" \
# 	go run ./client/

# # ==================== BUILD ====================

# # Compila binários para distribuir
# build:
# 	@echo " Compilando servidor e cliente..."
# 	@mkdir -p bin
# 	@go build -o bin/server ./server/
# 	@go build -o bin/client ./client/
# 	@echo " Binários compilados em ./bin/"

# # ==================== LIMPEZA ====================

# # Para todos os NATS locais
# stop-all-nats:
# 	@echo "Parando todos os NATS..."
# 	@docker rm -f nats1 nats2 nats3 nats-server || true

# # Limpa binários
# clean:
# 	@echo " Limpando..."
# 	@rm -rf bin/

# # ==================== HELP ====================

# help:
# 	@echo "════════════════════════════════════════════════════"
# 	@echo " PBL Server - Comandos Disponíveis"
# 	@echo "════════════════════════════════════════════════════"
# 	@echo ""
# 	@echo " DESENVOLVIMENTO LOCAL (1 máquina, 3 terminais):"
# 	@echo "  make run-pair1     - NATS1 + Servidor 1"
# 	@echo "  make run-pair2     - NATS2 + Servidor 2"
# 	@echo "  make run-pair3     - NATS3 + Servidor 3"
# 	@echo "  make run-client    - Cliente local"
# 	@echo ""
# 	@echo " PRODUÇÃO (3 máquinas separadas):"
# 	@echo "  make nats          - NATS standalone (Máquina 1)"
# 	@echo "  make server1       - Servidor 1 (Máquina 1)"
# 	@echo "  make server2       - Servidor 2 (Máquina 2)"
# 	@echo "  make server3       - Servidor 3 (Máquina 3)"
# 	@echo "  make client        - Cliente conectando ao cluster"
# 	@echo ""
# 	@echo "  Configure IPs no topo do Makefile ou:"
# 	@echo "     make server1 MACHINE1_IP=10.0.0.5 ..."
# 	@echo ""
# 	@echo " BUILD:"
# 	@echo "  make build         - Compila servidor e cliente"
# 	@echo ""
# 	@echo " LIMPEZA:"
# 	@echo "  make stop-all-nats - Para todos os NATS"
# 	@echo "  make stop-nats     - Para NATS standalone"
# 	@echo "  make clean         - Remove binários"
# 	@echo ""
# 	@echo "════════════════════════════════════════════════════"

# .PHONY: run-pair1 run-pair2 run-pair3 run-client \
#         server1 server2 server3 nats stop-nats client \
#         build stop-all-nats clean help

