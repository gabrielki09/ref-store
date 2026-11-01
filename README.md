# RefStore — armazenamento deduplicado de arquivos por referência

## Estado final desta etapa

O RefStore é uma biblioteca/CLI em Go para armazenar e restaurar arquivos por
blocos, usando armazenamento endereçável por conteúdo (CAS).

## Visão geral e objetivos

Em vez de gravar cada arquivo integralmente, o RefStore o divide em blocos de
**64 KiB**. Cada bloco é identificado por seu SHA-256 e salvo uma única vez no
repositório CAS. Um manifesto JSON guarda a sequência ordenada desses hashes e
os metadados necessários para remontar o arquivo.

Objetivos:

- reduzir duplicação entre arquivos e dentro do próprio arquivo;
- processar store e restore em *streaming*, sem carregar o arquivo completo na
  memória;
- garantir a integridade do conteúdo por hashes SHA-256;
- publicar blocos e manifestos de forma segura com temporários e operações
  atômicas locais;
- expor estatísticas que distinguem reutilização física de repetição lógica.

## Arquitetura e fluxo

```text
arquivo de entrada
  └─ leitura sequencial em chunks de 64 KiB
       ├─ SHA-256 de cada chunk → bloco CAS (hash = endereço)
       ├─ sequência de hashes → manifesto JSON versionado
       └─ SHA-256 do arquivo inteiro → verificação fim a fim

manifesto + blocos CAS
  └─ restore sequencial dos hashes → arquivo restaurado
       └─ SHA-256 final comparado ao hash original
```

### Store

1. A CLI lê o arquivo em chunks de `ChunkSize` (64 KiB).
2. Para cada chunk, calcula SHA-256, atualiza o hash do arquivo original e o
   acrescenta à sequência do manifesto.
3. `StoreBlock` tenta publicar o conteúdo sob o nome do hash. O conteúdo é
   primeiro escrito em arquivo temporário; a publicação usa hard link para
   tornar a criação exclusiva. Se o destino já existir, o bloco é reutilizado.
4. Ao fim da leitura, é gravado um manifesto JSON, versionado e identificado
   por hash, com a ordem dos blocos, o tamanho e o hash do arquivo completo.

O processamento é incremental: o último chunk pode ser menor que 64 KiB, e
`io.ErrUnexpectedEOF` encerra normalmente essa última leitura parcial.

### Restore

O restore abre o manifesto, percorre a sequência na ordem original e grava cada
bloco no arquivo de saída em streaming. Durante a reconstrução, calcula o
SHA-256 do resultado; ao final, ele deve coincidir com o hash registrado no
manifesto. Um bloco ausente, ilegível ou um hash final divergente torna a
operação inválida.

### CAS, publicação e deduplicação

- **CAS SHA-256:** o nome/endereço de um bloco é o SHA-256 de seu conteúdo.
- **Publicação segura:** temporários evitam deixar um bloco parcialmente
  publicado; hard links permitem que concorrentes tentem publicar o mesmo hash
  sem sobrescrever o conteúdo já existente.
- **Deduplicação física:** se um hash já está no CAS, bytes não são regravados.
- **Deduplicação lógica:** uma repetição do hash dentro do mesmo arquivo segue
  no manifesto para preservar a posição do conteúdo, mas não exige outra cópia
  física do bloco.

## Manifesto

O manifesto é JSON e possui versão explícita, permitindo evolução compatível do
formato. Em termos conceituais, contém:

```json
{
  "version": 1,
  "original_file_hash": "<sha256 do arquivo completo>",
  "total_bytes": 637427,
  "blocks": ["<sha256 do chunk 1>", "<sha256 do chunk 2>"]
}
```

O identificador do manifesto também é derivado por hash. A lista de `blocks` é
ordenada: hashes iguais podem aparecer mais de uma vez, pois representam
ocorrências distintas do mesmo conteúdo na montagem do arquivo.

## Componentes principais

Os nomes abaixo correspondem às versões de código compartilhadas nesta etapa.

| Componente/função | Responsabilidade |
| --- | --- |
| CLI Go | Interpreta `-store`, `-restore`, `-delete` e `-debug` e encaminha a operação ao repositório. |
| `ProcessByte` | Lê o arquivo em chunks, calcula hashes, monta a sequência de blocos e consolida as estatísticas. |
| `CreateHasherSHA256` | Cria o hasher incremental do arquivo completo. |
| `ByteToHash` | Produz o hash SHA-256 de um bloco. |
| `StoreBlock` | Armazena ou reutiliza um bloco CAS e informa se houve criação física. |
| `ProcessResult` | Transporta sequência, hash final e totais de blocos/bytes novos, reutilizados e repetidos. |
| Manifesto JSON | Persiste a referência versionada ao arquivo e seus metadados de integridade. |

## Uso da CLI

Os modos concluídos são:

```text
-store    armazena um arquivo e gera sua referência/manifesto
-restore  remonta um arquivo a partir de uma referência/manifesto
-delete   remove a referência/manifesto indicada
-debug    exibe informações de diagnóstico/estatísticas
```

Exemplos representativos (substitua os caminhos e argumentos conforme a
interface da versão local da CLI):

```powershell
go run . -store "C:\dados\video.mp4"
go run . -restore "<hash-ou-referência-do-manifesto>" "C:\dados\video-restaurado.mp4"
go run . -delete "<hash-ou-referência-do-manifesto>"
go run . -debug
```

`-delete` atua sobre a referência/manifesto; a coleta de blocos CAS que tenham
ficado sem referência não faz parte do estado comprovado desta etapa.

## Estatísticas produzidas

`ProcessResult` separa as métricas para evitar interpretar reutilização como
nova ocupação de disco:

- `TotalBlock` e `TotalBytes`: ocorrências lidas e bytes lógicos do arquivo;
- `TotalUniqueBlocks`: hashes distintos encontrados no arquivo;
- `TotalNewBlocks` / `TotalNewBytes`: blocos/bytes efetivamente novos no CAS;
- `TotalReusedBlocks` / `TotalReusedBytes`: blocos já existentes no CAS e
  referenciados pela primeira vez naquele processamento;
- `TotalRepeatedBlocks` / `TotalRepeatedBytes`: ocorrências repetidas dentro do
  próprio arquivo;
- `OriginalFileHash`: SHA-256 do fluxo de entrada inteiro;
- `BlockSequence`: ordem necessária para a restauração.

## Resultado demonstrado com PDF

No teste apresentado para um PDF:

- tamanho de entrada: **637.427 bytes**;
- divisão: **10 chunks** de até 64 KiB;
- houve reutilização de blocos;
- quando a verificação de hash foi executada, o SHA-256 do arquivo restaurado
  foi idêntico ao SHA-256 do original.

Esses números registram o caso demonstrado; esta documentação não infere outros
testes, desempenho ou garantias de concorrência além do que foi compartilhado.
