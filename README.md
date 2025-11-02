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

## Executando com Makefile (Local ou Distribuído)

O `Makefile` oferece dois modos de execução: um para simular os 3 servidores localmente (Desenvolvimento) e outro para rodar em 3 máquinas físicas diferentes (Produção).

### Cenário 1: Rodando Localmente (Sua Máquina)

Este modo simula os 3 servidores na sua própria máquina (`localhost`), onde cada um roda seu próprio NATS. Você precisará de **4 terminais**.

1.  **Terminal 1:** Inicie o primeiro par (NATS 1 + Servidor 1):
    ```bash
    make run-pair1
    ```
   
2.  **Terminal 2:** Inicie o segundo par (NATS 2 + Servidor 2):
    ```bash
    make run-pair2
    ```
   
3.  **Terminal 3:** Inicie o terceiro par (NATS 3 + Servidor 3):
    ```bash
    make run-pair3
    ```
   
4.  **Terminal 4:** Inicie o cliente:
    ```bash
    make run-client
    ```
   

> **Para parar tudo:** Feche os terminais dos servidores e rode `make stop-all-nats` para parar os contêineres NATS que ficaram rodando.

### Cenário 2: Rodando em Múltiplas Máquinas 

Este modo é para implantar o sistema em 3 máquinas diferentes em uma rede.

#### Passo 1: Configuração (Obrigatório)

Antes de tudo, você **deve** dizer ao código quais são os IPs das suas máquinas.

1.  **Editar o `Makefile`:**
    Abra o `Makefile` e mude os IPs no topo do arquivo para os IPs reais das suas 3 máquinas.
    ```makefile
    # ==================== CONFIGURAÇÃO ====================
    # EDITE AQUI OS IPs DAS MÁQUINAS PARA PRODUÇÃO
    MACHINE1_IP ?= 172.16.201.16
    MACHINE2_IP ?= 172.16.201.15
    MACHINE3_IP ?= 172.16.201.1
    # ...
    ```

2.  **Editar o `client/main.go`:**
    Abra o arquivo `client/main.go` e atualize a lista `servers` para usar os mesmos IPs públicos e as portas NATS correspondentes (definidas no `Makefile`).
    ```go
    // ... em client/main.go
    servers := []models.ServerInfo{
        {ID: 1, Name: "Servidor 1", NATS: "nats://172.16.201.16:4223"},
        {ID: 2, Name: "Servidor 2", NATS: "nats://172.16.201.15:4224"},
        {ID: 3, Name: "Servidor 3", NATS: "nats://172.16.201.1:4225"},
    }
    // ...
    ```

#### Passo 2: Compilação

1.  Copie a pasta inteira do projeto para as 3 máquinas.
2.  Em **cada uma** das 3 máquinas, abra um terminal e compile os binários:
    ```bash
    make build
    ```
   

#### Passo 3: Execução

1.  **Na Máquina 1:** Abra um terminal e rode:
    ```bash
    make server1
    ```
    > (Isso irá iniciar o `prod-nats1` via Docker e o `./bin/server` ID 1)

2.  **Na Máquina 2:** Abra um terminal e rode:
    ```bash
    make server2
    ```
    > (Isso irá iniciar o `prod-nats2` via Docker e o `./bin/server` ID 2)

3.  **Na Máquina 3:** Abra um terminal e rode:
    ```bash
    make server3
    ```
    > (Isso irá iniciar o `prod-nats3` via Docker e o `./bin/server` ID 3)

#### Passo 4: Jogar

Em qualquer máquina (sua, ou uma das 3) que tenha o projeto compilado, rode:
```bash
make client
````

> O cliente (`./bin/client`) irá ler os IPs que você configurou no `client/main.go` e se conectar à rede de servidores.

#### Passo 5: Parar (Produção)

Para parar os servidores, use `Ctrl+C` nos terminais. Para parar os contêineres NATS de produção, rode em cada máquina:

```bash
make stop-prod-nats
```

## Testando o Servidor (Testes de Integração)

O projeto inclui testes de integração (`server/server_integration_test.go`) que simulam múltiplos clientes "falsos" se conectando ao servidor para testar logins, abertura de pacotes e matchmaking (normal e de stress).

**Importante:** Estes testes **exigem** que os servidores NATS e os servidores Raft estejam rodando. A forma mais fácil de fazer isso é usando o cenário de desenvolvimento local.

### Como Rodar os Testes:

1.  **Inicie os Servidores:** Abra 3 terminais e inicie os 3 servidores usando o `Makefile` (assim como no "Cenário 1: Rodando Localmente"):

      * **Terminal 1:** `make run-pair1`
      * **Terminal 2:** `make run-pair2`
      * **Terminal 3:** `make run-pair3`

2.  **Aguarde a Eleição:** Espere alguns segundos para os servidores se conectarem e elegerem um líder.

3.  **Rode os Testes:** Abra um **quarto terminal** na raiz do projeto e execute:

    ```bash
    # Navega até a pasta do servidor e roda os testes com verbose
    cd server
    go test -v
    ```

    *ou, da raiz do projeto:*

    ```bash
    go test ./server/ -v
    ```

    > O `-v` (verbose) é importante para ver os logs em tempo real, como "Cliente A entrou na fila..." e os resultados dos testes de stress.

4.  **(Opcional) Rodar um Teste Específico:** Para rodar apenas um teste (ex: o de stress de matchmaking), use a flag `-run`:

    ```bash
    # Estando na pasta 'server/'
    go test -v -run TestIntegration_StressMatchmaking
    ```

-----

## Outros Comandos `Makefile`

```bash
# Compila os binários do cliente e servidor para ./bin/
make build
```

```bash
# Remove a pasta ./bin/
make clean
```

```bash
# Para TODOS os contêineres NATS (dev e prod)
make stop-all-nats
```

```bash
# Mostra a lista de todos os comandos
make help
```

```
```