# Terraform Provider for Procurator

Полноценный Terraform provider для платформы виртуализации **Procurator**.

Позволяет управлять:

* виртуальными машинами
* шаблонами
* хранилищами (datastore)
* сетями (Umbra)
* коммутаторами
* дисками и NIC
* snapshot’ами

Работает напрямую через gRPC API сервисов:

* **core** — управление VM и datastore
* **umbra** — сеть (switch / network / NIC)
* **storage** — backend storage (используется через core)

---

# Архитектура

Procurator состоит из нескольких сервисов:

| Сервис  | Порт  | Назначение            |
| ------- | ----- | --------------------- |
| core    | 3641  | VM, datastore, задачи |
| umbra   | 50051 | сети и коммутаторы    |
| storage | 3642  | backend storage       |

Terraform provider:

* подключается ко всем сервисам
* использует единый TLS + token
* управляет сущностями через правильный сервис (core / umbra)

---

# Установка

## Сборка

```bash
GOOS=darwin GOARCH=arm64 go build -mod=mod -o terraform-provider-procurator
```

## Установка плагина

```bash
mkdir -p ~/.terraform.d/plugins/local/procurator/procurator/0.1.0/darwin_arm64

cp terraform-provider-procurator \
  ~/.terraform.d/plugins/local/procurator/procurator/0.1.0/darwin_arm64/

chmod +x ~/.terraform.d/plugins/local/procurator/procurator/0.1.0/darwin_arm64/terraform-provider-procurator
```

---

# Provider configuration

```hcl
provider "procurator" {
  endpoint         = "10.10.102.22:3641"
  umbra_endpoint   = "10.10.102.22:50051"
  storage_endpoint = "10.10.102.22:3642"

  ca_file   = "/path/to/ca.pem"
  authority = "127.0.0.1"
}
```

## Параметры

| Поле             | Описание                      | Опционально
| ---------------- | ----------------------------- | -----------------------------
| endpoint         | core gRPC endpoint            | Поле обязательное. Порт указывать не обязательно. Если порт измениться то указываем полный endpoint |
| umbra_endpoint   | endpoint для сетевого сервиса | необязательное поле если порт не изменяли
| storage_endpoint | storage backend               | необязательное поле если порт не изменяли
| ca_file          | CA сертификат                 | необязательное поле если не изменяли сертификат |
| authority        | TLS authority                 | необязательное поле если не изменяли |

---

# Основные ресурсы

## Datastore (LVM через core)

```hcl
resource "procurator_datastore_lvm" "data1" {
  name = "data1"

  devices = [
    "sdb"
  ]
}
```

### Важно

* создание идёт через **core**
* storage вызывается внутри core
* `vg_name` генерируется автоматически
* пользователь задаёт только:

  * имя datastore
  * список устройств

---

## Datastore (универсальный)

```hcl
resource "procurator_datastore" "data1" {
  name      = "data1"
  type_code = 2

  devices = ["sdb"]
}
```

### type_code

| Код | Тип          |
| --- | ------------ |
| 1   | Local        |
| 2   | LVM (sStorm) |
| 3   | dStorm       |
| 4   | NFS          |

---

## Datastore folder

```hcl
resource "procurator_datastore_folder" "images" {
  path = "DATASTORE_ID:/images"
}
```

---

## Switch (Umbra)

```hcl
resource "procurator_switch" "main" {
  mtu = 1500

  nics = {
    active  = ["enp1s0"]
    standby = []
    unused  = []
    inherit = false
  }
}
```

---

## Network

```hcl
resource "procurator_network" "net1" {
  name      = "prod-net"
  vlan      = 120
  switch_id = "SWITCH_ID"
}
```

---

## VM

```hcl
resource "procurator_vm" "vm1" {
  name = "vm1"
  cpu  = 4
  ram  = 8192
}
```

---

# Data Sources

## NIC list

```hcl
data "procurator_nics" "all" {}

output "nics" {
  value = data.procurator_nics.all.nics
}
```

---

## Switch list

```hcl
data "procurator_switches" "all" {}

output "switches" {
  value = data.procurator_switches.all.switches
}
```

---

## Storage devices

```hcl
data "procurator_storage_devices" "all" {}

output "devices" {
  value = data.procurator_storage_devices.all.items
}
```

---

# Важные особенности

## 1. Datastore создаётся через core

Terraform НЕ вызывает storage напрямую.

Flow:

```
Terraform → core → storage → core → datastore создан
```

---

## 2. vg_name генерируется автоматически

* пользователь не задаёт vg_name
* storage генерирует его сам
* core сохраняет его как pool_name

---

## 3. Устройства передаются как имя

Правильно:

```hcl
devices = ["sdb"]
```

Допускается:

```hcl
devices = ["/dev/sdb"]
```

Provider нормализует:

```
/dev/sdb → sdb
```

---

## 4. NIC schema

* `nics` — это **object**, не block
* писать нужно через `=`

---

## 5. Empty list vs null

Provider учитывает различие:

* `[]` (пустой список)
* `null`

Это важно для корректной работы Terraform.

---

# Отладка

## Включить debug

```bash
TF_LOG=DEBUG terraform apply
```

---

## Проверка схемы provider

```bash
terraform providers schema -json | jq
```

---

# Типичные ошибки

## wipefs / pvcreate / vgcreate

```
wipefs -a exit status 1
pvcreate exit status 5
```

Причина:

* диск уже используется
* системный диск
* неверное устройство

Решение:

* использовать свободный диск
* проверить через:

```bash
lsblk
pvs
vgs
```

---

## Provider produced inconsistent result

Причина:

* null vs [] mismatch

Решение:

* уже исправлено в provider

---

# Roadmap

* datastore_nfs resource
* datastore detail data source
* network improvements
* vm lifecycle (clone/import)
* async task tracking улучшения

---

# Разработка

## Структура

```
internal/
  client/      → gRPC клиенты
  provider/    → Terraform ресурсы
```

---

## Сборка

```bash
go build ./...
```

---

## Локальное тестирование

```bash
terraform init
terraform apply
```

---

# Итог

Provider реализует:

* полный lifecycle datastore через core
* управление сетью через umbra
* работу с storage через backend
* корректную модель Terraform (без протечек внутреннего API)

---

# Автор

Procurator Terraform Provider
