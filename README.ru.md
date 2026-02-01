# refract

[![en](https://img.shields.io/badge/lang-English-blue.svg)](https://github.com/quyxishi/refract/blob/main/README.md)
[![ru](https://img.shields.io/badge/lang-Russian-blue.svg)](https://github.com/quyxishi/refract/blob/main/README.ru.md)

Refract - это сервис обеспечения соблюдения политик, который предоставляет контроль конкурентных сессий в реальном времени для Xray-core.

Он мониторит логи доступа, чтобы обнаруживать одновременные подключения с разных IP-адресов под одним и тем же идентификатором пользователя, и применяет ограничения, взаимодействуя напрямую с ядром Linux через Netlink и IPSet.

## Возможности

- **Применение политик в реальном времени**: Обнаруживает и блокирует несанкционированные конкурентные сессии за считанные миллисекунды.
- **Блокировка на уровне ядра**: Использует `ipset` и `iptables` для эффективного сброса пакетов на сетевом уровне.
- **Мгновенный разрыв сессий**: Использует `SOCK_DIAG` (netlink) для немедленного уничтожения существующих TCP-соединений заблокированных пользователей.
- **Zero-Copy парсинг логов**: Высокоэффективный механизм чтения хвоста лога, разработанный для сред с высокой пропускной способностью.
- **Docker Ready**: Полностью поддерживает `network_mode: host` при наличии прав `NET_ADMIN`.

## Архитектура

Refract работает как sidecar-контейнер для Xray-core. Он следит за access-логом, отслеживает состояние активных пользователей и при обнаружении нарушения выполняет следующие действия:
1. Добавляет IP-адрес нарушителя в `ipset` под названием `refract_banned_users` (с настраиваемым таймаутом).
2. Немедленно разрывает соответствующий TCP-сокет.
3. Гарантирует наличие правила в `iptables`, которое сбрасывает трафик от этого ipset'а.

## Использование

### Предварительные требования

- Права root (или Docker в режиме rootful с `CAP_NET_ADMIN`) необходимы для управления ipset, iptables и сокетами.
- Xray должен быть настроен на логирование реальных IP-адресов клиентов (через `proxy_protocol` или прямое подключение).
- Ядро Linux с поддержкой `ipset` и `netlink`.

### Конфигурация

Refract настраивается с помощью флагов командной строки:
```shell
./refract \
    --proto=tcp \
    --dport=443 \
    --window=5s \
    --timeout=1m \
    --access.log=/var/log/xray/access.log \
    --block.log=/var/log/xray/block.log
```

| Флаг | Описание | Значение по умолчанию |
| :--- | :--- | :--- |
| `proto` | Транспортный протокол (tcp/udp), используемый для фильтрации соединений при применении ограничений. | tcp |
| `dport` | Порт назначения соединений, которые необходимо отслеживать и блокировать. | 443 |
| `window` | Временное окно для проверки конкурентности. Допускает кратковременное перекрытие сессий при переключении сетей (например, с WiFi на LTE). | 5s |
| `timeout` | Длительность блокировки конфликтующего IP перед автоматическим снятием ограничения. | 1m |
| `access.log` | Путь к access-логу Xray. | /var/log/xray/access.log |
| `block.log` | Путь к логу аудита Refract (история действий по блокировке). | /var/log/xray/block.log |

### Запуск через Docker Compose

Для работы Refract требуется режим сети `host`, чтобы видеть реальные IP-адреса клиентов и управлять фаерволом хоста.

```yaml
services:
  refract:
    container_name: refract
    build: .
    restart: unless-stopped
    network_mode: host
    cap_add:
      - NET_ADMIN
    volumes:
      - /var/log/xray/access.log:/var/log/xray/access.log:ro
      - /var/log/xray/block.log:/var/log/xray/block.log
```

### Запуск из бинарного файла

Если вы предпочитаете запустить Refract как systemd-сервис без Docker, выполните следующие шаги.

1. Установите необходимые зависимости:
```bash
# Для Debian/Ubuntu
sudo apt install -y ipset iptables iproute2
# Для CentOS/RHEL/Fedora
sudo dnf install -y ipset iptables iproute
# Для Alpine
apk add ipset iptables iproute2
```

2. Либо клонируйте репозиторий и соберите проект (через `make build`), либо скачайте последний релиз для вашей архитектуры со [страницы Releases](https://github.com/quyxishi/refract/releases/latest).
3. Распакуйте бинарный файл в системную директорию и сделайте его исполняемым:
```shell
chmod +x /opt/refract/refract
```

4. Создайте файл конфигурации для systemd-сервиса:
```shell
mkdir -p /etc/default
cat <<EOF > /etc/default/refract
REFRACT_PROTO="tcp"
REFRACT_DPORT="443"
REFRACT_WINDOW="5s"
REFRACT_TIMEOUT="1m"
REFRACT_ACCESS_LOG="/var/log/xray/access.log"
REFRACT_BLOCK_LOG="/var/log/xray/block.log"
EOF
```

5. Скопируйте [файл systemd-сервиса](/refract.service) в `/etc/systemd/system/refract.service` и запустите службу:
```shell
systemctl daemon-reload
systemctl enable --now refract
systemctl status refract
```

***

> [!WARNING]
> **При работе за обратным прокси** (Nginx/HAProxy), убедитесь, что реальный IP клиента доходит до Xray через PROXY protocol; в противном случае вы рискуете заблокировать localhost или IP вашего сервера вместо реального нарушителя.

### Настройка обратного прокси (Reverse Proxy)

###### Пример конфигурации Xray:
```json
{
  "log": {
    "error": "/var/log/xray/error.log",
    "access": "/var/log/xray/access.log",
    "loglevel": "warning"
  },
  "inbounds": [
    {
      "tag": "NIDX00-INBOUND-IDX00",
      "port": 14443,
      "listen": "127.0.0.1",
      "protocol": "vless",
      "sniffing": {
        "enabled": true,
        "destOverride": [
          "http",
          "tls",
          "quic"
        ]
      },
      "streamSettings": {
        "network": "xhttp",
        "security": "reality",
        "sockopt": {
          "acceptProxyProtocol": true // Принимаем PROXY v1/v2 от обратной прокси
        }
      }
    }
  ]
}
```

###### Пример конфигурации Nginx:
```nginx
stream {
    # Карта для маршрутизации по SNI, апстримам и т.д.

    server {
        listen 443;
        listen [::]:443;

        ssl_preread on;

        # Отправлять PROXY protocol на бэкенд
        proxy_protocol on;
        # Xray inbound
        proxy_pass $backend_name;
    }
}
```

###### Пример конфигурации HAProxy:
```haproxy
backend xray_backend
    mode tcp
    server xray1 127.0.0.1:14443 send-proxy-v2
```

Это гарантирует, что Xray получит реальный IP-адрес клиента в своих access-логах, позволяя Refract блокировать правильные адреса.

## Известные проблемы

### Сохранение таймаута в IPSet

Refract использует `ipset` (с именем `refract_banned_users`) для отслеживания заблокированных IP.
Если вы измените значение `--timeout` после того, как сервис уже создал этот сет, **новый таймаут может не примениться сразу**.

Это происходит потому, что команда `ipset create` не перезаписывает параметры уже существующего сета. Ядро сохраняет значение таймаута, заданное при первоначальном создании сета.

**Решение:**
Если вы меняете конфигурацию таймаута, необходимо вручную удалить существующий сет, чтобы изменения вступили в силу:

```shell
# Найти номер правила Refract
sudo iptables -L INPUT --line-numbers | grep refract

# Удалить правило, замените $X на номер правила
sudo iptables -D INPUT $X

# Очистить и уничтожить старый сет
sudo ipset destroy refract_banned_users
```

## Разработка

###### Сборка
```shell
make build
```

###### Запуск
```shell
make run
```

## Лицензия

MIT License, см. [LICENSE](https://github.com/quyxishi/refract/blob/main/LICENSE).
