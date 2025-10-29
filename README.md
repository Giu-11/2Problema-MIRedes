# Problema 2 do MI de Concorrência e Conectividade.

Projeto desenvolvido para a disciplina **MI de Concorrência e Conectividade (TEC502)**.

## Pré-requisitos
- Go >= 1.25  
- Docker  
- Docker Compose (opcional, para rodar tudo em containers)

> ⚠️ Abra o terminal na **pasta raiz do projeto** antes de rodar qualquer comando.

## Executando o servidor com Docker Compose

### 1. Construir e rodar o container

```bash
docker-compose up --build
```

> Este comando irá **construir a imagem do servidor** e **iniciá-lo em um container**.<br>
> Para parar o servidor, pressione Ctrl + C no terminal.

#### Parar e remover o container

```bash
docker-compose down
```

> Isso irá **parar e remover o container** criado pelo Compose.

## 2. Rodando o cliente localmente no host

1. Abra um **novo terminal**.
2. Entre na pasta do cliente:

```bash
cd client
```

3. Execute o cliente com Go:

```bash
go run main.go
```

> O cliente irá se conectar ao servidor que está rodando no Docker.

---


