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