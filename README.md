# subscribe-service

### Методы gRPC
#### Метод ```Subscribe```
- Клиент подключается ```SubscribeRequest { "key": "notifications" } ```
- Сервер в отдельной горутине отправляет клиенту сообщения через ```stream.Send()```
- При отключении отписывает клиента от ```notifications```

#### Метод ```Publish```
- Клиент отправляет запрос ```PublishRequest { "key": "notifications", data:"Hi" } ```
- Сервер рассылает всем подписчикам ```notifications```
- Возвращает ```Empty{}``` или ошибку


### Компоненты
- gRPC сервер принимает подключения, маршрутизирует вызовы Publish/Subscribe
- PubSubService управляет подписками и сообщениями с помощью subpub

### Быстрый запуск:
```
go run cmd/server/main.go --config=./config/local.yaml
```

### Запуск через docker:
Собрать докер образ
```
docker build -t mm-pubsub-server .
```
Запустить, указав путь до конфига (внутри контейнера) и порты   
Директория с конфигом монтируется, так что конфиг берется "из вне", и не копируется в контейнер при сборке.
```
docker run -p 50051:50051 -e CONFIG_PATH=/app/config/local.yaml -v ${pwd}/config:/app/config mm-pubsub-server
```
### Что используется:

 - Реализован **graceful shutdown**
 - Логирование с помощью **slog**
 - Конфиг: **/config/local.yaml**, парсится с использованием **github.com/ilyakaznacheev/cleanenv**   
Конфиг по умолчанию:
```
env: "local"
grpc:
  host: "0.0.0.0"
  port: 50051
  timeout: 30s
```
