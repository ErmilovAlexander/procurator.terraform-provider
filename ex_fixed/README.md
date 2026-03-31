# Terraform examples: порядок проверки

Во всех каталогах запускать `terraform init` отдельно.


## 1. Создание VM

```bash
cd 01_create_vm
rm -rf .terraform terraform.tfstate terraform.tfstate.backup
terraform init
terraform apply
```

Проверить в UI:
- VM `tf-create-vm-01` создана
- datastore `DEV-STOR-0`
- firmware `efi`
- NIC `VLAN106`, model `virtio`
- disk 30 GB
- state `stopped`

Сохранить `vm_id` из output. Он понадобится для import.

## 2. Power on / power off

Этот сценарий работает через import существующей VM в отдельный state.

```bash
cd ../02_power_cycle
rm -rf .terraform terraform.tfstate terraform.tfstate.backup
terraform init
terraform import procurator_vm.power_test <vm_id_из_шага_1>
terraform apply -var-file=01-on.tfvars
terraform apply -var-file=02-off.tfvars
```

Проверить в UI:
- после `01-on.tfvars` VM `running`
- после `02-off.tfvars` VM `stopped`

## 3. Update CPU / RAM

Тоже через import той же VM.

```bash
cd ../03_update_cpu_ram
rm -rf .terraform terraform.tfstate terraform.tfstate.backup
terraform init
terraform import procurator_vm.resize_test <vm_id_из_шага_1>
terraform apply -var-file=01-small.tfvars
terraform apply -var-file=02-big.tfvars
```

Проверить в UI:
- после `01-small.tfvars`: 2 vCPU, 4096 MB
- после `02-big.tfvars`: 4 vCPU, 8192 MB

## 4. Convert VM -> Template

Перед запуском убедиться, что VM `tf-create-vm-01` выключена.

```bash
cd ../04_convert_vm_to_template
rm -rf .terraform terraform.tfstate terraform.tfstate.backup
terraform init
terraform apply
```

Проверить:
- output `template_id`, `template_uuid`
- объект `tf-create-vm-01` стал template

## 5. Deploy from template

```bash
cd ../05_deploy_from_template
rm -rf .terraform terraform.tfstate terraform.tfstate.backup
terraform init
terraform apply
```

Проверить в UI:
- создана VM `tf-from-template-01`
- firmware унаследован от template
- сеть унаследована от template
- VM в состоянии `stopped`

## 6. Полный цикл в одном каталоге

Этот сценарий независим от предыдущих шагов.

```bash
cd ../06_full_cycle_create_to_template_to_vm
rm -rf .terraform terraform.tfstate terraform.tfstate.backup
terraform init
terraform apply
```

Проверить:
- создана `tf-full-cycle-source`
- затем она конвертирована в template
- создана `tf-full-cycle-deployed`

## Важные замечания

- Каталоги `02_power_cycle` и `03_update_cpu_ram` специально сделаны под `terraform import`.
- Не запускать `02` и `03` после шага 4, если `tf-create-vm-01` уже превращена в template.
- Каждый каталог хранит свой отдельный state.


## 7. Snapshots

Перед запуском должна существовать VM `tf-create-vm-01` (из шага 1 или своя с таким именем).

```bash
cd ../07_snapshots
rm -rf .terraform terraform.tfstate terraform.tfstate.backup
terraform init
terraform apply
```

Проверить в UI:
- у VM `tf-create-vm-01` появился snapshot `tf-snap-01`
- в output есть `snapshot_numeric_id`

Проверка удаления snapshot:

```bash
terraform destroy
```

Проверить в UI:
- snapshot `tf-snap-01` удалён

## 8. Attach / detach disk

Перед запуском должна существовать VM `tf-create-vm-01`.

```bash
cd ../08_disk_attach_detach
rm -rf .terraform terraform.tfstate terraform.tfstate.backup
terraform init
terraform apply
```

Проверить в UI:
- у VM появился дополнительный диск `vdb`
- размер 5 GB

Проверка detach/remove:

```bash
terraform destroy
```

Проверить в UI:
- диск `vdb` исчез

## 9. Attach / detach network

Перед запуском должна существовать VM `tf-create-vm-01`.

```bash
cd ../09_network_attach_detach
rm -rf .terraform terraform.tfstate terraform.tfstate.backup
terraform init
terraform apply
```

Проверить в UI:
- у VM появился NIC `eth1`
- сеть `VLAN106`

Проверка detach:

```bash
terraform destroy
```

Проверить в UI:
- NIC `eth1` удалён
