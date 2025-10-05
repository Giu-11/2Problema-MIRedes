# Imagem base do Go
FROM golang:1.25

# Criar diretório de trabalho dentro do container
WORKDIR /app

# Copiar os arquivos do projeto
COPY . .

# Baixar dependências e compilar
RUN go mod tidy
RUN go build -o server main.go

# Definir variáveis padrão (podem ser sobrescritas no docker-compose)
ENV PORT=8001
ENV PEERS=""

# Expor a porta
EXPOSE 8001

# Rodar o servidor
CMD ["./server"]
