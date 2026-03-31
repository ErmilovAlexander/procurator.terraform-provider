# Procurator Terraform Provider — Resources

Полное описание всех ресурсов Terraform provider для Procurator.

---

# Общие правила

## Типы полей

* **Required** — обязательное поле
* **Optional** — необязательное
* **Computed** — возвращается API, задавать нельзя

---

# procurator_datastore_lvm

Создание LVM datastore через **core gRPC**.

## Описание

* НЕ вызывает storage напрямую
* core сам вызывает storage
* `vg_name` генерируется автоматически

## Аргументы

| Поле    | Тип          | Описание         |
| ------- | ------------ | ---------------- |
| name    | string       | имя datastore    |
| devices | list(string) | список устройств |

## Атрибуты

| Поле              | Тип    | Описание                 |
| ----------------- | ------ | ------------------------ |
| id                | string | ID datastore             |
| pool_name         | string | внутреннее имя (vg_name) |
| state             | string | состояние                |
| status            | string | статус                   |
| capacity_mb       | number | объём                    |
| free_mb           | number | свободно                 |
| used_mb           | number | занято                   |
| provisioned_mb    | number | выделено                 |
| thin_provisioning | bool   | thin provisioning        |
| access_mode       | string | режим доступа            |

## Пример

```hcl
resource "procurator_datastore_lvm" "data1" {
  name = "data1"

  devices = ["sdb"]
}
```

---

# procurator_datastore

Универсальный datastore resource.

## Аргументы

| Поле      | Тип          | Описание      |
| --------- | ------------ | ------------- |
| name      | string       | имя           |
| type_code | number       | тип datastore |
| devices   | list(string) | устройства    |
| server    | string       | NFS server    |
| folder    | string       | NFS path      |

## type_code

| Код | Тип    |
| --- | ------ |
| 1   | Local  |
| 2   | LVM    |
| 3   | dStorm |
| 4   | NFS    |

## Пример

```hcl
resource "procurator_datastore" "data1" {
  name      = "data1"
  type_code = 2

  devices = ["sdb"]
}
```

---

# procurator_datastore_folder

Создание папки внутри datastore.

## Аргументы

| Поле | Тип    | Описание                    |
| ---- | ------ | --------------------------- |
| path | string | путь `<datastore_id>:/path` |

## Пример

```hcl
resource "procurator_datastore_folder" "images" {
  path = "datastore_id:/images"
}
```

---

# procurator_switch

Коммутатор Umbra.

## Аргументы

| Поле | Тип    | Описание      |
| ---- | ------ | ------------- |
| mtu  | number | MTU           |
| nics | object | настройки NIC |

## nics

| Поле    | Тип          |
| ------- | ------------ |
| active  | list(string) |
| standby | list(string) |
| unused  | list(string) |
| inherit | bool         |

## Атрибуты

| Поле     | Тип          |
| -------- | ------------ |
| id       | string       |
| state    | string       |
| errors   | list(string) |
| networks | list(string) |

## Пример

```hcl
resource "procurator_switch" "main" {
  mtu = 9000

  nics = {
    active  = ["enp1s0"]
    standby = []
    unused  = []
    inherit = false
  }
}
```

---

# procurator_network

Сеть Umbra.

## Аргументы

| Поле      | Тип    |
| --------- | ------ |
| name      | string |
| vlan      | number |
| switch_id | string |

## Атрибуты

| Поле         | Тип          |
| ------------ | ------------ |
| id           | string       |
| state        | string       |
| kind         | string       |
| vms_count    | number       |
| active_ports | number       |
| net_bridge   | string       |
| errors       | list(string) |

## Пример

```hcl
resource "procurator_network" "net1" {
  name      = "prod-net"
  vlan      = 120
  switch_id = "switch-id"
}
```

---

# procurator_vm

Виртуальная машина.

## Аргументы

| Поле | Тип    |
| ---- | ------ |
| name | string |
| cpu  | number |
| ram  | number |

## Атрибуты

| Поле  | Тип    |
| ----- | ------ |
| id    | string |
| state | string |

---

# procurator_template

Шаблон VM.

## Аргументы

| Поле | Тип    |
| ---- | ------ |
| name | string |

---

# procurator_vm_snapshot

Snapshot VM.

## Аргументы

| Поле  | Тип    |
| ----- | ------ |
| vm_id | string |
| name  | string |

---

# procurator_vm_disk_attachment

Подключение диска.

## Аргументы

| Поле         | Тип    |
| ------------ | ------ |
| vm_id        | string |
| datastore_id | string |
| size         | number |

---

# procurator_vm_network_attachment

Подключение сети к VM.

## Аргументы

| Поле       | Тип    |
| ---------- | ------ |
| vm_id      | string |
| network_id | string |

---

# Data Sources

---

# procurator_nics

Список NIC.

```hcl
data "procurator_nics" "all" {}
```

---

# procurator_switches

Список switch.

```hcl
data "procurator_switches" "all" {}
```

---

# procurator_networks

Список сетей.

```hcl
data "procurator_networks" "all" {}
```

---

# procurator_storage_devices

Список дисков.

```hcl
data "procurator_storage_devices" "all" {}
```

---

# procurator_storage_adapters

Список storage адаптеров.

```hcl
data "procurator_storage_adapters" "all" {}
```

---

# Итог

Provider покрывает:

* VM lifecycle
* datastore lifecycle через core
* сети через umbra
* storage через backend
* полную интеграцию всех слоёв Procurator
