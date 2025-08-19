# Платформа для размещения медиа-файлов Ehworld

***
## Как собрать
Для того, чтобы проделать следующие шаги на Windows, установите [Git Bash](https://gitforwindows.org/) и [Golang](https://go.dev/doc/install)

1. Склонируйте репозиторий

```shell
git clone https://github.com/detoxique/ehworld.git
```

2. Соберите приложение

```shell
go build -o app.exe
```

3. Заполните .env файл своими данными из [Twitch Dev Console](https://dev.twitch.tv/console)
   
4. Создайте БД в postgresql, используя файл sql.txt
